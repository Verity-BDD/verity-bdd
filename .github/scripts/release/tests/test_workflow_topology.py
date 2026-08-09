import re
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]
RELEASE = ROOT / ".github" / "workflows" / "release.yml"
DOCS_TRIGGER = ROOT / ".github" / "workflows" / "trigger-docs.yml"
PINNED_CHECKOUT = "actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683"


def indented_block(text: str, header: str, indent: int) -> str:
    lines = text.splitlines()
    start = next(i for i, line in enumerate(lines) if line == " " * indent + header)
    block = []
    for line in lines[start + 1 :]:
        if line.strip() and len(line) - len(line.lstrip()) <= indent:
            break
        block.append(line)
    return "\n".join(block)


def job_blocks(text: str) -> dict[str, str]:
    jobs = indented_block(text, "jobs:", 0).splitlines()
    result: dict[str, list[str]] = {}
    current = None
    for line in jobs:
        match = re.fullmatch(r"  ([a-zA-Z0-9_-]+):", line)
        if match:
            current = match.group(1)
            result[current] = []
        elif current is not None:
            result[current].append(line)
    return {name: "\n".join(lines) for name, lines in result.items()}


def action_refs(job: str) -> list[str]:
    return [
        match.group(1)
        for match in re.finditer(r"^\s+(?:-\s+)?uses:\s*([^\s#]+)", job, re.MULTILINE)
    ]


def contract_violations(workflow: str) -> list[str]:
    violations = []
    try:
        jobs = job_blocks(workflow)
        trigger = indented_block(workflow, "workflow_run:", 2)
    except (KeyError, StopIteration):
        return ["workflow topology is malformed"]
    if set(jobs) != {"build-release", "publish-release"}:
        violations.append("unexpected job topology")
    build = jobs.get("build-release", "")
    publish = jobs.get("publish-release", "")
    for required in (
        "branches: [main]",
        'workflows: ["CI"]',
        "types: [completed]",
    ):
        if required not in trigger:
            violations.append(f"missing trigger contract: {required}")
    for required in (
        "github.event.workflow_run.conclusion == 'success'",
        "github.event.workflow_run.event == 'push'",
        "github.event.workflow_run.head_repository.full_name == github.repository",
        "github.event.workflow_run.head_branch == github.event.repository.default_branch",
        "github.event_name == 'workflow_dispatch'",
        "ref: ${{ steps.source.outputs.library_sha }}",
        "persist-credentials: false",
        "python3 .github/scripts/release/verify_provenance.py",
    ):
        if required not in build:
            violations.append(f"missing build contract: {required}")
    if action_refs(build) != [PINNED_CHECKOUT]:
        violations.append("build actions are not exactly pinned checkout")
    if "contents: write" in build:
        violations.append("build has write permission")
    if workflow.count("    runs-on: ubuntu-latest") != 2:
        violations.append("release jobs must use the reviewed runner")
    if re.search(r"(?m)^(?:env|defaults):", workflow):
        violations.append("workflow-level execution environment is forbidden")
    if re.search(r"(?m)^    (?:container|services):", workflow):
        violations.append("job container or services are forbidden")
    if re.search(r"(?m)^        if:", build + "\n" + publish):
        violations.append("release steps may not be conditionally skipped")
    if "    if: needs.build-release.outputs.version != ''" not in publish:
        violations.append("publish release condition changed")
    if "        run: python3 .github/scripts/release/prepare_release.py" not in build:
        violations.append("release preparation command changed")
    if action_refs(publish):
        violations.append("publish job contains an action")
    if "needs: build-release" not in publish:
        violations.append("publish does not depend on build")
    if "contents: write" not in publish:
        violations.append("publish lacks contents write")
    if "--library-sha" not in publish or "sha256sum --check" not in publish:
        violations.append("publish exact-SHA/checksum invocation is missing")
    for forbidden in ("continue-on-error:", "shell:", "repository-dispatch", "DOCS_REPO_TOKEN"):
        if forbidden in workflow:
            violations.append(f"forbidden execution control: {forbidden}")
    return violations


class ReleaseWorkflowTopologyTest(unittest.TestCase):
    def setUp(self) -> None:
        self.workflow = RELEASE.read_text()
        self.jobs = job_blocks(self.workflow)

    def test_only_exact_successful_push_ci_can_enter_automatic_release(self) -> None:
        trigger = indented_block(self.workflow, "workflow_run:", 2)
        self.assertIn('workflows: ["CI"]', trigger)
        self.assertIn("types: [completed]", trigger)
        self.assertIn("branches: [main]", trigger)
        self.assertNotIn("master", trigger)

        gate = self.jobs["build-release"]
        for required in (
            "github.event_name == 'workflow_run'",
            "github.event.workflow_run.conclusion == 'success'",
            "github.event.workflow_run.event == 'push'",
            "github.event.workflow_run.head_repository.full_name == github.repository",
            "github.event.workflow_run.head_branch == github.event.repository.default_branch",
            "github.event_name == 'workflow_dispatch'",
        ):
            self.assertIn(required, gate)

    def test_manual_dispatch_requires_an_explicit_exact_sha(self) -> None:
        dispatch = indented_block(self.workflow, "workflow_dispatch:", 2)
        self.assertRegex(dispatch, r"(?m)^ {6}library_sha:$")
        self.assertRegex(dispatch, r"(?m)^ {8}required: true$")
        self.assertIn("python3 .github/scripts/release/verify_provenance.py", self.jobs["build-release"])

    def test_build_checks_out_and_verifies_the_selected_exact_sha(self) -> None:
        build = self.jobs["build-release"]
        self.assertEqual(action_refs(build), [PINNED_CHECKOUT])
        self.assertIn("ref: ${{ steps.source.outputs.library_sha }}", build)
        self.assertIn("persist-credentials: false", build)
        self.assertIn('test "$(git rev-parse HEAD)" = "${{ steps.source.outputs.library_sha }}"', build)

    def test_write_boundary_is_one_action_free_step_with_minimum_permission(self) -> None:
        publish = self.jobs["publish-release"]
        self.assertIn("contents: write", publish)
        self.assertNotIn("packages: write", self.workflow)
        self.assertEqual(action_refs(publish), [])
        self.assertEqual(len(re.findall(r"(?m)^      - name:", publish)), 1)
        self.assertIn("publish_release.py", publish)
        self.assertIn("sha256sum --check", publish)

    def test_cross_repository_dispatch_is_removed(self) -> None:
        self.assertFalse(DOCS_TRIGGER.exists())
        self.assertNotIn("repository-dispatch", self.workflow)
        self.assertNotIn("DOCS_REPO_TOKEN", self.workflow)

    def test_release_never_pushes_or_targets_a_floating_ref(self) -> None:
        release_dir = ROOT / ".github" / "scripts" / "release"
        release_files = list(release_dir.glob("*.py")) if release_dir.exists() else []
        executable_contract = self.workflow + "\n" + "\n".join(path.read_text() for path in release_files)
        self.assertNotRegex(executable_contract, r"\bgit\s+push\b")
        self.assertNotIn('"target_commitish": "main"', executable_contract)
        self.assertIn("--library-sha", self.jobs["publish-release"])

    def test_contract_checker_rejects_security_topology_mutations(self) -> None:
        self.assertEqual(contract_violations(self.workflow), [])
        mutations = {
            "accept PR CI": (
                "github.event.workflow_run.event == 'push'",
                "github.event.workflow_run.event == 'pull_request'",
            ),
            "floating checkout": (
                "ref: ${{ steps.source.outputs.library_sha }}",
                "ref: ${{ github.sha }}",
            ),
            "mutable checkout": (
                PINNED_CHECKOUT,
                "actions/checkout@v4",
            ),
            "persist credentials": (
                "persist-credentials: false",
                "persist-credentials: true",
            ),
            "write-capable build": (
                "      contents: read",
                "      contents: write",
            ),
            "bypass failure": (
                "        run: python3 .github/scripts/release/verify_provenance.py",
                "        continue-on-error: true\n        run: python3 .github/scripts/release/verify_provenance.py",
            ),
            "untrusted publish action": (
                "    steps:\n      - name: Reconcile exact tag and GitHub Release",
                "    steps:\n      - uses: attacker/release@v1\n      - name: Reconcile exact tag and GitHub Release",
            ),
            "skip publish": (
                "    if: needs.build-release.outputs.version != ''",
                "    if: false",
            ),
            "publish container": (
                "    needs: build-release\n",
                "    needs: build-release\n    container: attacker/image:latest\n",
            ),
            "publish services": (
                "    needs: build-release\n",
                "    needs: build-release\n    services:\n      helper:\n        image: attacker/image:latest\n",
            ),
            "changed runner": (
                "    runs-on: ubuntu-latest\n",
                "    runs-on: self-hosted\n",
            ),
            "workflow environment injection": (
                "permissions: {}\n",
                "permissions: {}\nenv:\n  PYTHONPATH: /tmp/attacker\n",
            ),
            "skip provenance step": (
                "      - name: Prove successful push CI provenance\n",
                "      - name: Prove successful push CI provenance\n        if: false\n",
            ),
            "alter preparation command": (
                "        run: python3 .github/scripts/release/prepare_release.py",
                "        run: python3 -c \"print('fake')\"",
            ),
        }
        for name, (old, new) in mutations.items():
            with self.subTest(name=name):
                self.assertIn(old, self.workflow)
                mutated = self.workflow.replace(old, new, 1)
                self.assertNotEqual(mutated, self.workflow)
                self.assertNotEqual(contract_violations(mutated), [])

    def test_ci_executes_the_release_contract_suite(self) -> None:
        ci = (ROOT / ".github" / "workflows" / "ci.yml").read_text()
        self.assertIn(
            "python3 -m unittest discover -s .github/scripts/release/tests -p 'test_*.py' -v",
            ci,
        )


if __name__ == "__main__":
    unittest.main()
