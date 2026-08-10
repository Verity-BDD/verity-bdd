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
MAX_RESPONSE_BYTES = 1_048_576


class ProvenanceError(RuntimeError):
    pass


def _positive_int(value: Any) -> bool:
    return type(value) is int and value > 0


def _repo_name(value: Any) -> str | None:
    if type(value) is not dict or type(value.get("full_name")) is not str:
        return None
    return value["full_name"]


def _validate_run(
    run: Any, repository: str, default_branch: str, library_sha: str
) -> str:
    if type(run) is not dict:
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


def validate_manual_candidate(
    head: Any,
    runs_payload: Any,
    repository: str,
    default_branch: str,
    library_sha: str,
) -> str:
    if default_branch != "main":
        raise ProvenanceError("release dispatch is allowed only on main")
    if type(head) is not dict or head.get("ref") != "refs/heads/main":
        raise ProvenanceError("candidate is not the current remote main HEAD")
    head_object = head.get("object")
    if (
        type(head_object) is not dict
        or head_object.get("type") != "commit"
        or head_object.get("sha") != library_sha
    ):
        raise ProvenanceError("candidate is not the current remote main HEAD")
    return validate_manual_runs(runs_payload, repository, default_branch, library_sha)


def validate_manual_runs(
    payload: Any, repository: str, default_branch: str, library_sha: str
) -> str:
    _validate_inputs(repository, default_branch, library_sha)
    if type(payload) is not dict:
        raise ProvenanceError("Actions response must be an object")
    runs = payload.get("workflow_runs")
    count = payload.get("total_count")
    if type(count) is not int or count != 1:
        raise ProvenanceError("manual release requires exactly one matching CI run")
    if type(runs) is not list or len(runs) != 1:
        raise ProvenanceError("manual release CI run list is malformed or ambiguous")
    return _validate_run(runs[0], repository, default_branch, library_sha)


def _api_origin(api_url: str) -> str:
    try:
        parsed = urllib.parse.urlsplit(api_url)
        valid_port = parsed.port in (None, 443)
    except ValueError as error:
        raise ProvenanceError("invalid GitHub API URL") from error
    if (
        parsed.scheme != "https"
        or parsed.hostname != "api.github.com"
        or parsed.username is not None
        or parsed.password is not None
        or not valid_port
        or parsed.path not in ("", "/")
        or parsed.query
        or parsed.fragment
    ):
        raise ProvenanceError("invalid GitHub API URL")
    return "https://api.github.com"


class _NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):  # noqa: ANN001
        return None


def _read_json_response(response: Any) -> dict[str, Any]:
    if type(response.status) is not int or response.status != 200:
        raise ProvenanceError("Actions API did not return HTTP 200")
    content_length = response.headers.get("Content-Length")
    declared_length = None
    if content_length is not None:
        if type(content_length) is not str or not content_length.isdecimal():
            raise ProvenanceError("Actions API Content-Length is malformed")
        declared_length = int(content_length)
        if declared_length > MAX_RESPONSE_BYTES:
            raise ProvenanceError("Actions API response exceeded size limit")
    raw = response.read(MAX_RESPONSE_BYTES + 1)
    if type(raw) is not bytes or len(raw) > MAX_RESPONSE_BYTES:
        raise ProvenanceError("Actions API response exceeded size limit")
    if declared_length is not None and len(raw) != declared_length:
        raise ProvenanceError("Actions API Content-Length did not match response body")
    try:
        decoded = json.loads(raw)
    except (json.JSONDecodeError, UnicodeDecodeError) as error:
        raise ProvenanceError("Actions API returned malformed JSON") from error
    if type(decoded) is not dict:
        raise ProvenanceError("Actions API response must be an object")
    return decoded


def fetch_main_head(
    api_url: str, token: str, repository: str
) -> dict[str, Any]:
    if not token or "\n" in token or "\r" in token:
        raise ProvenanceError("invalid Actions token")
    origin = _api_origin(api_url)
    request = urllib.request.Request(
        f"{origin}/repos/{repository}/git/ref/heads/main",
        headers={
            "Accept": "application/vnd.github+json",
            "Authorization": f"Bearer {token}",
            "User-Agent": "verity-bdd-release-provenance",
            "X-GitHub-Api-Version": "2022-11-28",
        },
    )
    opener = urllib.request.build_opener(_NoRedirect)
    try:
        with opener.open(request, timeout=20) as response:
            return _read_json_response(response)
    except urllib.error.HTTPError as error:
        raise ProvenanceError(f"GitHub API returned HTTP {error.code}") from error
    except (urllib.error.URLError, TimeoutError, OSError) as error:
        raise ProvenanceError("unable to read current main HEAD") from error


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
    opener = urllib.request.build_opener(_NoRedirect)
    try:
        with opener.open(request, timeout=20) as response:
            return _read_json_response(response)
    except urllib.error.HTTPError as error:
        raise ProvenanceError(f"Actions API returned HTTP {error.code}") from error
    except (urllib.error.URLError, TimeoutError, OSError) as error:
        raise ProvenanceError("unable to read successful CI provenance") from error


def _write_output(library_sha: str) -> None:
    output_path = os.environ.get("GITHUB_OUTPUT")
    if not output_path:
        raise ProvenanceError("GITHUB_OUTPUT is not set")
    with Path(output_path).open("a", encoding="utf-8") as output:
        output.write(f"library_sha={library_sha}\n")


def main() -> int:
    try:
        repository = os.environ["REPOSITORY"]
        default_branch = os.environ["DEFAULT_BRANCH"]
        library_sha = os.environ["LIBRARY_SHA"]
        api_url = os.environ["API_URL"]
        token = os.environ["ACTIONS_TOKEN"]
        head = fetch_main_head(api_url, token, repository)
        payload = fetch_manual_runs(
            api_url,
            token,
            repository,
            default_branch,
            library_sha,
        )
        validated_sha = validate_manual_candidate(
            head, payload, repository, default_branch, library_sha
        )
        _write_output(validated_sha)
        print(f"validated successful push CI provenance for {validated_sha}")
        return 0
    except (KeyError, OSError, json.JSONDecodeError, ProvenanceError) as error:
        print(f"release provenance rejected: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
