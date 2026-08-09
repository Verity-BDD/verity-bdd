#!/usr/bin/env python3
"""Prepare deterministic metadata for an exact-SHA GitHub Release."""

from __future__ import annotations

import base64
import hashlib
import os
import re
import subprocess
import sys
from pathlib import Path
from typing import NamedTuple


SHA_RE = re.compile(r"^[0-9a-f]{40}$")
SEMVER_RE = re.compile(r"^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$")
BREAKING_SUBJECT_RE = re.compile(r"^[a-zA-Z][a-zA-Z0-9_-]*(?:\([^)]*\))?!:")
FEATURE_SUBJECT_RE = re.compile(r"^feat(?:\([^)]*\))?:")
PUBLISH_SCRIPT = Path(".github/scripts/release/publish_release.py")


class PrepareError(RuntimeError):
    pass


class ReleaseMetadata(NamedTuple):
    library_sha: str
    version: str
    body: str
    publish_script_sha256: str


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


def _reachable_versions(repo: Path, library_sha: str) -> list[tuple[tuple[int, int, int], str]]:
    versions: list[tuple[tuple[int, int, int], str]] = []
    for tag in _git(repo, "tag", "--list").stdout.splitlines():
        match = SEMVER_RE.fullmatch(tag)
        if match is None:
            continue
        ancestor = _git(
            repo, "merge-base", "--is-ancestor", f"refs/tags/{tag}", library_sha, check=False
        )
        if ancestor.returncode == 0:
            version = (int(match.group(1)), int(match.group(2)), int(match.group(3)))
            versions.append((version, tag))
        elif ancestor.returncode != 1:
            raise PrepareError(f"could not determine ancestry for tag {tag}")
    return sorted(versions)


def _commits(repo: Path, revision_range: str) -> list[tuple[str, str, str]]:
    hashes = _git(repo, "rev-list", "--reverse", revision_range).stdout.splitlines()
    result = []
    for commit_sha in hashes:
        subject = _git(repo, "show", "-s", "--format=%s", commit_sha).stdout.rstrip("\n")
        body = _git(repo, "show", "-s", "--format=%b", commit_sha).stdout.rstrip("\n")
        result.append((commit_sha, subject, body))
    return result


def _next_version(
    current: tuple[int, int, int] | None, commits: list[tuple[str, str, str]]
) -> str:
    if current is None:
        return "v0.1.0"
    major, minor, patch = current
    breaking = any(
        BREAKING_SUBJECT_RE.match(subject) or "BREAKING CHANGE:" in body
        for _, subject, body in commits
    )
    feature = any(FEATURE_SUBJECT_RE.match(subject) for _, subject, _ in commits)
    if breaking:
        return f"v{major + 1}.0.0"
    if feature:
        return f"v{major}.{minor + 1}.0"
    return f"v{major}.{minor}.{patch + 1}"


def _release_body(library_sha: str, commits: list[tuple[str, str, str]]) -> str:
    changes = "\n".join(f"- {subject} (`{commit_sha[:7]}`)" for commit_sha, subject, _ in commits)
    return (
        "## Changes\n\n"
        f"{changes}\n\n"
        "## Provenance\n\n"
        f"Library commit: `{library_sha}`\n"
    )


def build_release(repo: Path, library_sha: str) -> ReleaseMetadata | None:
    repo = repo.resolve()
    _exact_commit(repo, library_sha)
    versions = _reachable_versions(repo, library_sha)
    current: tuple[int, int, int] | None = None
    version: str | None = None
    exact_versions = [
        (version, tag)
        for version, tag in versions
        if _git(repo, "rev-parse", f"refs/tags/{tag}^{{commit}}").stdout.strip()
        == library_sha
    ]
    if len(exact_versions) > 1:
        raise PrepareError("selected commit has ambiguous release tags")
    if exact_versions:
        current, version = exact_versions[0]
        predecessors = [entry for entry in versions if entry not in exact_versions]
        revision_range = (
            f"refs/tags/{predecessors[-1][1]}..{library_sha}"
            if predecessors
            else library_sha
        )
    elif versions:
        current, latest_tag = versions[-1]
        version = None
        revision_range = f"refs/tags/{latest_tag}..{library_sha}"
    else:
        current = None
        version = None
        revision_range = library_sha
    commits = _commits(repo, revision_range)
    if not commits:
        return None

    script_path = repo / PUBLISH_SCRIPT
    try:
        script_digest = hashlib.sha256(script_path.read_bytes()).hexdigest()
    except OSError as error:
        raise PrepareError("publish script is unavailable") from error

    return ReleaseMetadata(
        library_sha=library_sha,
        version=version or _next_version(current, commits),
        body=_release_body(library_sha, commits),
        publish_script_sha256=script_digest,
    )


def _write_outputs(metadata: ReleaseMetadata | None) -> None:
    output_name = os.environ.get("GITHUB_OUTPUT")
    if not output_name:
        raise PrepareError("GITHUB_OUTPUT is not set")
    values = {
        "version": "",
        "release_body_b64": "",
        "publish_script_sha256": "",
    }
    if metadata is not None:
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
        metadata = build_release(Path.cwd(), os.environ["LIBRARY_SHA"])
        _write_outputs(metadata)
        if metadata is None:
            print("selected commit is already released; no publication required")
        else:
            print(f"prepared {metadata.version} for {metadata.library_sha}")
        return 0
    except (KeyError, OSError, PrepareError, subprocess.SubprocessError) as error:
        print(f"release preparation failed: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
