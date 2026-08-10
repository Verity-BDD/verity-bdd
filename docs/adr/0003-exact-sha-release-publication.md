# ADR 0003: Manual SemVer release publication

- Status: Accepted
- Date: 2026-08-10

## Context

A release must publish an operator-chosen version of exactly the commit that passed push CI. Automatically releasing every `main` push removes the deliberate version decision, while deriving versions from commit messages or existing tags makes mutable repository conventions the source of release intent. Allowing a user-supplied or historical SHA can publish code other than the current reviewed branch head.

Published history can also become inconsistent when a run creates a tag but stops before the corresponding GitHub Release. Retries must complete the exact intended publication without moving tags or interpreting partial state as a new version. Prerelease promotion adds a legitimate case in which one tested commit is published first as a release candidate and later promoted without a source change.

The release credential must remain unavailable to checked-out repository code and read-time validation. Workflow tests can catch accidental contract drift, but because the workflow and tests can change together, branch protection and independent review—not in-repository tests—are the boundary against malicious changes.

## Decision

Releases are initiated only with `workflow_dispatch`. The operator must provide `version`; there is no push-triggered release and no SHA input. The workflow candidate is the dispatch commit, and it is eligible only when it is the current `main` HEAD and has exactly one completed, successful `push` run of the repository's `CI` workflow for that repository, branch, and SHA. The publisher rechecks that `main` still points to the candidate before creating the first public state, so an ordinary run fails if `main` advances during preparation.

The version input is the sole version intent. It must be canonical SemVer with a mandatory `v` prefix, may contain a SemVer prerelease, and may not contain build metadata. Numeric identifiers cannot contain forbidden leading zeroes. All releases share one globally increasing SemVer sequence; there are no parallel release lines. Commit messages never select or increment a version. Commits since the latest published release only produce deterministic release notes.

Published history is defined by non-draft GitHub Releases, including prereleases, and their matching tags. Draft releases are ignored. Publication chronology comes from valid, unique `published_at` timestamps rather than GitHub API list order, and versions must increase strictly in that chronology. Every published SemVer Release must have a matching tag, and every SemVer tag—including one with otherwise valid build metadata—must have a matching published Release. An orphan tag fails closed except when it is the one exactly requested version, resolves to the candidate SHA, and therefore represents the partial publication being retried. Duplicate, malformed, non-increasing, or conflicting history also fails.

Publishing the same SHA again is allowed only when the highest published release points to that SHA and is a prerelease, and the requested version is a higher prerelease or the final version of the same core `major.minor.patch`. Once that SHA has a final release, another version requires a different commit. The GitHub Release `prerelease` flag is derived from the validated version rather than entered separately.

Execution remains split across two security boundaries:

1. `build-release` has only `actions: read` and `contents: read`. It checks out the exact candidate with persisted credentials disabled, proves provenance, reads published history, validates version policy, creates deterministic notes, and computes the publisher checksum.
2. `publish-release` has only `contents: write`, performs no checkout or build, and has one action-free step. It downloads the publisher from the candidate SHA without redirects, verifies the prepared SHA-256 checksum, and runs it in a minimal isolated environment with the token stored in a private temporary file.

The publisher reconciles the exact requested tag and non-draft GitHub Release deterministically and without force updates. Exact existing state is idempotent success. A matching requested tag without its Release is completed as a partial retry, including after `main` has advanced; any mismatch in SHA, name, notes, draft state, or derived prerelease state fails closed. Publication is serialized.

Focused contract tests protect the chosen invariants against accidental drift: manual-only initiation, current-`main` and push-CI provenance, exact-SHA propagation, permissions, immutable actions, the minimal write step, deterministic reconciliation, checksum enforcement, and credential isolation. They are intentionally not a generic validator for the complete GitHub Actions YAML schema.

## Rejected alternatives

- **Automatic release on every `main` push.** Rejected because release timing and version remain an explicit operator decision.
- **Auto-bump from conventional commit messages.** Rejected because commit syntax is useful for notes but is not authoritative version intent.
- **Use a discovered tag as version intent.** Rejected because tags define published history or exact retry state; they do not choose the requested version.
- **Accept an arbitrary historical or user-supplied SHA.** Rejected because only current `main` HEAD with exact successful push-CI provenance is eligible.
- **Provide a dynamic version-suggestion UI.** Rejected because suggestions can become stale or appear authoritative; the operator supplies one explicit canonical version and server-side validation decides eligibility.
- **Maintain parallel release lines or release branches.** Rejected for now in favor of one globally increasing sequence with one current-`main` authority.
- **Allow SemVer build metadata.** Rejected because it does not affect SemVer precedence and would complicate the single ordered publication history.
- **Use a monolithic write-capable job.** Rejected because checkout, provenance queries, repository scripts, dependencies, and build tools would share the release credential boundary.
- **Treat every existing tag as success or every orphan as failure.** Rejected because the former accepts conflicts and the latter prevents deterministic completion of the exact requested partial retry.
- **Use a generic full-YAML validator.** Rejected because it duplicates GitHub Actions parsing, freezes harmless structure, and still cannot defend against coordinated malicious changes to tests and workflow.

## Consequences

An operator must choose and enter each version, including release candidates. Invalid versions, non-increasing versions, stale `main`, orphan tags, or conflicting publication state produce a failed run that requires the underlying state or request to be corrected; the workflow does not guess or repair conflicts.

A release candidate can be promoted on the same tested SHA—for example, `v1.2.0-rc.1` to `v1.2.0-rc.2` or `v1.2.0`—with explicit deterministic promotion notes. Any release after a final version requires a new commit, even if that commit only records the next releasable state.

Publication remains exact-SHA, fail-closed, non-forceful, deterministic, and idempotent. The read-only preparation/minimal publisher boundary, checksum verification, and credential isolation reduce the write-capable surface, while independent review remains necessary for security-sensitive workflow changes.
