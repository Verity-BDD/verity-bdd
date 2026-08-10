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


def direct_keys(text: str, indent: int) -> list[str]:
    pattern = rf"^ {{{indent}}}([A-Za-z][A-Za-z0-9_-]*):"
    return re.findall(pattern, text, re.MULTILINE)


def step_blocks(job: str, indent: int) -> list[tuple[str, str]]:
    lines = job.splitlines()
    starts = [
        (index, match.group(1))
        for index, line in enumerate(lines)
        if (match := re.fullmatch(rf" {{{indent}}}- name: (.+)", line))
    ]
    result = []
    for position, (start, name) in enumerate(starts):
        end = starts[position + 1][0] if position + 1 < len(starts) else len(lines)
        result.append((name, "\n".join(lines[start:end])))
    return result


def exact_keys(text: str, indent: int, expected: set[str]) -> bool:
    actual = direct_keys(text, indent)
    return len(actual) == len(expected) and set(actual) == expected


def direct_field_value(text: str, indent: int, key: str) -> str | None:
    """Read one reviewed scalar or literal-block field from a bounded step block."""
    lines = text.splitlines()
    matches = [
        (index, match.group(1))
        for index, line in enumerate(lines)
        if (match := re.fullmatch(rf" {{{indent}}}{re.escape(key)}:(?: (.*))?", line))
    ]
    if len(matches) != 1:
        return None
    index, value = matches[0]
    if value != "|":
        return value
    body = []
    for line in lines[index + 1 :]:
        if line.strip() and len(line) - len(line.lstrip()) <= indent:
            break
        if line.strip() and not line.startswith(" " * (indent + 2)):
            return None
        body.append(line[indent + 2 :] if line else "")
    return "\n".join(body).rstrip()


def block_or_empty(text: str, header: str, indent: int) -> str:
    try:
        return indented_block(text, header, indent)
    except StopIteration:
        return ""


def exact_step_schema(
    job: str, indent: int, expected: list[tuple[str, set[str]]]
) -> bool:
    actual = step_blocks(job, indent)
    return [name for name, _ in actual] == [name for name, _ in expected] and all(
        exact_keys(block, indent + 2, keys - {"name"})
        for (_, block), (_, keys) in zip(actual, expected)
    )


def ci_contract_violations(workflow: str) -> list[str]:
    """Guard reviewed CI structure against accidental/configuration drift.

    This in-repository contract cannot defend against a malicious commit that edits
    both workflow and tests; branch protection and independent review own that limit.
    """
    violations = []
    try:
        jobs = job_blocks(workflow)
    except (KeyError, StopIteration):
        return ["CI topology is malformed"]
    if set(jobs) != {"test-verity", "test-examples", "lint"}:
        violations.append("unexpected CI job topology")
    if direct_keys(workflow, 0) != ["name", "on", "permissions", "jobs"]:
        violations.append("unexpected CI workflow keys")
    if block_or_empty(workflow, "permissions:", 0).strip() != "contents: read":
        violations.append("CI permissions changed")
    refs = re.findall(r"(?m)^\s+-?\s*uses:\s*([^\s#]+)", workflow)
    expected_refs = [
        "actions/checkout@11d5960a326750d5838078e36cf38b85af677262",
        "actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff",
        "actions/cache@0057852bfaa89a56745cba8c7296529d2fc39830",
        "codecov/codecov-action@b9fd7d16f6d7d1b5d2bec1a2887e65ceed900238",
        "actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02",
    ] * 2 + [
        "actions/checkout@11d5960a326750d5838078e36cf38b85af677262",
        "actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff",
        "actions/cache@0057852bfaa89a56745cba8c7296529d2fc39830",
        "golangci/golangci-lint-action@ba0d7d2ec06a0ea1cb5fa41b2e4a3ab91d21278a",
        "actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02",
    ]
    if refs != expected_refs:
        violations.append("CI actions differ from reviewed immutable pins")
    if workflow.count("    runs-on: ubuntu-latest") != 3:
        violations.append("CI jobs must use the reviewed runner")
    if workflow.count('test "$(git rev-parse HEAD)" = "${{ github.sha }}"') != 3:
        violations.append("CI checkout identity checks changed")
    if re.search(r"(?:\|\||&&)\s*true\b|\btrue\s*\|\|", workflow):
        violations.append("CI command failure is suppressed")

    common = [
        ("Checkout code", {"name", "uses", "with"}),
        ("Verify checkout identity", {"name", "run"}),
        ("Setup Go", {"name", "uses", "with"}),
        ("Cache Go modules", {"name", "uses", "with"}),
        ("Download dependencies", {"name", "run"}),
    ]
    schemas = {
        "test-verity": common
        + [
            ("Run tests", {"name", "run"}),
            ("Convert coverage to lcov", {"name", "run"}),
            ("Upload coverage to Codecov", {"name", "uses", "with"}),
            ("Upload coverage artifacts", {"name", "uses", "with"}),
        ],
        "test-examples": common
        + [
            ("Run tests", {"name", "run"}),
            ("Convert coverage to lcov", {"name", "run"}),
            ("Upload coverage to Codecov", {"name", "uses", "with"}),
            ("Upload coverage artifacts", {"name", "uses", "with"}),
        ],
        "lint": common[:2]
        + [
            ("Test release workflow contracts", {"name", "run"}),
            *common[2:],
            ("Run golangci-lint", {"name", "uses", "with"}),
            ("Upload lint results", {"name", "uses", "if", "with"}),
        ],
    }
    expected_runs = {
        "test-verity": {
            "Verify checkout identity": 'test "$(git rev-parse HEAD)" = "${{ github.sha }}"',
            "Download dependencies": "go mod download",
            "Run tests": "go test ./... -v -race -coverprofile=verity-coverage.out",
            "Convert coverage to lcov": (
                "go install github.com/jandelgado/gcov2lcov@latest\n"
                "gcov2lcov -infile verity-coverage.out -outfile verity-coverage.lcov"
            ),
        },
        "test-examples": {
            "Verify checkout identity": 'test "$(git rev-parse HEAD)" = "${{ github.sha }}"',
            "Download dependencies": "go mod download",
            "Run tests": "go test ./examples/... -v -race -coverprofile=examples-coverage.out",
            "Convert coverage to lcov": (
                "go install github.com/jandelgado/gcov2lcov@latest\n"
                "gcov2lcov -infile examples-coverage.out -outfile examples-coverage.lcov"
            ),
        },
        "lint": {
            "Verify checkout identity": 'test "$(git rev-parse HEAD)" = "${{ github.sha }}"',
            "Test release workflow contracts": (
                "python3 -m unittest discover -s .github/scripts/release/tests "
                "-p 'test_*.py' -v"
            ),
            "Download dependencies": "go mod download",
        },
    }
    for name, job in jobs.items():
        allowed_job_keys = {"name", "runs-on", "steps"}
        if name != "lint":
            allowed_job_keys.add("strategy")
        if not exact_keys(job, 4, allowed_job_keys):
            violations.append(f"unexpected keys in CI job {name}")
        if name != "lint":
            strategy = block_or_empty(job, "strategy:", 4).strip()
            if strategy != "matrix:\n        go-version: ['1.23.4', '1.24.x']":
                violations.append(f"CI strategy changed in {name}")
        if not exact_step_schema(job, 4, schemas.get(name, [])):
            violations.append(f"unexpected steps or step keys in CI job {name}")
        steps = dict(step_blocks(job, 4))
        for step_name, expected_run in expected_runs.get(name, {}).items():
            if direct_field_value(steps.get(step_name, ""), 6, "run") != expected_run:
                violations.append(f"CI command changed in {name}: {step_name}")
        checkout = dict(step_blocks(job, 4)).get("Checkout code", "")
        if block_or_empty(checkout, "with:", 6).strip() != "persist-credentials: false":
            violations.append(f"checkout settings changed in CI job {name}")
    if workflow.count("      if: always()") != 1:
        violations.append("CI step condition changed")
    policy_command = (
        "      run: python3 -m unittest discover -s .github/scripts/release/tests "
        "-p 'test_*.py' -v"
    )
    if workflow.count(policy_command) != 1:
        violations.append("release policy command changed")
    return violations


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
    if direct_keys(workflow, 0) != [
        "name",
        "on",
        "permissions",
        "concurrency",
        "jobs",
    ]:
        violations.append("unexpected workflow keys")
    try:
        concurrency = indented_block(workflow, "concurrency:", 0).strip()
    except StopIteration:
        concurrency = ""
    if concurrency != "group: release-publication\n  cancel-in-progress: false":
        violations.append("release concurrency contract changed")
    if "permissions: {}\n\nconcurrency:" not in workflow:
        violations.append("workflow permissions changed")
    expected_permissions = {
        "build-release": "actions: read\n      contents: read",
        "publish-release": "contents: write",
    }
    for name, job in jobs.items():
        try:
            permissions = indented_block(job, "permissions:", 4).strip()
        except StopIteration:
            permissions = ""
        if permissions != expected_permissions.get(name):
            violations.append(f"permissions changed in release job {name}")
    expected_job_keys = {
        "build-release": {
            "name", "if", "permissions", "runs-on", "timeout-minutes", "outputs", "steps"
        },
        "publish-release": {
            "name", "needs", "if", "permissions", "runs-on", "timeout-minutes", "steps"
        },
    }
    for name, job in jobs.items():
        if not exact_keys(job, 4, expected_job_keys.get(name, set())):
            violations.append(f"unexpected keys in release job {name}")
    build_steps = [
        ("Select exact candidate SHA", {"name", "id", "env", "run"}),
        ("Checkout exact candidate", {"name", "uses", "with"}),
        ("Verify checkout identity", {"name", "run"}),
        ("Prove successful push CI provenance", {"name", "id", "env", "run"}),
        ("Prepare deterministic release metadata", {"name", "id", "env", "run"}),
    ]
    publish_steps = [
        ("Reconcile exact tag and GitHub Release", {"name", "env", "run"})
    ]
    if not exact_step_schema(build, 6, build_steps):
        violations.append("build step topology changed")
    if not exact_step_schema(publish, 6, publish_steps):
        violations.append("publish step topology changed")
    expected_gate = """      (github.event_name == 'workflow_run' &&
       github.event.workflow_run.conclusion == 'success' &&
       github.event.workflow_run.event == 'push' &&
       github.event.workflow_run.head_repository.full_name == github.repository &&
       github.event.workflow_run.head_branch == github.event.repository.default_branch) ||
      github.event_name == 'workflow_dispatch'"""
    try:
        gate = indented_block(build, "if: >-", 4)
    except StopIteration:
        gate = ""
    if gate != expected_gate:
        violations.append("release authorization expression changed")
    if re.search(r"(?:\|\||&&)\s*true\b|\btrue\s*\|\|", workflow):
        violations.append("release command failure is suppressed")
    for match in re.finditer(r"(?m)^        run: \|\n(?P<body>(?:^ {10}.*\n?)*)", workflow):
        first = next((line.strip() for line in match.group("body").splitlines() if line.strip()), "")
        if first != "set -euo pipefail":
            violations.append("multiline release command does not start with strict shell")
    step_lookup = dict(step_blocks(build, 6)) | dict(step_blocks(publish, 6))
    checkout_with = block_or_empty(
        step_lookup.get("Checkout exact candidate", ""), "with:", 8
    ).strip()
    expected_checkout_with = """ref: ${{ steps.source.outputs.library_sha }}
          fetch-depth: 0
          fetch-tags: true
          persist-credentials: false"""
    if checkout_with != expected_checkout_with:
        violations.append("checkout settings changed in release build")
    expected_runs = {
        "Select exact candidate SHA": '''set -euo pipefail
if [[ "$EVENT_NAME" == "workflow_run" ]]; then
  library_sha="$WORKFLOW_RUN_SHA"
elif [[ "$EVENT_NAME" == "workflow_dispatch" ]]; then
  library_sha="$MANUAL_SHA"
else
  echo "unsupported release event: $EVENT_NAME" >&2
  exit 1
fi
[[ "$library_sha" =~ ^[0-9a-f]{40}$ ]] || {
  echo "release SHA must be exactly 40 lowercase hexadecimal characters" >&2
  exit 1
}
echo "library_sha=$library_sha" >> "$GITHUB_OUTPUT"''',
        "Verify checkout identity": '''set -euo pipefail
test "$(git rev-parse HEAD)" = "${{ steps.source.outputs.library_sha }}"
test -z "$(git status --porcelain)"''',
        "Prove successful push CI provenance": (
            "python3 .github/scripts/release/verify_provenance.py"
        ),
        "Prepare deterministic release metadata": (
            "python3 .github/scripts/release/prepare_release.py"
        ),
        "Reconcile exact tag and GitHub Release": r'''set -euo pipefail
umask 077
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT

printf '%s' "$RELEASE_TOKEN" > "$temp_dir/token"
unset RELEASE_TOKEN

script_url="https://raw.githubusercontent.com/${REPOSITORY}/${LIBRARY_SHA}/.github/scripts/release/publish_release.py"
env -i HOME="$temp_dir" PATH=/usr/bin:/bin \
  /usr/bin/curl -q --fail --silent --show-error \
  --proto '=https' --tlsv1.2 --max-redirs 0 \
  --output "$temp_dir/publish_release.py" "$script_url"
printf '%s  %s\n' "$PUBLISH_SCRIPT_SHA256" "$temp_dir/publish_release.py" \
  > "$temp_dir/publish_release.py.sha256"
sha256sum --check "$temp_dir/publish_release.py.sha256"

env -i HOME="$temp_dir" PATH=/usr/bin:/bin \
  python3 "$temp_dir/publish_release.py" \
    --api-url "$API_URL" \
    --default-branch "$DEFAULT_BRANCH" \
    --library-sha "$LIBRARY_SHA" \
    --release-body-b64 "$RELEASE_BODY_B64" \
    --repository "$REPOSITORY" \
    --token-file "$temp_dir/token" \
    --version "$VERSION"''',
    }
    for name, expected_run in expected_runs.items():
        if direct_field_value(step_lookup.get(name, ""), 8, "run") != expected_run:
            violations.append(f"release command changed in {name}")
    expected_env_keys = {
        "Select exact candidate SHA": {"EVENT_NAME", "WORKFLOW_RUN_SHA", "MANUAL_SHA"},
        "Prove successful push CI provenance": {
            "ACTIONS_TOKEN", "API_URL", "DEFAULT_BRANCH", "EVENT_NAME", "EVENT_PATH",
            "LIBRARY_SHA", "REPOSITORY",
        },
        "Prepare deterministic release metadata": {"LIBRARY_SHA"},
        "Reconcile exact tag and GitHub Release": {
            "API_URL", "DEFAULT_BRANCH", "LIBRARY_SHA", "PUBLISH_SCRIPT_SHA256",
            "RELEASE_BODY_B64", "RELEASE_TOKEN", "REPOSITORY", "VERSION",
        },
    }
    for name, expected in expected_env_keys.items():
        try:
            env = indented_block(step_lookup.get(name, ""), "env:", 8)
        except StopIteration:
            env = ""
        if not exact_keys(env, 10, expected):
            violations.append(f"release environment changed in {name}")
    for name in (
        "Prove successful push CI provenance",
        "Reconcile exact tag and GitHub Release",
    ):
        api_url = direct_field_value(step_lookup.get(name, ""), 10, "API_URL")
        if api_url != "${{ github.api_url }}":
            violations.append(f"release API URL changed in {name}")
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
            "authorize everything before reviewed expression": (
                "      (github.event_name == 'workflow_run' &&",
                "      true || (github.event_name == 'workflow_run' &&",
            ),
            "suppress publish failure": (
                '              --version "$VERSION"',
                '              --version "$VERSION" || true',
            ),
            "add actions write permission": (
                "    permissions:\n      contents: write\n",
                "    permissions:\n      actions: write\n      contents: write\n",
            ),
            "inject BASH_ENV": (
                "          API_URL: ${{ github.api_url }}\n",
                "          API_URL: ${{ github.api_url }}\n          BASH_ENV: /tmp/attacker\n",
            ),
            "run before strict shell": (
                "        run: |\n          set -euo pipefail\n",
                "        run: |\n          attacker-command\n          set -euo pipefail\n",
            ),
            "remove concurrency": (
                "concurrency:\n  group: release-publication\n  cancel-in-progress: false\n\n",
                "",
            ),
            "cancel an in-progress publication": (
                "  cancel-in-progress: false",
                "  cancel-in-progress: true",
            ),
        }
        for name, (old, new) in mutations.items():
            with self.subTest(name=name):
                self.assertIn(old, self.workflow)
                mutated = self.workflow.replace(old, new, 1)
                self.assertNotEqual(mutated, self.workflow)
                self.assertNotEqual(contract_violations(mutated), [])

    def test_contract_checker_rejects_publish_api_url_redirection(self) -> None:
        publish = self.jobs["publish-release"]
        old = "          API_URL: ${{ github.api_url }}"
        self.assertIn(old, publish)
        mutated_publish = publish.replace(
            old, "          API_URL: https://attacker.example", 1
        )
        self.assertNotEqual(mutated_publish, publish)
        mutated = self.workflow.replace(publish, mutated_publish, 1)
        self.assertNotEqual(contract_violations(mutated), [])

    def test_contract_checker_rejects_checkout_destination_override(self) -> None:
        old = "          persist-credentials: false"
        build = self.jobs["build-release"]
        checkout = dict(step_blocks(build, 6))["Checkout exact candidate"]
        self.assertIn(old, checkout)
        mutated_checkout = checkout.replace(
            old, old + "\n          github-server-url: https://attacker.example", 1
        )
        self.assertNotEqual(mutated_checkout, checkout)
        mutated = self.workflow.replace(checkout, mutated_checkout, 1)
        self.assertNotEqual(contract_violations(mutated), [])

    def test_contract_checker_rejects_inert_release_checkout_identity(self) -> None:
        old = """        run: |
          set -euo pipefail
          test \"$(git rev-parse HEAD)\" = \"${{ steps.source.outputs.library_sha }}\"
          test -z \"$(git status --porcelain)\""""
        new = """        # test \"$(git rev-parse HEAD)\" = \"${{ steps.source.outputs.library_sha }}\"
        run: true"""
        self.assertIn(old, self.workflow)
        mutated = self.workflow.replace(old, new, 1)
        self.assertNotEqual(mutated, self.workflow)
        self.assertNotEqual(contract_violations(mutated), [])

    def test_contract_checker_rejects_duplicate_or_suppressed_release_run(self) -> None:
        old = "        run: python3 .github/scripts/release/verify_provenance.py"
        mutations = (
            old + "\n        run: true",
            "        # " + old.strip() + "\n        run: true",
        )
        self.assertIn(old, self.workflow)
        for replacement in mutations:
            with self.subTest(replacement=replacement):
                mutated = self.workflow.replace(old, replacement, 1)
                self.assertNotEqual(mutated, self.workflow)
                self.assertNotEqual(contract_violations(mutated), [])

    def test_ci_executes_the_release_contract_suite(self) -> None:
        ci = (ROOT / ".github" / "workflows" / "ci.yml").read_text()
        self.assertIn(
            "python3 -m unittest discover -s .github/scripts/release/tests -p 'test_*.py' -v",
            ci,
        )

    def test_ci_actions_are_immutable_and_policy_checkout_is_exact(self) -> None:
        ci = (ROOT / ".github" / "workflows" / "ci.yml").read_text()
        refs = re.findall(r"(?m)^\s+-?\s*uses:\s*([^\s#]+)", ci)
        self.assertTrue(refs)
        self.assertTrue(
            all(re.fullmatch(r"[^@\s]+@[0-9a-f]{40}", ref) for ref in refs), refs
        )
        self.assertEqual(
            ci.count('test "$(git rev-parse HEAD)" = "${{ github.sha }}"'), 3
        )

    def test_ci_policy_wiring_propagates_failure_and_rejects_execution_injection(self) -> None:
        ci = (ROOT / ".github" / "workflows" / "ci.yml").read_text()
        self.assertEqual(ci_contract_violations(ci), [])
        mutations = {
            "mutable CI action": (
                "actions/checkout@11d5960a326750d5838078e36cf38b85af677262",
                "actions/checkout@v4",
            ),
            "skip lint policy job": ("  lint:\n", "  lint:\n    if: false\n"),
            "expand CI permissions": (
                "permissions:\n  contents: read",
                "permissions:\n  actions: write\n  contents: read",
            ),
            "suppress policy command failure": (
                "      run: python3 -m unittest discover -s .github/scripts/release/tests -p 'test_*.py' -v",
                "      run: python3 -m unittest discover -s .github/scripts/release/tests -p 'test_*.py' -v || true",
            ),
            "inject policy BASH_ENV": (
                "    steps:\n    - name: Checkout code\n",
                "    env:\n      BASH_ENV: /tmp/attacker\n    steps:\n    - name: Checkout code\n",
            ),
        }
        for name, (old, new) in mutations.items():
            with self.subTest(name=name):
                self.assertIn(old, ci)
                self.assertNotEqual(ci_contract_violations(ci.replace(old, new, 1)), [])

    def test_ci_contract_rejects_disabled_go_tests_and_inert_identity_text(self) -> None:
        ci = (ROOT / ".github" / "workflows" / "ci.yml").read_text()
        mutations = {
            "disabled package tests": (
                "      run: go test ./... -v -race -coverprofile=verity-coverage.out",
                "      run: true",
            ),
            "disabled example tests": (
                "      run: go test ./examples/... -v -race -coverprofile=examples-coverage.out",
                "      run: true",
            ),
            "inert checkout identity": (
                '      run: test "$(git rev-parse HEAD)" = "${{ github.sha }}"',
                '      # test "$(git rev-parse HEAD)" = "${{ github.sha }}"\n      run: true',
            ),
        }
        for name, (old, new) in mutations.items():
            with self.subTest(name=name):
                self.assertIn(old, ci)
                mutated = ci.replace(old, new, 1)
                self.assertNotEqual(mutated, ci)
                self.assertNotEqual(ci_contract_violations(mutated), [])

    def test_ci_contract_rejects_duplicate_required_run_key(self) -> None:
        ci = (ROOT / ".github" / "workflows" / "ci.yml").read_text()
        old = "      run: go test ./... -v -race -coverprofile=verity-coverage.out"
        self.assertIn(old, ci)
        mutated = ci.replace(old, old + "\n      run: true", 1)
        self.assertNotEqual(mutated, ci)
        self.assertNotEqual(ci_contract_violations(mutated), [])


if __name__ == "__main__":
    unittest.main()
