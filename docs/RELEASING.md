# Releasing Verity-BDD

Releases are created manually by the GitHub Actions **Release** workflow. The workflow publishes the current `main` commit only; it does not choose a version, accept a SHA, or run automatically after a push.

The release model and security boundaries are recorded in [ADR 0003](adr/0003-exact-sha-release-publication.md).

## Start a release

1. Merge the intended changes to `main`.
2. Wait for the `CI` workflow triggered by that push to complete successfully.
3. Open **Actions → Release → Run workflow** in GitHub.
4. Select the `main` branch.
5. Enter the complete version, for example:
   - `v1.0.0-rc.1` for a release candidate
   - `v1.0.0` for a final release
6. Run the workflow and verify the resulting tag and GitHub Release.

Versions must be canonical SemVer with a `v` prefix. Prerelease identifiers are supported; build metadata such as `v1.0.0+build.1` is not. The requested version must normally be strictly greater than every published release, including prereleases. The project maintains one global version sequence rather than parallel release lines.

## Eligibility and publication rules

The workflow fails closed unless all of these conditions hold:

- The dispatch commit is the current remote `main` HEAD.
- Exactly one completed, successful push run of the repository's `CI` workflow exists for that commit on `main`.
- Published, non-draft releases have valid publication timestamps and form a strictly increasing SemVer sequence in publication order.
- Every published SemVer release has a matching tag, and no release-like SemVer tag is orphaned.
- The requested version and candidate SHA satisfy the retry and same-SHA rules below.

Draft releases are ignored when validating published history. Arbitrary malformed tags such as `vfoo` are not treated as release tags.

## Retries and release candidates

Publication is deterministic and idempotent:

- Rerunning the exact highest published version is allowed only when its tag resolves to the candidate SHA. The workflow reconstructs the same release notes and reconciles the exact existing release metadata.
- If an attempt created the requested tag at the candidate SHA but did not create the GitHub Release, rerun the same version to complete publication.
- Any other orphan SemVer tag blocks publication, including a valid SemVer tag with build metadata.
- A candidate SHA already used by an older release cannot be reused after a newer release was published from another SHA.

A prerelease may be promoted from the same tested commit only when the highest published release is on that SHA, is a prerelease, and the requested version is higher with the same `major.minor.patch` core. For example, `v1.0.0-rc.1` may become `v1.0.0-rc.2` or `v1.0.0`. After a final release, another version requires a new commit.

## Release notes

Release notes are generated deterministically from commit subjects since the predecessor release tag. They include abbreviated commit SHAs and the exact library commit. Same-SHA promotions also identify the prerelease being promoted. Commit message conventions do not select or increment the release version.

## Troubleshooting

- **Candidate is not current `main`:** pull or merge the intended changes, wait for `main` to settle, then dispatch the workflow from `main` again.
- **CI provenance is missing or ambiguous:** confirm the push-triggered `CI` run for the current `main` SHA completed successfully. Do not use a pull-request run as a substitute.
- **Version must exceed published history:** inspect all non-draft GitHub Releases, including prereleases, and choose a globally higher version. An exact older release cannot be rerun after history advances.
- **Published history is malformed or non-increasing:** correct the conflicting release metadata or publication state before retrying; the workflow will not guess an order.
- **Orphan tag:** if it is the exact requested tag at the candidate SHA, rerun that same version. Otherwise resolve the conflicting repository state before releasing; do not move a published tag.
- **Same-SHA promotion rejected:** ensure the highest release is a prerelease on the candidate SHA and the new version has the same core version. A final release or intervening release on another SHA requires a new commit.
- **Existing release conflicts:** compare its tag target, name, notes, draft state, and prerelease flag with the expected release. Publication never force-updates conflicting state.
