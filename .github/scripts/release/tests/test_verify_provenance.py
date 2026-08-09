import importlib.util
import unittest
from pathlib import Path


MODULE_PATH = Path(__file__).resolve().parents[1] / "verify_provenance.py"
spec = importlib.util.spec_from_file_location("verify_provenance", MODULE_PATH)
assert spec is not None
assert spec.loader is not None
verify_provenance = importlib.util.module_from_spec(spec)
spec.loader.exec_module(verify_provenance)


SHA = "a" * 40
REPOSITORY = "Verity-BDD/verity-bdd"
DEFAULT_BRANCH = "main"


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


class WorkflowRunProvenanceTest(unittest.TestCase):
    def test_accepts_only_the_exact_successful_upstream_push_run(self) -> None:
        event = {"workflow_run": valid_workflow_run()}
        self.assertEqual(
            verify_provenance.validate_workflow_run_event(
                event, REPOSITORY, DEFAULT_BRANCH, SHA
            ),
            SHA,
        )

    def test_rejects_pr_failed_foreign_wrong_branch_or_wrong_sha_runs(self) -> None:
        mutations = {
            "pull request": ("event", "pull_request"),
            "failed": ("conclusion", "failure"),
            "foreign": ("head_repository", {"full_name": "attacker/fork"}),
            "wrong branch": ("head_branch", "feature"),
            "wrong SHA": ("head_sha", "b" * 40),
        }
        for name, (field, value) in mutations.items():
            with self.subTest(name=name):
                run = valid_workflow_run()
                run[field] = value
                with self.assertRaises(verify_provenance.ProvenanceError):
                    verify_provenance.validate_workflow_run_event(
                        {"workflow_run": run}, REPOSITORY, DEFAULT_BRANCH, SHA
                    )


class ManualProvenanceTest(unittest.TestCase):
    def test_accepts_one_exact_completed_successful_push_ci_run(self) -> None:
        payload = {"total_count": 1, "workflow_runs": [valid_workflow_run()]}
        self.assertEqual(
            verify_provenance.validate_manual_runs(
                payload, REPOSITORY, DEFAULT_BRANCH, SHA
            ),
            SHA,
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


if __name__ == "__main__":
    unittest.main()
