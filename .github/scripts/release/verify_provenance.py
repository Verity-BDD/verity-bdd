#!/usr/bin/env python3
"""Fail-closed provenance validation for release candidates."""

from __future__ import annotations

import json
import os
import re
import sys
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any


SHA_RE = re.compile(r"^[0-9a-f]{40}$")
WORKFLOW_NAME = "CI"
WORKFLOW_PATH = ".github/workflows/ci.yml"


class ProvenanceError(RuntimeError):
    pass


def _positive_int(value: Any) -> bool:
    return type(value) is int and value > 0


def _repo_name(value: Any) -> str | None:
    return value.get("full_name") if isinstance(value, dict) else None


def _validate_run(
    run: Any, repository: str, default_branch: str, library_sha: str
) -> str:
    if not isinstance(run, dict):
        raise ProvenanceError("workflow run must be an object")
    expected = {
        "name": WORKFLOW_NAME,
        "path": WORKFLOW_PATH,
        "event": "push",
        "head_branch": default_branch,
        "head_sha": library_sha,
        "status": "completed",
        "conclusion": "success",
    }
    for field, value in expected.items():
        if run.get(field) != value:
            raise ProvenanceError(f"workflow run has unexpected {field}")
    if not _positive_int(run.get("id")):
        raise ProvenanceError("workflow run id must be a positive integer")
    if _repo_name(run.get("head_repository")) != repository:
        raise ProvenanceError("workflow run head repository does not match")
    if _repo_name(run.get("repository")) != repository:
        raise ProvenanceError("workflow run repository does not match")
    return library_sha


def _validate_inputs(repository: str, default_branch: str, library_sha: str) -> None:
    if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repository):
        raise ProvenanceError("invalid repository identity")
    if not re.fullmatch(r"[A-Za-z0-9._/-]+", default_branch):
        raise ProvenanceError("invalid default branch")
    if not SHA_RE.fullmatch(library_sha):
        raise ProvenanceError("library SHA must be 40 lowercase hexadecimal characters")


def validate_workflow_run_event(
    event: Any, repository: str, default_branch: str, library_sha: str
) -> str:
    _validate_inputs(repository, default_branch, library_sha)
    if not isinstance(event, dict):
        raise ProvenanceError("event payload must be an object")
    return _validate_run(event.get("workflow_run"), repository, default_branch, library_sha)


def validate_manual_runs(
    payload: Any, repository: str, default_branch: str, library_sha: str
) -> str:
    _validate_inputs(repository, default_branch, library_sha)
    if not isinstance(payload, dict):
        raise ProvenanceError("Actions response must be an object")
    runs = payload.get("workflow_runs")
    count = payload.get("total_count")
    if type(count) is not int or count != 1:
        raise ProvenanceError("manual release requires exactly one matching CI run")
    if not isinstance(runs, list) or len(runs) != 1:
        raise ProvenanceError("manual release CI run list is malformed or ambiguous")
    return _validate_run(runs[0], repository, default_branch, library_sha)


def _api_origin(api_url: str) -> str:
    parsed = urllib.parse.urlsplit(api_url)
    if (
        parsed.scheme != "https"
        or not parsed.hostname
        or parsed.username is not None
        or parsed.password is not None
        or parsed.port not in (None, 443)
        or parsed.query
        or parsed.fragment
    ):
        raise ProvenanceError("invalid GitHub API URL")
    return api_url.rstrip("/")


def fetch_manual_runs(
    api_url: str,
    token: str,
    repository: str,
    default_branch: str,
    library_sha: str,
) -> Any:
    if not token or "\n" in token or "\r" in token:
        raise ProvenanceError("invalid Actions token")
    origin = _api_origin(api_url)
    workflow = urllib.parse.quote(WORKFLOW_PATH, safe="")
    query = urllib.parse.urlencode(
        {
            "branch": default_branch,
            "event": "push",
            "head_sha": library_sha,
            "per_page": "2",
        }
    )
    url = f"{origin}/repos/{repository}/actions/workflows/{workflow}/runs?{query}"
    request = urllib.request.Request(
        url,
        headers={
            "Accept": "application/vnd.github+json",
            "Authorization": f"Bearer {token}",
            "User-Agent": "verity-bdd-release-provenance",
            "X-GitHub-Api-Version": "2022-11-28",
        },
    )
    try:
        with urllib.request.urlopen(request, timeout=20) as response:
            if response.status != 200:
                raise ProvenanceError(f"Actions API returned HTTP {response.status}")
            return json.load(response)
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError) as error:
        raise ProvenanceError("unable to read successful CI provenance") from error


def _write_output(library_sha: str) -> None:
    output_path = os.environ.get("GITHUB_OUTPUT")
    if not output_path:
        raise ProvenanceError("GITHUB_OUTPUT is not set")
    with Path(output_path).open("a", encoding="utf-8") as output:
        output.write(f"library_sha={library_sha}\n")


def main() -> int:
    try:
        event_name = os.environ["EVENT_NAME"]
        event_path = Path(os.environ["EVENT_PATH"])
        repository = os.environ["REPOSITORY"]
        default_branch = os.environ["DEFAULT_BRANCH"]
        library_sha = os.environ["LIBRARY_SHA"]
        if event_name == "workflow_run":
            with event_path.open(encoding="utf-8") as source:
                event = json.load(source)
            validated_sha = validate_workflow_run_event(
                event, repository, default_branch, library_sha
            )
        elif event_name == "workflow_dispatch":
            payload = fetch_manual_runs(
                os.environ["API_URL"],
                os.environ["ACTIONS_TOKEN"],
                repository,
                default_branch,
                library_sha,
            )
            validated_sha = validate_manual_runs(
                payload, repository, default_branch, library_sha
            )
        else:
            raise ProvenanceError("unsupported release event")
        _write_output(validated_sha)
        print(f"validated successful push CI provenance for {validated_sha}")
        return 0
    except (KeyError, OSError, json.JSONDecodeError, ProvenanceError) as error:
        print(f"release provenance rejected: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
