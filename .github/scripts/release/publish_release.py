#!/usr/bin/env python3
"""Idempotently publish a GitHub Release bound to one exact commit."""

from __future__ import annotations

import argparse
import base64
import binascii
import json
import os
import re
import stat
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any, Protocol


SHA_RE = re.compile(r"^[0-9a-f]{40}$")
VERSION_RE = re.compile(r"^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$")
MAX_RESPONSE_BYTES = 1_048_576
MAX_BODY_BYTES = 524_288
MAX_TAG_INDIRECTIONS = 5


class PublishError(RuntimeError):
    pass


class ApiError(PublishError):
    pass


class Client(Protocol):
    def request(
        self, method: str, path: str, payload: object | None = None
    ) -> tuple[int, Any]: ...


class _NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):  # noqa: ANN001
        return None


class GitHubClient:
    def __init__(self, api_url: str, token: str) -> None:
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
            raise PublishError("invalid GitHub API URL")
        if not token or "\n" in token or "\r" in token:
            raise PublishError("invalid release token")
        self._origin = api_url.rstrip("/")
        self._token = token
        self._opener = urllib.request.build_opener(_NoRedirect)

    def request(
        self, method: str, path: str, payload: object | None = None
    ) -> tuple[int, Any]:
        if not path.startswith("/") or "?" in path or "#" in path:
            raise ApiError("invalid API path")
        body = None
        if payload is not None:
            body = json.dumps(payload, separators=(",", ":")).encode()
        request = urllib.request.Request(
            self._origin + path,
            data=body,
            method=method,
            headers={
                "Accept": "application/vnd.github+json",
                "Authorization": f"Bearer {self._token}",
                "Content-Type": "application/json",
                "User-Agent": "verity-bdd-exact-release",
                "X-GitHub-Api-Version": "2022-11-28",
            },
        )
        try:
            with self._opener.open(request, timeout=20) as response:
                raw = response.read(MAX_RESPONSE_BYTES + 1)
                status_code = response.status
        except urllib.error.HTTPError as error:
            raw = error.read(MAX_RESPONSE_BYTES + 1)
            status_code = error.code
        except (urllib.error.URLError, TimeoutError, OSError) as error:
            raise ApiError("GitHub API transport failure") from error
        if len(raw) > MAX_RESPONSE_BYTES:
            raise ApiError("GitHub API response exceeded size limit")
        if not raw:
            decoded: Any = {}
        else:
            try:
                decoded = json.loads(raw)
            except json.JSONDecodeError as error:
                raise ApiError("GitHub API returned malformed JSON") from error
        return status_code, decoded


def _validate_inputs(
    repository: str, default_branch: str, library_sha: str, version: str, body: str
) -> None:
    if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repository):
        raise PublishError("invalid repository identity")
    if not re.fullmatch(r"[A-Za-z0-9._/-]+", default_branch):
        raise PublishError("invalid default branch")
    if not SHA_RE.fullmatch(library_sha):
        raise PublishError("invalid library SHA")
    if not VERSION_RE.fullmatch(version):
        raise PublishError("invalid release version")
    if not body or len(body.encode()) > MAX_BODY_BYTES:
        raise PublishError("invalid release body")


def _release_path(repository: str, version: str) -> str:
    return f"/repos/{repository}/releases/tags/{urllib.parse.quote(version, safe='')}"


def _tag_path(repository: str, version: str) -> str:
    return f"/repos/{repository}/git/ref/tags/{urllib.parse.quote(version, safe='')}"


def _expected_release(version: str, library_sha: str, body: str) -> dict[str, Any]:
    return {
        "tag_name": version,
        "target_commitish": library_sha,
        "name": f"Release {version}",
        "body": body,
        "draft": False,
        "prerelease": False,
    }


def _require_exact_release(
    value: Any, version: str, library_sha: str, body: str
) -> None:
    if not isinstance(value, dict):
        raise PublishError("release response must be an object")
    expected = _expected_release(version, library_sha, body)
    for field in ("tag_name", "target_commitish", "name", "body"):
        if value.get(field) != expected[field]:
            raise PublishError(f"existing release has conflicting {field}")
    if value.get("draft") is not False:
        raise PublishError("release must be explicitly non-draft")
    if value.get("prerelease") is not False:
        raise PublishError("release must be explicitly non-prerelease")


def _get_release(client: Client, repository: str, version: str) -> Any | None:
    status_code, value = client.request("GET", _release_path(repository, version))
    if status_code == 404:
        return None
    if status_code != 200:
        raise PublishError(f"release lookup returned HTTP {status_code}")
    return value


def _resolve_tag(
    client: Client, repository: str, version: str
) -> str | None:
    status_code, value = client.request("GET", _tag_path(repository, version))
    if status_code == 404:
        return None
    if status_code != 200 or not isinstance(value, dict):
        raise PublishError("tag lookup failed")
    if value.get("ref") != f"refs/tags/{version}":
        raise PublishError("tag lookup returned a different ref")
    target = value.get("object")
    for _ in range(MAX_TAG_INDIRECTIONS + 1):
        if not isinstance(target, dict):
            raise PublishError("tag target is malformed")
        object_type = target.get("type")
        object_sha = target.get("sha")
        if not SHA_RE.fullmatch(object_sha) if isinstance(object_sha, str) else True:
            raise PublishError("tag target SHA is malformed")
        if object_type == "commit":
            return object_sha
        if object_type != "tag":
            raise PublishError("tag target has unsupported object type")
        status_code, tag = client.request(
            "GET", f"/repos/{repository}/git/tags/{object_sha}"
        )
        if status_code != 200 or not isinstance(tag, dict):
            raise PublishError("annotated tag lookup failed")
        target = tag.get("object")
    raise PublishError("tag has excessive annotation indirection")


def _require_fresh_default_branch(
    client: Client, repository: str, default_branch: str, library_sha: str
) -> None:
    branch = urllib.parse.quote(default_branch, safe="")
    status_code, value = client.request(
        "GET", f"/repos/{repository}/git/ref/heads/{branch}"
    )
    target = value.get("object") if isinstance(value, dict) else None
    if (
        status_code != 200
        or not isinstance(target, dict)
        or target.get("type") != "commit"
        or target.get("sha") != library_sha
    ):
        raise PublishError("default branch advanced before release publication")


def _require_exact_tag(
    client: Client, repository: str, version: str, library_sha: str
) -> None:
    resolved = _resolve_tag(client, repository, version)
    if resolved != library_sha:
        raise PublishError("release tag does not resolve to the exact library SHA")


def publish(
    client: Client,
    repository: str,
    default_branch: str,
    library_sha: str,
    version: str,
    body: str,
) -> None:
    _validate_inputs(repository, default_branch, library_sha, version, body)
    existing_release = _get_release(client, repository, version)
    if existing_release is not None:
        _require_exact_release(existing_release, version, library_sha, body)
        _require_exact_tag(client, repository, version, library_sha)
        return

    existing_tag = _resolve_tag(client, repository, version)
    if existing_tag not in (None, library_sha):
        raise PublishError("release tag already targets another commit")

    payload = _expected_release(version, library_sha, body)
    # A matching tag proves an earlier publication attempt already created public
    # state. Reconcile its missing Release deterministically even if main advanced.
    # Freshness remains mandatory before creating the first tag/publication state.
    if existing_tag is None:
        _require_fresh_default_branch(client, repository, default_branch, library_sha)
    try:
        client.request("POST", f"/repos/{repository}/releases", payload)
    except ApiError:
        # The server may have committed the release before the response was lost.
        pass

    final_release = None
    for attempt in range(3):
        final_release = _get_release(client, repository, version)
        if final_release is not None:
            break
        if attempt < 2:
            time.sleep(1)
    if final_release is None:
        raise PublishError("release creation could not be reconciled")
    _require_exact_release(final_release, version, library_sha, body)
    _require_exact_tag(client, repository, version, library_sha)


def _decode_body(encoded: str) -> str:
    try:
        raw = base64.b64decode(encoded, validate=True)
        body = raw.decode("utf-8")
    except (binascii.Error, UnicodeDecodeError) as error:
        raise PublishError("invalid base64 release body") from error
    if len(raw) > MAX_BODY_BYTES:
        raise PublishError("release body exceeds size limit")
    return body


def _read_token(path: Path) -> str:
    info = path.stat()
    if not stat.S_ISREG(info.st_mode) or info.st_mode & 0o077:
        raise PublishError("token file must be a private regular file")
    return path.read_text(encoding="utf-8")


def _arguments() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--api-url", required=True)
    parser.add_argument("--default-branch", required=True)
    parser.add_argument("--library-sha", required=True)
    parser.add_argument("--release-body-b64", required=True)
    parser.add_argument("--repository", required=True)
    parser.add_argument("--token-file", required=True, type=Path)
    parser.add_argument("--version", required=True)
    return parser.parse_args()


def main() -> int:
    try:
        args = _arguments()
        body = _decode_body(args.release_body_b64)
        token = _read_token(args.token_file)
        client = GitHubClient(args.api_url, token)
        publish(
            client,
            args.repository,
            args.default_branch,
            args.library_sha,
            args.version,
            body,
        )
        print(f"verified {args.version} and its tag at {args.library_sha}")
        return 0
    except (OSError, PublishError) as error:
        print(f"release publication failed: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
