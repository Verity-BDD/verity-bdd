import importlib.util
import io
import json
import unittest
import urllib.error
from pathlib import Path
from unittest import mock


MODULE_PATH = Path(__file__).resolve().parents[1] / "verify_provenance.py"
spec = importlib.util.spec_from_file_location("verify_provenance", MODULE_PATH)
assert spec is not None
assert spec.loader is not None
verify_provenance = importlib.util.module_from_spec(spec)
spec.loader.exec_module(verify_provenance)


SHA = "a" * 40
REPOSITORY = "Verity-BDD/verity-bdd"
DEFAULT_BRANCH = "main"


class InstrumentedResponse:
    def __init__(
        self,
        body: bytes,
        *,
        status: object = 200,
        content_length: str | None = None,
    ) -> None:
        self.body = body
        self.status = status
        self.headers = {}
        if content_length is not None:
            self.headers["Content-Length"] = content_length
        self.read_sizes: list[int] = []

    def __enter__(self):
        return self

    def __exit__(self, *_args: object) -> None:
        return None

    def read(self, size: int = -1) -> bytes:
        self.read_sizes.append(size)
        if size < 0:
            raise AssertionError("unbounded read")
        return self.body[:size]


class FakeOpener:
    def __init__(self, response: object) -> None:
        self.response = response
        self.requests = []

    def open(self, request, timeout: int):  # noqa: ANN001
        self.requests.append(request)
        if isinstance(self.response, Exception):
            raise self.response
        return self.response


def fetch_with(response: object):
    opener = FakeOpener(response)
    with (
        mock.patch.object(
            verify_provenance.urllib.request, "build_opener", return_value=opener
        ),
        mock.patch.object(
            verify_provenance.urllib.request,
            "urlopen",
            side_effect=AssertionError("default redirect-capable opener used"),
        ),
    ):
        result = verify_provenance.fetch_manual_runs(
            "https://api.github.com", "dummy-token", REPOSITORY, DEFAULT_BRANCH, SHA
        )
    return result, opener


def valid_workflow_run() -> dict:
    return {
        "id": 123,
        "name": "CI",
        "path": ".github/workflows/ci.yml",
        "event": "push",
        "head_branch": DEFAULT_BRANCH,
        "head_sha": SHA,
        "status": "completed",
        "conclusion": "success",
        "head_repository": {"full_name": REPOSITORY},
        "repository": {"full_name": REPOSITORY},
    }


class ManualProvenanceTest(unittest.TestCase):
    def test_accepts_only_current_main_head_with_one_successful_ci_run(self) -> None:
        payload = {"total_count": 1, "workflow_runs": [valid_workflow_run()]}
        head = {"ref": "refs/heads/main", "object": {"type": "commit", "sha": SHA}}
        self.assertEqual(
            verify_provenance.validate_manual_candidate(
                head, payload, REPOSITORY, DEFAULT_BRANCH, SHA
            ),
            SHA,
        )
        for branch, candidate_head in (
            ("release", head),
            ("main", {**head, "object": {"type": "commit", "sha": "b" * 40}}),
        ):
            with self.subTest(branch=branch, head=candidate_head):
                with self.assertRaises(verify_provenance.ProvenanceError):
                    verify_provenance.validate_manual_candidate(
                        candidate_head, payload, REPOSITORY, branch, SHA
                    )

    def test_accepts_realistic_main_ref_payload_with_extra_fields(self) -> None:
        payload = {"total_count": 1, "workflow_runs": [valid_workflow_run()]}
        head = {
            "ref": "refs/heads/main",
            "node_id": "MDM6UmVmMTpyZWZzL2hlYWRzL21haW4=",
            "url": f"https://api.github.com/repos/{REPOSITORY}/git/refs/heads/main",
            "object": {
                "type": "commit",
                "sha": SHA,
                "url": f"https://api.github.com/repos/{REPOSITORY}/git/commits/{SHA}",
            },
        }

        self.assertEqual(
            verify_provenance.validate_manual_candidate(
                head, payload, REPOSITORY, DEFAULT_BRANCH, SHA
            ),
            SHA,
        )

    def test_rejects_missing_or_wrong_main_ref_fields(self) -> None:
        payload = {"total_count": 1, "workflow_runs": [valid_workflow_run()]}
        valid_head = {
            "ref": "refs/heads/main",
            "object": {"type": "commit", "sha": SHA},
        }
        invalid_heads = (
            None,
            {},
            {**valid_head, "ref": "refs/heads/release"},
            {**valid_head, "object": None},
            {**valid_head, "object": {"type": "tag", "sha": SHA}},
            {**valid_head, "object": {"type": "commit", "sha": "b" * 40}},
        )

        for head in invalid_heads:
            with self.subTest(head=head):
                with self.assertRaises(verify_provenance.ProvenanceError):
                    verify_provenance.validate_manual_candidate(
                        head, payload, REPOSITORY, DEFAULT_BRANCH, SHA
                    )

    def test_fails_closed_for_zero_multiple_pending_failed_or_malformed_runs(self) -> None:
        cases = {
            "zero": {"total_count": 0, "workflow_runs": []},
            "multiple": {
                "total_count": 2,
                "workflow_runs": [valid_workflow_run(), valid_workflow_run()],
            },
            "pending": {
                "total_count": 1,
                "workflow_runs": [{**valid_workflow_run(), "status": "in_progress"}],
            },
            "failed": {
                "total_count": 1,
                "workflow_runs": [{**valid_workflow_run(), "conclusion": "failure"}],
            },
            "boolean count": {"total_count": True, "workflow_runs": [valid_workflow_run()]},
            "malformed": {"total_count": 1, "workflow_runs": "not-a-list"},
        }
        for name, payload in cases.items():
            with self.subTest(name=name):
                with self.assertRaises(verify_provenance.ProvenanceError):
                    verify_provenance.validate_manual_runs(
                        payload, REPOSITORY, DEFAULT_BRANCH, SHA
                    )

    def test_rejects_wrong_workflow_identity_even_if_successful(self) -> None:
        for field, value in (
            ("name", "Other"),
            ("path", ".github/workflows/other.yml"),
            ("repository", {"full_name": "attacker/fork"}),
            ("id", True),
        ):
            with self.subTest(field=field):
                run = valid_workflow_run()
                run[field] = value
                with self.assertRaises(verify_provenance.ProvenanceError):
                    verify_provenance.validate_manual_runs(
                        {"total_count": 1, "workflow_runs": [run]},
                        REPOSITORY,
                        DEFAULT_BRANCH,
                        SHA,
                    )

    def test_fetches_current_main_head_from_fixed_github_api(self) -> None:
        response = InstrumentedResponse(
            json.dumps(
                {"ref": "refs/heads/main", "object": {"type": "commit", "sha": SHA}}
            ).encode()
        )
        opener = FakeOpener(response)
        with mock.patch.object(
            verify_provenance.urllib.request, "build_opener", return_value=opener
        ):
            head = verify_provenance.fetch_main_head(
                "https://api.github.com", "dummy-token", REPOSITORY
            )
        self.assertEqual(head["object"]["sha"], SHA)
        self.assertEqual(
            opener.requests[0].full_url,
            f"https://api.github.com/repos/{REPOSITORY}/git/ref/heads/main",
        )
        self.assertEqual(response.read_sizes, [verify_provenance.MAX_RESPONSE_BYTES + 1])

    def test_authenticated_actions_request_never_follows_redirects(self) -> None:
        for location in (
            "https://api.github.com/other",
            "https://attacker.example/steal",
        ):
            with self.subTest(location=location):
                redirect = urllib.error.HTTPError(
                    "https://api.github.com/original",
                    302,
                    "Found",
                    {"Location": location},
                    io.BytesIO(b"{}"),
                )
                opener = FakeOpener(redirect)
                with (
                    mock.patch.object(
                        verify_provenance.urllib.request,
                        "build_opener",
                        return_value=opener,
                    ),
                    mock.patch.object(
                        verify_provenance.urllib.request,
                        "urlopen",
                        side_effect=AssertionError("default opener used"),
                    ),
                ):
                    with self.assertRaises(verify_provenance.ProvenanceError):
                        verify_provenance.fetch_manual_runs(
                            "https://api.github.com",
                            "dummy-token",
                            REPOSITORY,
                            DEFAULT_BRANCH,
                            SHA,
                        )
                self.assertEqual(len(opener.requests), 1)
                self.assertEqual(
                    opener.requests[0].get_header("Authorization"),
                    "Bearer dummy-token",
                )

    def test_actions_response_body_is_read_with_a_strict_cap(self) -> None:
        payload = json.dumps(
            {"total_count": 1, "workflow_runs": [valid_workflow_run()]}
        ).encode()
        response = InstrumentedResponse(payload)
        result, _ = fetch_with(response)
        self.assertEqual(result["total_count"], 1)
        self.assertEqual(
            response.read_sizes, [verify_provenance.MAX_RESPONSE_BYTES + 1]
        )

    def test_actions_response_rejects_oversized_missing_or_dishonest_lengths(self) -> None:
        limit = 128
        cases = {
            "declared oversized": InstrumentedResponse(
                b"{}", content_length=str(limit + 1)
            ),
            "missing length oversized body": InstrumentedResponse(b"x" * (limit + 1)),
            "dishonest small length": InstrumentedResponse(
                b"x" * (limit + 1), content_length="2"
            ),
        }
        with mock.patch.object(
            verify_provenance, "MAX_RESPONSE_BYTES", limit, create=True
        ):
            for name, response in cases.items():
                with self.subTest(name=name):
                    with self.assertRaises(verify_provenance.ProvenanceError):
                        fetch_with(response)
                    self.assertNotIn(-1, response.read_sizes)

    def test_actions_response_rejects_malformed_json(self) -> None:
        with self.assertRaises(verify_provenance.ProvenanceError):
            fetch_with(InstrumentedResponse(b"not json", content_length="8"))

    def test_actions_api_url_is_exact_public_github_authority(self) -> None:
        invalid = (
            "http://api.github.com",
            "https://github.com",
            "https://api.github.com.evil.example",
            "https://user@api.github.com",
            "https://:password@api.github.com",
            "https://api.github.com:444",
            "https://api.github.com/#fragment",
        )
        for api_url in invalid:
            with self.subTest(api_url=api_url):
                with (
                    mock.patch.object(
                        verify_provenance.urllib.request,
                        "build_opener",
                        side_effect=AssertionError("unexpected opener construction"),
                    ) as build_opener,
                    mock.patch.object(
                        verify_provenance.urllib.request,
                        "urlopen",
                        side_effect=AssertionError("unexpected request"),
                    ) as urlopen,
                ):
                    with self.assertRaises(verify_provenance.ProvenanceError):
                        verify_provenance.fetch_manual_runs(
                            api_url,
                            "dummy-token",
                            REPOSITORY,
                            DEFAULT_BRANCH,
                            SHA,
                        )
                build_opener.assert_not_called()
                urlopen.assert_not_called()


if __name__ == "__main__":
    unittest.main()
