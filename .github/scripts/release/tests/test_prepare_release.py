import base64
import importlib.util
import json
import os
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock


MODULE_PATH = Path(__file__).resolve().parents[1] / "prepare_release.py"
spec = importlib.util.spec_from_file_location("prepare_release", MODULE_PATH)
assert spec is not None
assert spec.loader is not None
prepare_release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(prepare_release)


def git(repo: Path, *args: str) -> str:
    return subprocess.check_output(["git", *args], cwd=repo, text=True).strip()


def commit(repo: Path, subject: str, body: str = "") -> str:
    marker = repo / "history.txt"
    marker.write_text(marker.read_text() + subject + "\n" if marker.exists() else subject + "\n")
    git(repo, "add", "history.txt")
    message = subject if not body else f"{subject}\n\n{body}"
    git(repo, "commit", "-m", message)
    return git(repo, "rev-parse", "HEAD")


def published(
    tag: str,
    target: str,
    prerelease: bool | None = None,
    published_at: str = "2020-01-01T00:00:00Z",
) -> dict[str, object]:
    return {
        "tag_name": tag,
        "target_commitish": target,
        "draft": False,
        "prerelease": "-" in tag if prerelease is None else prerelease,
        "published_at": published_at,
    }


class ApiResponse:
    status = 200
    headers: dict[str, str] = {}

    def __init__(self, value: object) -> None:
        self.body = json.dumps(value).encode()
        self.read_sizes: list[int] = []

    def __enter__(self):
        return self

    def __exit__(self, *_args: object) -> None:
        return None

    def read(self, size: int = -1) -> bytes:
        self.read_sizes.append(size)
        return self.body[:size]


class ApiOpener:
    def __init__(self, responses: list[ApiResponse]) -> None:
        self.responses = list(responses)
        self.requests = []

    def open(self, request, timeout: int):  # noqa: ANN001
        self.requests.append(request)
        return self.responses.pop(0)


class PrepareReleaseTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.repo = Path(self.temp.name)
        git(self.repo, "init", "-b", "main")
        git(self.repo, "config", "user.name", "Release Test")
        git(self.repo, "config", "user.email", "release-test@example.invalid")
        publish = self.repo / ".github" / "scripts" / "release" / "publish_release.py"
        publish.parent.mkdir(parents=True)
        publish.write_text("print('publish')\n")
        git(self.repo, "add", str(publish.relative_to(self.repo)))
        git(self.repo, "commit", "-m", "chore: bootstrap")
        git(self.repo, "tag", "v1.2.3")

    def tearDown(self) -> None:
        self.temp.cleanup()

    def test_fetches_complete_published_release_history_with_bounded_pages(self) -> None:
        pages = [
            ApiResponse([{"id": value} for value in range(100)]),
            ApiResponse([{"id": 100}]),
        ]
        opener = ApiOpener(pages)
        with mock.patch.object(
            prepare_release.urllib.request, "build_opener", return_value=opener
        ):
            releases = prepare_release.fetch_releases(
                "https://api.github.com", "dummy-token", "Verity-BDD/verity-bdd"
            )

        self.assertEqual(len(releases), 101)
        self.assertEqual(
            [request.full_url for request in opener.requests],
            [
                "https://api.github.com/repos/Verity-BDD/verity-bdd/releases?per_page=100&page=1",
                "https://api.github.com/repos/Verity-BDD/verity-bdd/releases?per_page=100&page=2",
            ],
        )
        self.assertTrue(
            all(
                response.read_sizes == [prepare_release.MAX_RESPONSE_BYTES + 1]
                for response in pages
            )
        )

    def test_release_history_fails_closed_at_pagination_limit(self) -> None:
        page = ApiResponse([{"id": value} for value in range(100)])
        opener = ApiOpener([page])
        with (
            mock.patch.object(prepare_release, "MAX_RELEASE_PAGES", 1),
            mock.patch.object(
                prepare_release.urllib.request, "build_opener", return_value=opener
            ),
        ):
            with self.assertRaisesRegex(prepare_release.PrepareError, "pagination limit"):
                prepare_release.fetch_releases(
                    "https://api.github.com", "dummy-token", "Verity-BDD/verity-bdd"
                )

    def test_cli_fetches_history_and_propagates_version_and_sha(self) -> None:
        metadata = prepare_release.ReleaseMetadata("a" * 40, "v2.0.0-rc.1", "notes", "b" * 64)
        history = [published("v1.9.0", "main")]
        environment = {
            "ACTIONS_TOKEN": "dummy-token",
            "API_URL": "https://api.github.com",
            "GITHUB_OUTPUT": "/tmp/not-used",
            "LIBRARY_SHA": "a" * 40,
            "REPOSITORY": "Verity-BDD/verity-bdd",
            "VERSION": "v2.0.0-rc.1",
        }
        with (
            mock.patch.dict(os.environ, environment, clear=True),
            mock.patch.object(prepare_release, "fetch_releases", return_value=history) as fetch,
            mock.patch.object(prepare_release, "build_release", return_value=metadata) as build,
            mock.patch.object(prepare_release, "_write_outputs") as write,
        ):
            self.assertEqual(prepare_release.main(), 0)

        fetch.assert_called_once_with(
            "https://api.github.com", "dummy-token", "Verity-BDD/verity-bdd"
        )
        build.assert_called_once_with(Path.cwd(), "a" * 40, "v2.0.0-rc.1", history)
        write.assert_called_once_with(metadata)

    def test_write_outputs_emits_exact_final_and_prerelease_metadata(self) -> None:
        cases = (
            prepare_release.ReleaseMetadata(
                library_sha="a" * 40,
                version="v1.3.0",
                body="## Changes\n\n- final release\n\n## Provenance\n\nLibrary commit: `" + "a" * 40 + "`\n",
                publish_script_sha256="b" * 64,
                release_type="final",
                predecessor="v1.2.3",
                changelog_range="refs/tags/v1.2.3.." + "a" * 40,
            ),
            prepare_release.ReleaseMetadata(
                library_sha="c" * 40,
                version="v1.3.0-rc.2",
                body="## Changes\n\n- prerelease\n\n## Provenance\n\nLibrary commit: `" + "c" * 40 + "`\n",
                publish_script_sha256="d" * 64,
                release_type="prerelease",
                predecessor="v1.3.0-rc.1",
                changelog_range="refs/tags/v1.3.0-rc.1.." + "c" * 40,
            ),
        )

        for metadata in cases:
            with self.subTest(version=metadata.version):
                output = self.repo / f"{metadata.version}.output"
                with mock.patch.dict(os.environ, {"GITHUB_OUTPUT": str(output)}, clear=True):
                    prepare_release._write_outputs(metadata)

                actual = dict(line.split("=", 1) for line in output.read_text().splitlines())
                self.assertEqual(
                    actual,
                    {
                        "version": metadata.version,
                        "release_type": metadata.release_type,
                        "predecessor": metadata.predecessor,
                        "changelog_range": metadata.changelog_range,
                        "release_body_b64": base64.b64encode(metadata.body.encode()).decode("ascii"),
                        "publish_script_sha256": metadata.publish_script_sha256,
                    },
                )
                self.assertEqual(
                    base64.b64decode(actual["release_body_b64"]).decode(), metadata.body
                )

    def test_prepares_requested_final_version_with_deterministic_notes(self) -> None:
        target = commit(self.repo, "feat(api): add exact provenance")
        commit(self.repo, "fix: later commit must not enter selected release")
        published = [
            {
                "tag_name": "v1.2.3",
                "target_commitish": git(self.repo, "rev-parse", "v1.2.3^{commit}"),
                "draft": False,
                "prerelease": False,
                "published_at": "2020-01-01T00:00:00Z",
            }
        ]

        first = prepare_release.build_release(self.repo, target, "v1.3.0", published)
        second = prepare_release.build_release(self.repo, target, "v1.3.0", published)

        self.assertEqual(first, second)
        self.assertEqual(first.library_sha, target)
        self.assertEqual(first.version, "v1.3.0")
        self.assertIn("feat(api): add exact provenance", first.body)
        self.assertNotIn("later commit", first.body)
        self.assertIn(target, first.body)
        self.assertRegex(first.publish_script_sha256, r"^[0-9a-f]{64}$")

    def test_published_release_tag_is_authoritative_when_target_commitish_is_a_ref(self) -> None:
        previous = git(self.repo, "rev-parse", "v1.2.3^{commit}")
        target = commit(self.repo, "fix: candidate")

        metadata = prepare_release.build_release(
            self.repo,
            target,
            "v1.2.4",
            [published("v1.2.3", "main")],
        )

        self.assertEqual(metadata.version, "v1.2.4")
        self.assertNotEqual(previous, target)

    def test_semver_precedence_orders_prereleases_before_final(self) -> None:
        ordered = [
            "v1.0.0-rc.2",
            "v1.0.0",
            "v0.22.3",
            "v1.0.0-rc.1",
        ]
        self.assertEqual(
            sorted(ordered, key=prepare_release.parse_version),
            ["v0.22.3", "v1.0.0-rc.1", "v1.0.0-rc.2", "v1.0.0"],
        )

    def test_rejects_malformed_versions_build_metadata_and_numeric_leading_zeroes(self) -> None:
        invalid = (
            "1.0.0",
            "v01.0.0",
            "v1.0",
            "v1.0.0+build.1",
            "v1.0.0-rc.01",
            "v1.0.0-",
            "v1.0.0-rc_1",
        )
        for version in invalid:
            with self.subTest(version=version):
                with self.assertRaises(prepare_release.PrepareError):
                    prepare_release.parse_version(version)

    def test_same_sha_prerelease_promotion_uses_semver_order_and_explicit_notes(self) -> None:
        target = git(self.repo, "rev-parse", "HEAD")
        git(self.repo, "tag", "v1.3.0-rc.1", target)
        releases = [
            published("v1.2.3", target),
            published(
                "v1.3.0-rc.1", target, published_at="2020-01-02T00:00:00Z"
            ),
        ]

        metadata = prepare_release.build_release(
            self.repo, target, "v1.3.0-rc.2", releases
        )

        self.assertEqual(metadata.version, "v1.3.0-rc.2")
        self.assertIn("promotes v1.3.0-rc.1", metadata.body)
        self.assertIn("same tested commit", metadata.body)
        self.assertIn(target, metadata.body)

    def test_exact_highest_final_rerun_reconstructs_identical_metadata(self) -> None:
        previous = git(self.repo, "rev-parse", "v1.2.3^{commit}")
        target = commit(self.repo, "fix: candidate")
        predecessor_history = [published("v1.2.3", previous)]
        initial = prepare_release.build_release(
            self.repo, target, "v1.2.4", predecessor_history
        )
        git(self.repo, "tag", "v1.2.4", target)

        rerun = prepare_release.build_release(
            self.repo,
            target,
            "v1.2.4",
            [
                published("v1.2.3", previous),
                published(
                    "v1.2.4", target, published_at="2020-01-02T00:00:00Z"
                ),
            ],
        )

        self.assertEqual(rerun, initial)
        self.assertEqual(rerun.body, initial.body)

    def test_exact_first_release_rerun_reconstructs_identical_metadata(self) -> None:
        target = git(self.repo, "rev-parse", "HEAD")
        initial = prepare_release.build_release(self.repo, target, "v1.2.3", [])

        rerun = prepare_release.build_release(
            self.repo, target, "v1.2.3", [published("v1.2.3", target)]
        )

        self.assertEqual(rerun, initial)

    def test_rejects_exact_older_release_when_a_higher_release_exists(self) -> None:
        previous = git(self.repo, "rev-parse", "v1.2.3^{commit}")
        target = commit(self.repo, "fix: candidate")
        git(self.repo, "tag", "v1.2.4", target)

        with self.assertRaisesRegex(prepare_release.PrepareError, "must exceed"):
            prepare_release.build_release(
                self.repo,
                previous,
                "v1.2.3",
                [
                    published("v1.2.3", previous),
                    published(
                        "v1.2.4", target, published_at="2020-01-02T00:00:00Z"
                    ),
                ],
            )

    def test_rejects_same_sha_promotion_from_a_final_release(self) -> None:
        target = git(self.repo, "rev-parse", "HEAD")
        with self.assertRaisesRegex(prepare_release.PrepareError, "latest release is final"):
            prepare_release.build_release(
                self.repo, target, "v1.2.4", [published("v1.2.3", target)]
            )

    def test_rejects_new_core_version_on_same_sha_after_prerelease(self) -> None:
        target = git(self.repo, "rev-parse", "HEAD")
        git(self.repo, "tag", "v1.3.0-rc.1", target)
        releases = [
            published("v1.2.3", target),
            published(
                "v1.3.0-rc.1", target, published_at="2020-01-02T00:00:00Z"
            ),
        ]
        with self.assertRaisesRegex(prepare_release.PrepareError, "same core version"):
            prepare_release.build_release(self.repo, target, "v1.4.0-rc.1", releases)

    def test_rejects_reuse_of_older_release_sha_after_history_advanced(self) -> None:
        older_sha = git(self.repo, "rev-parse", "v1.2.3^{commit}")
        newer_sha = commit(self.repo, "fix: newer release")
        git(self.repo, "tag", "v1.2.4", newer_sha)
        history = [
            published("v1.2.3", older_sha),
            published("v1.2.4", newer_sha, published_at="2020-01-02T00:00:00Z"),
        ]

        with self.assertRaisesRegex(prepare_release.PrepareError, "already published"):
            prepare_release.build_release(self.repo, older_sha, "v1.2.5", history)

    def test_rejects_release_history_without_explicit_draft_state(self) -> None:
        target = commit(self.repo, "fix: candidate")
        previous = git(self.repo, "rev-parse", "v1.2.3^{commit}")
        malformed = published("v1.2.3", previous)
        del malformed["draft"]

        with self.assertRaisesRegex(prepare_release.PrepareError, "malformed"):
            prepare_release.build_release(self.repo, target, "v1.2.4", [malformed])

    def test_rejects_published_release_without_matching_local_tag(self) -> None:
        target = commit(self.repo, "fix: candidate")
        history = [published("v1.2.4", "main")]

        with self.assertRaisesRegex(prepare_release.PrepareError, "matching tag"):
            prepare_release.build_release(self.repo, target, "v1.2.5", history)

    def test_rejects_downgrade_and_prerelease_below_published_final(self) -> None:
        previous = git(self.repo, "rev-parse", "v1.2.3^{commit}")
        target = commit(self.repo, "fix: candidate")
        history = [published("v1.2.3", previous)]

        for requested in ("v1.2.2", "v1.2.3-rc.2"):
            with self.subTest(requested=requested):
                with self.assertRaisesRegex(prepare_release.PrepareError, "must exceed"):
                    prepare_release.build_release(self.repo, target, requested, history)

    def test_rejects_semver_regression_by_publication_chronology(self) -> None:
        released_sha = git(self.repo, "rev-parse", "v1.2.3^{commit}")
        git(self.repo, "tag", "v1.1.0", released_sha)
        git(self.repo, "tag", "v1.2.0", released_sha)
        target = commit(self.repo, "fix: candidate")
        history = [
            published(
                "v1.1.0", released_sha, published_at="2020-01-02T00:00:00Z"
            ),
            published(
                "v1.2.3", released_sha, published_at="2020-01-03T00:00:00Z"
            ),
            published(
                "v1.2.0", released_sha, published_at="2020-01-01T00:00:00Z"
            ),
        ]

        with self.assertRaisesRegex(prepare_release.PrepareError, "publication chronology"):
            prepare_release.build_release(self.repo, target, "v1.3.0", history)

    def test_rejects_malformed_or_duplicate_publication_timestamps(self) -> None:
        released_sha = git(self.repo, "rev-parse", "v1.2.3^{commit}")
        git(self.repo, "tag", "v1.2.4", released_sha)
        target = commit(self.repo, "fix: candidate")
        cases = {
            "missing": [
                {key: value for key, value in published("v1.2.3", released_sha).items() if key != "published_at"}
            ],
            "malformed": [
                published("v1.2.3", released_sha, published_at="not-a-timestamp")
            ],
            "duplicate": [
                published("v1.2.3", released_sha),
                published("v1.2.4", released_sha),
            ],
        }

        for name, history in cases.items():
            with self.subTest(name=name):
                with self.assertRaisesRegex(
                    prepare_release.PrepareError, "published_at|publication timestamps"
                ):
                    prepare_release.build_release(
                        self.repo, target, "v1.3.0", history
                    )

    def test_rejects_duplicate_published_release_tags(self) -> None:
        previous = git(self.repo, "rev-parse", "v1.2.3^{commit}")
        target = commit(self.repo, "fix: candidate")
        history = [published("v1.2.3", previous), published("v1.2.3", "main")]

        with self.assertRaisesRegex(prepare_release.PrepareError, "duplicate"):
            prepare_release.build_release(self.repo, target, "v1.2.4", history)

    def test_orphan_semver_tag_blocks_release(self) -> None:
        target = commit(self.repo, "fix: candidate")
        git(self.repo, "tag", "v1.2.4", target)
        previous = git(self.repo, "rev-parse", "v1.2.3^{commit}")

        with self.assertRaisesRegex(prepare_release.PrepareError, "orphan"):
            prepare_release.build_release(
                self.repo, target, "v1.2.5", [published("v1.2.3", previous)]
            )

    def test_build_metadata_semver_tag_blocks_as_orphan(self) -> None:
        target = commit(self.repo, "fix: candidate")
        git(self.repo, "tag", "v1.0.0+build.1", target)
        previous = git(self.repo, "rev-parse", "v1.2.3^{commit}")

        with self.assertRaisesRegex(prepare_release.PrepareError, "orphan"):
            prepare_release.build_release(
                self.repo, target, "v1.2.4", [published("v1.2.3", previous)]
            )

    def test_arbitrary_malformed_v_tag_is_not_release_state(self) -> None:
        target = commit(self.repo, "fix: candidate")
        git(self.repo, "tag", "vfoo", target)
        previous = git(self.repo, "rev-parse", "v1.2.3^{commit}")

        metadata = prepare_release.build_release(
            self.repo, target, "v1.2.4", [published("v1.2.3", previous)]
        )

        self.assertEqual(metadata.version, "v1.2.4")

    def test_exact_requested_orphan_tag_on_candidate_is_partial_retry(self) -> None:
        target = commit(self.repo, "fix: candidate")
        git(self.repo, "tag", "v1.2.4", target)
        previous = git(self.repo, "rev-parse", "v1.2.3^{commit}")

        metadata = prepare_release.build_release(
            self.repo, target, "v1.2.4", [published("v1.2.3", previous)]
        )

        self.assertEqual(metadata.version, "v1.2.4")
        self.assertIn("fix: candidate", metadata.body)

    def test_rejects_non_commit_or_non_exact_sha(self) -> None:
        for candidate in ("main", "A" * 40, "a" * 39):
            with self.subTest(candidate=candidate):
                with self.assertRaises(prepare_release.PrepareError):
                    prepare_release.build_release(
                        self.repo,
                        candidate,
                        "v1.2.4",
                        [
                            published(
                                "v1.2.3",
                                git(self.repo, "rev-parse", "v1.2.3^{commit}"),
                            )
                        ],
                    )


if __name__ == "__main__":
    unittest.main()
