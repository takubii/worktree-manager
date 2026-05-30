# Releasing

This document describes the maintainer workflow for automated releases.

## Overview

- CI (`.github/workflows/ci.yaml`) validates code quality and cross-platform builds.
- CD (`.github/workflows/cd.yaml`) creates release artifacts using GoReleaser.
- Release tags must match `v*` and must point to a commit contained in `main`.
- Tagged releases are created as **drafts** by default.

## Prerequisites

1. `main` is green in CI.
2. Version/tag to release is decided (for example: `v0.1.0`).
3. `README.md` and changelog-relevant docs are up to date.

## Standard Release Flow (Recommended)

1. Ensure local branch is up to date with `origin/main`.
2. Create and push a version tag:
   - `git tag vX.Y.Z`
   - `git push origin vX.Y.Z`
3. Wait for `CD` workflow to finish.
4. Open the generated draft release in GitHub.
5. Review and edit notes:
   - confirm generated Conventional Commit sections
   - add known limitations and migration notes when needed
6. Publish the draft release.

## Post-Release Smoke Checks (Recommended)

Run a quick install validation for each supported installer.

Windows PowerShell example:

```powershell
$env:WTM_VERSION = "vX.Y.Z" # set this to the tag being verified
iwr https://raw.githubusercontent.com/takubii/worktree-manager/main/scripts/install.ps1 -UseBasicParsing | iex
wtm --help
```

CLI smoke checks:

```sh
wtm create --help
wtm path --help
wtm remove --help
wtm doctor
```

Repository smoke checks:

```bat
wtm list
wtm path --branch main
```
```sh
wtm list
wtm path --branch main
```

## Manual CD Run (`workflow_dispatch`)

Use manual runs for snapshot checks or controlled release runs.

- `mode=snapshot`
  - Builds archives and checksums without publishing a GitHub release.
- `mode=release`
  - Requires `tag` input (existing remote tag).
  - `GORELEASER_CURRENT_TAG` is always set from the resolved tag in preflight (same behavior for push tag and workflow_dispatch).
  - Supports overrides:
    - `draft` (default: true)
    - `prerelease` (default: false)

## RC -> GA on the same commit

When creating GA from the same commit as an RC tag (for example, `v0.3.0-rc1` then `v0.3.0`):

1. Push the GA tag (`git push origin vX.Y.Z`), or run `workflow_dispatch` with `mode=release` and `tag=vX.Y.Z`.
2. Confirm preflight resolves the GA tag correctly.
3. Confirm GoReleaser logs show `GORELEASER_CURRENT_TAG=vX.Y.Z`.

If these checks pass, the workflow will publish assets for the GA tag rather than reusing RC tag metadata.

## Troubleshooting

### Tag rejected as "not contained in main"

Cause:
- tag points to a commit not reachable from `origin/main`.

Fix:
1. retag using a commit from `main`
2. push corrected tag

### Tag format validation failed

Cause:
- tag does not match `vMAJOR.MINOR.PATCH` (optional prerelease suffix supported).

Fix:
- use a valid tag, for example:
  - `v0.1.0`
  - `v0.1.1-rc1`

### Missing release permissions

Cause:
- workflow token cannot create/edit releases.

Fix:
- ensure `permissions.contents=write` in `cd.yaml`
- ensure repository Actions permissions allow workflow writes

### Unexpected changelog grouping

Cause:
- commit message does not follow Conventional Commit format.

Fix:
1. fix commit style in future commits (`CONTRIBUTING.md`)
2. manually edit release notes before publishing the draft
