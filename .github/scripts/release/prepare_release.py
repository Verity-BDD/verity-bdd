#!/usr/bin/env python3
"""Prepare deterministic metadata for an exact-SHA GitHub Release."""

from __future__ import annotations

import base64
import hashlib
import json
import os
import re
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime
from pathlib import Path
from typing import Any, NamedTuple


SHA_RE = re.compile(r"^[0-9a-f]{40}$")
SEMVER_RE = re.compile(
    r"^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)"
    r"(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$"
)
RELEASE_TAG_RE = re.compile(
    r"^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)"
    r"(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?"
    r"(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$"
)
PUBLISH_SCRIPT = Path(".github/scripts/release/publish_release.py")
MAX_RESPONSE_BYTES = 8_388_608
RELEASES_PER_PAGE = 100
MAX_RELEASE_PAGES = 10


class PrepareError(RuntimeError):
    pass


class ReleaseMetadata(NamedTuple):
    library_sha: str
    version: str
    body: str
    publish_script_sha256: str


class _NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):  # noqa: ANN001
        return None


def fetch_releases(api_url: str, token: str, repository: str) -> list[dict[str, object]]:
    if api_url != "https://api.github.com":
        raise PrepareError("invalid GitHub API URL")
    if not token or "\n" in token or "\r" in token:
        raise PrepareError("invalid Actions token")
    if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repository):
        raise PrepareError("invalid repository identity")
    opener = urllib.request.build_opener(_NoRedirect)
    releases: list[dict[str, object]] = []
    for page in range(1, MAX_RELEASE_PAGES + 1):
        query = urllib.parse.urlencode(
            {"per_page": str(RELEASES_PER_PAGE), "page": str(page)}
        )
        request = urllib.request.Request(
            f"{api_url}/repos/{repository}/releases?{query}",
            headers={
                "Accept": "application/vnd.github+json",
                "Authorization": f"Bearer {token}",
                "User-Agent": "verity-bdd-release-preparation",
                "X-GitHub-Api-Version": "2022-11-28",
            },
        )
        try:
            with opener.open(request, timeout=20) as response:
                if response.status != 200:
                    raise PrepareError("GitHub Releases API did not return HTTP 200")
                raw = response.read(MAX_RESPONSE_BYTES + 1)
        except urllib.error.HTTPError as error:
            raise PrepareError(f"GitHub Releases API returned HTTP {error.code}") from error
        except (urllib.error.URLError, TimeoutError, OSError) as error:
            raise PrepareError("unable to read published release history") from error
        if not isinstance(raw, bytes) or len(raw) > MAX_RESPONSE_BYTES:
            raise PrepareError("GitHub Releases API response exceeded size limit")
        try:
            decoded: Any = json.loads(raw)
        except (json.JSONDecodeError, UnicodeDecodeError) as error:
            raise PrepareError("GitHub Releases API returned malformed JSON") from error
        if not isinstance(decoded, list) or not all(isinstance(item, dict) for item in decoded):
            raise PrepareError("GitHub Releases API response must be a list of objects")
        releases.extend(decoded)
        if len(decoded) < RELEASES_PER_PAGE:
            return releases
    raise PrepareError("published release history exceeds pagination limit")


def parse_version(version: str) -> tuple[object, ...]:
    match = SEMVER_RE.fullmatch(version)
    if match is None:
        raise PrepareError("version must be canonical SemVer with a v prefix")
    identifiers: list[tuple[int, object]] = []
    prerelease = match.group(4)
    if prerelease is not None:
        for identifier in prerelease.split("."):
            if identifier.isdecimal():
                if len(identifier) > 1 and identifier.startswith("0"):
                    raise PrepareError(
                        "numeric prerelease identifiers must not contain leading zeroes"
                    )
                identifiers.append((0, int(identifier)))
            else:
                identifiers.append((1, identifier))
    return (
        int(match.group(1)),
        int(match.group(2)),
        int(match.group(3)),
        1 if prerelease is None else 0,
        tuple(identifiers),
    )


def _is_release_tag(tag: str) -> bool:
    match = RELEASE_TAG_RE.fullmatch(tag)
    if match is None:
        return False
    prerelease = match.group(4)
    return prerelease is None or not any(
        identifier.isdecimal()
        and len(identifier) > 1
        and identifier.startswith("0")
        for identifier in prerelease.split(".")
    )


def _git(repo: Path, *args: str, check: bool = True) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", *args],
        cwd=repo,
        check=check,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )


def _exact_commit(repo: Path, library_sha: str) -> None:
    if not SHA_RE.fullmatch(library_sha):
        raise PrepareError("library SHA must be exactly 40 lowercase hexadecimal characters")
    result = _git(repo, "cat-file", "-t", library_sha, check=False)
    if result.returncode != 0 or result.stdout.strip() != "commit":
        raise PrepareError("library SHA does not identify a commit")
    resolved = _git(repo, "rev-parse", library_sha).stdout.strip()
    if resolved != library_sha:
        raise PrepareError("library SHA did not resolve exactly")


def _commits(repo: Path, revision_range: str) -> list[tuple[str, str]]:
    hashes = _git(repo, "rev-list", "--reverse", revision_range).stdout.splitlines()
    result = []
    for commit_sha in hashes:
        subject = _git(repo, "show", "-s", "--format=%s", commit_sha).stdout.rstrip("\n")
        result.append((commit_sha, subject))
    return result


def _release_body(
    library_sha: str,
    commits: list[tuple[str, str]],
    promoted_from: str | None = None,
) -> str:
    changes = "\n".join(f"- {subject} (`{commit_sha[:7]}`)" for commit_sha, subject in commits)
    promotion = (
        f"This release promotes {promoted_from} from the same tested commit.\n\n"
        if promoted_from is not None
        else ""
    )
    return (
        "## Changes\n\n"
        f"{promotion}{changes}\n\n"
        "## Provenance\n\n"
        f"Library commit: `{library_sha}`\n"
    )


def build_release(
    repo: Path, library_sha: str, requested_version: str, releases: list[dict[str, object]]
) -> ReleaseMetadata:
    repo = repo.resolve()
    _exact_commit(repo, library_sha)
    requested_key = parse_version(requested_version)
    published: list[tuple[tuple[object, ...], str, str]] = []
    chronology: list[tuple[datetime, tuple[object, ...], str]] = []
    published_tags: set[str] = set()
    for release in releases:
        if release.get("draft") is True:
            continue
        if release.get("draft") is not False:
            raise PrepareError("published release metadata is malformed")
        tag = release.get("tag_name")
        target = release.get("target_commitish")
        if not isinstance(tag, str) or not isinstance(target, str):
            raise PrepareError("published release metadata is malformed")
        version_key = parse_version(tag)
        if tag in published_tags:
            raise PrepareError(f"duplicate published release for {tag}")
        published_tags.add(tag)
        if release.get("prerelease") is not ("-" in tag):
            raise PrepareError(f"published release {tag} has conflicting prerelease flag")
        published_at = release.get("published_at")
        if not isinstance(published_at, str) or not re.fullmatch(
            r"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}"
            r"(?:\.[0-9]+)?(?:Z|[+-][0-9]{2}:[0-9]{2})",
            published_at,
        ):
            raise PrepareError(f"published release {tag} has malformed published_at")
        try:
            published_time = datetime.fromisoformat(published_at.replace("Z", "+00:00"))
        except ValueError as error:
            raise PrepareError(
                f"published release {tag} has malformed published_at"
            ) from error
        if published_time.utcoffset() is None:
            raise PrepareError(f"published release {tag} has malformed published_at")
        resolved = _git(repo, "rev-parse", f"refs/tags/{tag}^{{commit}}", check=False)
        if resolved.returncode != 0:
            raise PrepareError(f"published release {tag} does not have a matching tag")
        published.append((version_key, tag, resolved.stdout.strip()))
        chronology.append((published_time, version_key, tag))
    chronology.sort(key=lambda entry: entry[0])
    for previous, current in zip(chronology, chronology[1:]):
        if current[0] == previous[0]:
            raise PrepareError("published releases have duplicate publication timestamps")
        if current[1] <= previous[1]:
            raise PrepareError("published release versions regress by publication chronology")
    published.sort()
    local_semver_tags = {
        tag for tag in _git(repo, "tag", "--list").stdout.splitlines() if _is_release_tag(tag)
    }
    orphan_tags = local_semver_tags - published_tags
    retry_orphan = orphan_tags == {requested_version} and (
        _git(repo, "rev-parse", f"refs/tags/{requested_version}^{{commit}}", check=False).stdout.strip()
        == library_sha
    )
    if orphan_tags and not retry_orphan:
        raise PrepareError(f"orphan SemVer tags block release: {', '.join(sorted(orphan_tags))}")
    latest = published[-1] if published else None
    exact_latest = latest is not None and requested_key == latest[0]
    if latest is not None and requested_key < latest[0]:
        raise PrepareError("requested version must exceed the highest published version")
    if exact_latest:
        assert latest is not None
        if latest[2] != library_sha:
            raise PrepareError("highest published release does not match the candidate SHA")
    if (
        latest is not None
        and latest[2] != library_sha
        and any(entry[2] == library_sha for entry in published)
    ):
        raise PrepareError("candidate SHA was already published before the latest release")

    if exact_latest:
        predecessor = published[-2] if len(published) > 1 else None
    else:
        predecessor = latest
    promoted_from = None
    if predecessor is not None and predecessor[2] == library_sha:
        promoted_from = predecessor[1]
        if not exact_latest and predecessor[0][3] == 1:
            raise PrepareError(
                "the same commit cannot be promoted because the latest release is final"
            )
        if not exact_latest and requested_key[:3] != predecessor[0][:3]:
            raise PrepareError("the same commit may only promote the same core version")
    revision_range = (
        f"refs/tags/{predecessor[1]}..{library_sha}"
        if predecessor is not None
        else library_sha
    )
    commits = _commits(repo, revision_range)

    script_path = repo / PUBLISH_SCRIPT
    try:
        script_digest = hashlib.sha256(script_path.read_bytes()).hexdigest()
    except OSError as error:
        raise PrepareError("publish script is unavailable") from error

    return ReleaseMetadata(
        library_sha=library_sha,
        version=requested_version,
        body=_release_body(library_sha, commits, promoted_from),
        publish_script_sha256=script_digest,
    )


def _write_outputs(metadata: ReleaseMetadata) -> None:
    output_name = os.environ.get("GITHUB_OUTPUT")
    if not output_name:
        raise PrepareError("GITHUB_OUTPUT is not set")
    values = {
        "version": metadata.version,
        "release_body_b64": base64.b64encode(metadata.body.encode()).decode("ascii"),
        "publish_script_sha256": metadata.publish_script_sha256,
    }
    with Path(output_name).open("a", encoding="utf-8") as output:
        for key, value in values.items():
            output.write(f"{key}={value}\n")


def main() -> int:
    try:
        releases = fetch_releases(
            os.environ["API_URL"],
            os.environ["ACTIONS_TOKEN"],
            os.environ["REPOSITORY"],
        )
        metadata = build_release(
            Path.cwd(),
            os.environ["LIBRARY_SHA"],
            os.environ["VERSION"],
            releases,
        )
        _write_outputs(metadata)
        print(f"prepared {metadata.version} for {metadata.library_sha}")
        return 0
    except (KeyError, OSError, PrepareError, subprocess.SubprocessError) as error:
        print(f"release preparation failed: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
