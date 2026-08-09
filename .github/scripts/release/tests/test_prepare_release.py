import importlib.util
import subprocess
import tempfile
import unittest
from pathlib import Path


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

    def test_prepares_deterministic_minor_release_for_exact_sha(self) -> None:
        target = commit(self.repo, "feat(api): add exact provenance")
        commit(self.repo, "fix: later commit must not enter selected release")

        first = prepare_release.build_release(self.repo, target)
        second = prepare_release.build_release(self.repo, target)

        self.assertEqual(first, second)
        self.assertIsNotNone(first)
        assert first is not None
        self.assertEqual(first.library_sha, target)
        self.assertEqual(first.version, "v1.3.0")
        self.assertIn("feat(api): add exact provenance", first.body)
        self.assertNotIn("later commit", first.body)
        self.assertIn(target, first.body)
        self.assertRegex(first.publish_script_sha256, r"^[0-9a-f]{64}$")

    def test_tagged_selected_sha_still_emits_metadata_for_retry_reconciliation(self) -> None:
        tagged = git(self.repo, "rev-parse", "HEAD")
        release = prepare_release.build_release(self.repo, tagged)
        self.assertIsNotNone(release)
        assert release is not None
        self.assertEqual(release.version, "v1.2.3")
        self.assertEqual(release.library_sha, tagged)
        self.assertIn("chore: bootstrap", release.body)

    def test_rejects_ambiguous_release_tags_on_selected_sha(self) -> None:
        tagged = git(self.repo, "rev-parse", "HEAD")
        git(self.repo, "tag", "v1.2.4", tagged)

        with self.assertRaisesRegex(prepare_release.PrepareError, "ambiguous release tags"):
            prepare_release.build_release(self.repo, tagged)

    def test_breaking_change_takes_precedence_over_feature_and_patch(self) -> None:
        commit(self.repo, "fix: ordinary correction")
        target = commit(self.repo, "feat!: remove old contract", "BREAKING CHANGE: old API removed")
        release = prepare_release.build_release(self.repo, target)
        assert release is not None
        self.assertEqual(release.version, "v2.0.0")

    def test_rejects_non_commit_or_non_exact_sha(self) -> None:
        for candidate in ("main", "A" * 40, "a" * 39):
            with self.subTest(candidate=candidate):
                with self.assertRaises(prepare_release.PrepareError):
                    prepare_release.build_release(self.repo, candidate)


if __name__ == "__main__":
    unittest.main()
