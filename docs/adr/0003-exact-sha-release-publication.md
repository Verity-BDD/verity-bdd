# ADR 0003: Exact-SHA release publication

- Status: Accepted
- Date: 2026-08-10

## Context

A release must publish exactly the commit that passed the repository's push CI. Using the current default branch, a floating ref, or an unverified manually supplied SHA at publication time can release code that was not tested. A write-capable job that also checks out source, runs third-party actions, builds artifacts, or retains checkout credentials gives more code than necessary access to the release token.

Release attempts can also stop after creating a tag but before creating or observing the corresponding GitHub Release. Retrying such a partial publication must reconcile the same intended state rather than select a new version, move a tag, or fail merely because the expected tag already exists.

Workflow tests can catch accidental configuration drift, but they are stored beside the workflow and can be changed by the same commit. They are not a security boundary against a malicious contributor. Branch protection and independent review own that boundary.

## Decision

Automatic releases accept only a completed, successful `push` run of the repository's `CI` workflow on the default branch. Manual releases require one explicit lowercase 40-hex commit SHA and independently prove that exactly one matching successful push-CI run exists for that workflow, repository, event, branch, and SHA.

Split release execution into two jobs:

1. `build-release` has only `actions: read` and `contents: read`. It selects and validates the candidate SHA, checks out that exact commit with persisted credentials disabled, verifies the checkout identity, proves CI provenance, and prepares deterministic release metadata.
2. `publish-release` depends on the build outputs and has only `contents: write`. It contains one action-free step and does not check out or build repository code. It downloads the publication script from the selected SHA over HTTPS without redirects, verifies its prepared SHA-256 checksum, and executes it with an isolated environment and a token file confined to a private temporary directory.

Publication is serialized and non-forceful. The publisher creates or reconciles the exact semantic-version tag and final GitHub Release for the selected SHA. Existing tags and releases are accepted only when their target and metadata match the deterministic expected state; ambiguous or conflicting state fails closed. Retries reconstruct the same release metadata and complete a valid partial publication.

All GitHub Actions use immutable commit references, checkout credentials are not persisted, and executable Go tools installed by CI use reviewed versions rather than `@latest`. The obsolete cross-repository documentation dispatch workflow is removed so release publication has no unrelated external write capability.

Workflow contract tests remain focused on accidental drift of the security invariants above: provenance gating, exact-SHA propagation, permissions, immutable actions, the single action-free write step, checksum failure propagation, credential isolation, and absence of floating publication targets. They do not freeze harmless workflow formatting, ordinary step order, coverage commands, artifact handling, or the full YAML key schema.

## Rejected alternatives

- **Publish from `main` or another floating ref.** Rejected because the ref can advance after CI and no longer identify the tested commit.
- **Trust any manually supplied SHA.** Rejected because syntactic SHA validation does not prove a successful push-CI run for that commit.
- **Build and publish in one write-capable job.** Rejected because checkout, dependencies, build tools, and third-party actions would share the release credential boundary.
- **Use a third-party release action in the write job.** Rejected to keep the write-capable execution surface minimal and auditable.
- **Persist checkout credentials or force-update release tags.** Rejected because both broaden write capability and can replace previously published history.
- **Treat an existing tag as either unconditional success or unconditional failure.** Rejected because it cannot safely distinguish an idempotent retry from conflicting or partially completed publication.
- **Freeze the complete workflow topology in a custom validator.** Rejected because it duplicates GitHub Actions parsing, couples tests to harmless YAML structure, and can block safer maintenance changes. Narrow executable contracts protect the decisions that matter without pretending in-repository tests can replace review.

## Consequences

Every automatic or manual release is bound to one exact commit with successful push-CI provenance. Read-only preparation is separated from the smallest practical write boundary, and publication retries are deterministic and idempotent across matching partial state.

The release process now depends on versioned repository scripts and their behavioral tests rather than a broad third-party release action. Security-sensitive workflow changes require independent review, while focused contract tests provide fast feedback for accidental drift without making routine CI maintenance rewrite a snapshot validator.

The workflow is intentionally specific to public GitHub authority and the repository's `CI` workflow identity. Changing the hosting authority, provenance workflow, permission boundary, or publication model requires revisiting this decision rather than silently widening the existing contract.
