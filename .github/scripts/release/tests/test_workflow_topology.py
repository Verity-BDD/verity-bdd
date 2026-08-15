import re
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]
RELEASE = ROOT / ".github" / "workflows" / "release.yml"
CI = ROOT / ".github" / "workflows" / "ci.yml"
DOCS_TRIGGER = ROOT / ".github" / "workflows" / "trigger-docs.yml"
PINNED_CHECKOUT = "actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683"
PINNED_COVERAGE_TOOL = "github.com/jandelgado/gcov2lcov@v1.1.1"
PUBLISH_ELIGIBILITY = "inputs.preview != true && needs.build-release.outputs.version != ''"


def indented_block(text: str, header: str, indent: int) -> str:
    lines = text.splitlines()
    start = next(i for i, line in enumerate(lines) if line == " " * indent + header)
    block = []
    for line in lines[start + 1 :]:
        if line.strip() and len(line) - len(line.lstrip()) <= indent:
            break
        block.append(line)
    return "\n".join(block)


def job_blocks(workflow: str) -> dict[str, str]:
    jobs = indented_block(workflow, "jobs:", 0).splitlines()
    starts = [
        (index, match.group(1))
        for index, line in enumerate(jobs)
        if (match := re.fullmatch(r"  ([a-zA-Z0-9_-]+):", line))
    ]
    return {
        name: "\n".join(jobs[start : starts[position + 1][0]])
        if position + 1 < len(starts)
        else "\n".join(jobs[start:])
        for position, (start, name) in enumerate(starts)
    }


def action_refs(text: str) -> list[str]:
    return re.findall(r"(?m)^\s+-?\s*uses:\s*([^\s#]+)", text)


def normalized_job_expression(job: str, key: str) -> str:
    matches = re.findall(rf"(?m)^    {re.escape(key)}:\s*(.+?)\s*$", job)
    if len(matches) != 1:
        raise AssertionError(f"expected exactly one {key!r} expression")
    return " ".join(matches[0].split())


def assert_publish_is_preview_safe(publish: str) -> None:
    if normalized_job_expression(publish, "if") != PUBLISH_ELIGIBILITY:
        raise AssertionError("publish eligibility must be the complete preview-safe expression")
    if "continue-on-error:" in publish:
        raise AssertionError("publication must not suppress failures")
    if re.search(r"(?:\|\||&&)\s*true\b|\btrue\s*\|\|", publish):
        raise AssertionError("publication must not contain a shell bypass")


class ReleaseWorkflowSecurityTest(unittest.TestCase):
    def setUp(self) -> None:
        self.release = RELEASE.read_text()
        self.jobs = job_blocks(self.release)

    def test_release_is_manual_only_with_required_version_and_dispatch_sha(self) -> None:
        self.assertNotIn("workflow_run:", self.release)
        dispatch = indented_block(self.release, "workflow_dispatch:", 2)
        self.assertRegex(dispatch, r"(?m)^ {6}version:$")
        self.assertRegex(dispatch, r"(?m)^ {8}required: true$")
        self.assertNotIn("library_sha:", dispatch)
        build = self.jobs["build-release"]
        self.assertEqual(indented_block(build, "if: >-", 4).strip(), "github.ref == 'refs/heads/main'")
        self.assertIn("LIBRARY_SHA: ${{ github.sha }}", build)
        self.assertIn("VERSION: ${{ inputs.version }}", build)
        for history_env in (
            "ACTIONS_TOKEN: ${{ github.token }}",
            "API_URL: ${{ github.api_url }}",
            "REPOSITORY: ${{ github.repository }}",
        ):
            self.assertGreaterEqual(build.count(history_env), 2)
        self.assertNotIn("WORKFLOW_RUN_SHA", build)
        self.assertNotIn("MANUAL_SHA", build)
        self.assertEqual(
            build.count("run: python3 .github/scripts/release/verify_provenance.py"),
            1,
        )

    def test_preview_is_read_only_and_structurally_skips_publication_after_shared_preparation(self) -> None:
        dispatch = indented_block(self.release, "workflow_dispatch:", 2)
        self.assertRegex(dispatch, r"(?m)^ {6}preview:$")
        self.assertRegex(dispatch, r"(?m)^ {8}default: false$")
        self.assertRegex(dispatch, r"(?m)^ {8}type: boolean$")

        build = self.jobs["build-release"]
        self.assertEqual(build.count("run: python3 .github/scripts/release/prepare_release.py"), 1)
        self.assertIn("if: inputs.preview", build)
        for preview_field in (
            "requested version",
            "release type",
            "library SHA",
            "predecessor",
            "changelog range",
            "release notes",
        ):
            self.assertIn(preview_field, build)
        self.assertNotIn("contents: write", build)

        publish = self.jobs["publish-release"]
        assert_publish_is_preview_safe(publish)

    def test_preview_true_cannot_make_write_capable_publish_job_eligible(self) -> None:
        publish = self.jobs["publish-release"]
        mutations = (
            (
                publish.replace(
                    PUBLISH_ELIGIBILITY,
                    f"{PUBLISH_ELIGIBILITY} || true",
                ),
                "preview-safe",
            ),
            (publish.replace("permissions:\n", "continue-on-error: true\n    permissions:\n"), "suppress"),
        )
        for mutated, failure in mutations:
            with self.subTest(failure=failure):
                self.assertNotEqual(mutated, publish)
                with self.assertRaisesRegex(AssertionError, failure):
                    assert_publish_is_preview_safe(mutated)

    def test_legacy_local_release_script_is_removed(self) -> None:
        self.assertFalse((ROOT / "scripts" / "release.sh").exists())

    def test_release_documentation_describes_read_only_preview(self) -> None:
        releasing = (ROOT / "docs" / "RELEASING.md").read_text().lower()
        self.assertIn("preview", releasing)
        self.assertIn("no remote writes", releasing)

    def test_build_is_read_only_and_checks_out_the_selected_sha(self) -> None:
        build = self.jobs["build-release"]
        self.assertEqual(action_refs(build), [PINNED_CHECKOUT])
        self.assertEqual(
            indented_block(build, "permissions:", 4).strip(),
            "actions: read\n      contents: read",
        )
        self.assertIn("ref: ${{ github.sha }}", build)
        self.assertIn("persist-credentials: false", build)
        self.assertIn(
            'test "$(git rev-parse HEAD)" = "${{ github.sha }}"',
            build,
        )
        self.assertNotIn("contents: write", build)

    def test_publish_is_one_action_free_write_step_bound_to_the_built_sha(self) -> None:
        publish = self.jobs["publish-release"]
        self.assertIn("needs: build-release", publish)
        assert_publish_is_preview_safe(publish)
        self.assertEqual(
            indented_block(publish, "permissions:", 4).strip(),
            "contents: write",
        )
        self.assertEqual(action_refs(publish), [])
        self.assertEqual(
            len(re.findall(r"(?m)^      - [A-Za-z][A-Za-z0-9_-]*:", publish)),
            1,
        )
        self.assertEqual(
            publish.count('env -i HOME="$temp_dir" PATH=/usr/bin:/bin'),
            2,
        )
        self.assertRegex(
            publish,
            r'(?m)^          sha256sum --check "\$temp_dir/publish_release\.py\.sha256"$',
        )
        for requirement in (
            "LIBRARY_SHA: ${{ needs.build-release.outputs.library_sha }}",
            "PUBLISH_SCRIPT_SHA256: ${{ needs.build-release.outputs.publish_script_sha256 }}",
            'script_url="https://raw.githubusercontent.com/${REPOSITORY}/${LIBRARY_SHA}/',
            "--proto '=https' --tlsv1.2 --max-redirs 0",
            'printf \'%s\' "$RELEASE_TOKEN" > "$temp_dir/token"',
            "unset RELEASE_TOKEN",
            '--library-sha "$LIBRARY_SHA"',
            '--token-file "$temp_dir/token"',
        ):
            self.assertIn(requirement, publish)
        self.assertNotRegex(publish, r"(?:\|\||&&)\s*true\b|\btrue\s*\|\|")

    def test_release_has_no_execution_bypasses_or_floating_targets(self) -> None:
        release_dir = ROOT / ".github" / "scripts" / "release"
        executable = self.release + "\n" + "\n".join(
            path.read_text() for path in release_dir.glob("*.py")
        )
        for forbidden in (
            "continue-on-error:",
            "repository-dispatch",
            "DOCS_REPO_TOKEN",
            '"target_commitish": "main"',
        ):
            self.assertNotIn(forbidden, executable)
        self.assertNotRegex(executable, r"\bgit\s+push\b")
        self.assertNotIn("container:", self.release)
        self.assertNotIn("services:", self.release)

    def test_cross_repository_dispatch_is_removed(self) -> None:
        self.assertFalse(DOCS_TRIGGER.exists())


class CiSecurityContractTest(unittest.TestCase):
    def setUp(self) -> None:
        self.ci = CI.read_text()

    def test_ci_runs_tests_and_release_policy_without_suppression(self) -> None:
        for command in (
            "go test ./... -v -race -coverprofile=verity-coverage.out",
            "go test ./examples/... -v -race -coverprofile=examples-coverage.out",
            "python3 -m unittest discover -s .github/scripts/release/tests -p 'test_*.py' -v",
        ):
            self.assertEqual(self.ci.count(command), 1)
        self.assertNotIn("continue-on-error:", self.ci)
        self.assertNotRegex(self.ci, r"(?:\|\||&&)\s*true\b|\btrue\s*\|\|")

    def test_ci_uses_read_only_permissions_and_immutable_actions(self) -> None:
        self.assertIn("permissions:\n  contents: read", self.ci)
        refs = action_refs(self.ci)
        self.assertTrue(refs)
        self.assertTrue(
            all(re.fullmatch(r"[^@\s]+@[0-9a-f]{40}", ref) for ref in refs),
            refs,
        )
        self.assertEqual(
            self.ci.count('test "$(git rev-parse HEAD)" = "${{ github.sha }}"'),
            3,
        )

    def test_ci_installs_only_the_reviewed_coverage_tool_version(self) -> None:
        installs = re.findall(r"go install (github\.com/[^\s@]+@[^\s]+)", self.ci)
        self.assertEqual(installs, [PINNED_COVERAGE_TOOL] * 2)
        self.assertNotIn("@latest", self.ci)


if __name__ == "__main__":
    unittest.main()
