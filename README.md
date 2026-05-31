# worktree-manager

`wtm` is a CLI tool for managing Git worktrees.

It focuses on worktree lifecycle operations:

- list worktrees
- create worktrees
- print a selected worktree path
- remove worktrees and optionally delete local branches
- inspect config and environment health
- self-update from GitHub Releases

## Install

Latest release:

- [v0.6.0](https://github.com/takubii/worktree-manager/releases/tag/v0.6.0)

Linux/macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/takubii/worktree-manager/main/scripts/install.sh | sh
```

Windows PowerShell:

```powershell
iwr https://raw.githubusercontent.com/takubii/worktree-manager/main/scripts/install.ps1 -UseBasicParsing | iex
```

Environment variables:

- `WTM_VERSION=vX.Y.Z` installs a specific release tag
- `WTM_INSTALL_DIR=<path>` changes installation directory
- `WTM_SKIP_CHECKSUM=1` skips SHA256 verification

## Quick Start

```sh
wtm list
wtm create feature/my-task
wtm path
wtm path --branch feature/my-task
wtm remove feature/my-task
wtm doctor
```

## Commands

### `wtm list`

Print worktrees from `git worktree list --porcelain`.

```sh
wtm list
wtm list --format table
wtm list --format raw
wtm list --format json
```

Flags:

- `--format table|raw|json`

When `wtm list` sees stale (`prunable`) or missing-path entries, table and JSON formats keep stdout unchanged and print guidance to stderr. Raw format stays identical to `git worktree list --porcelain`.

### `wtm create`

Create a worktree for a local, remote, or new branch.

```sh
wtm create
wtm create feature/my-task
wtm create feature/my-task --base main
wtm create feature/my-task --dry-run
wtm create feature/my-task --no-bootstrap
wtm create feature/my-task --output path
wtm create feature/my-task --output json
```

Behavior:

- Runs `git worktree prune --expire now` unless disabled.
- Runs `git fetch origin --prune` unless disabled.
- Creates worktrees under `<repo-parent>/worktrees/<branch>` by default.
- Uses `main` as the default base branch for new branches.
- Runs configured `create.bootstrap` file copy and post-create hooks after the worktree is created.
- `--dry-run` prints planned create/bootstrap actions without pruning, fetching, creating a worktree, copying files, or running hooks.

Flags:

- `--base <branch>`
- `--dry-run`
- `--no-fetch`
- `--no-prune`
- `--no-bootstrap`
- `--output none|path|json`

### `wtm path`

Select an active worktree and print only its path.

```sh
wtm path
wtm path --branch feature/my-task
wtm path --output json
```

Behavior:

- Does not run `git worktree prune`.
- Skips stale (`prunable`) and missing local-path entries.
- Prints warnings for skipped entries to `stderr`.

Flags:

- `--branch <branch>`
- `--output path|json`

### `wtm remove`

Remove a worktree and optionally delete its local branch.

```sh
wtm remove
wtm remove feature/my-task
wtm remove feature/my-task --dry-run
wtm remove feature/my-task --delete-branch none
wtm remove feature/my-task --force
```

Behavior:

- Refuses to remove a target worktree if the current directory is inside it.
- Cleans empty parent directories after branch-path worktree removal.
- If the selected entry is stale, runs `git worktree prune --expire now`.

Flags:

- `--dry-run`
- `--force`
- `--delete-branch none|safe|force`

### `wtm config`

Inspect and initialize config.

```sh
wtm config init
wtm config init --force
wtm config show
wtm config path
wtm config path --json
```

Config precedence:

```text
flag > repo (.wtmconfig.json) > global (config.json) > built-in defaults
```

Paths:

- Global: `<os.UserConfigDir()>/worktree-manager/config.json`
- Repo override: `<repo-root>/.wtmconfig.json`

Example:

```json
{
  "remote": "origin",
  "baseBranch": "main",
  "worktreeDirTemplate": "{repoParent}/worktrees/{branch}",
  "create": {
    "fetch": true,
    "prune": true,
    "bootstrap": {
      "copyFiles": [
        {
          "from": ".env",
          "to": ".env",
          "overwrite": false,
          "required": false
        }
      ],
      "postCreate": [
        {
          "name": "install dependencies",
          "command": ["npm", "install"]
        },
        {
          "name": "build frontend",
          "command": ["npm", "run", "build"],
          "cwd": "frontend"
        }
      ]
    }
  },
  "remove": {
    "deleteBranch": "safe"
  }
}
```

`worktreeDirTemplate` placeholders:

- `{repoParent}`
- `{repoRoot}`
- `{branch}`

`create.bootstrap` is optional. When it is omitted, `wtm create` does not copy files or run post-create hooks. Use `wtm create --no-bootstrap` to skip configured bootstrap actions for one run.

Bootstrap file copy:

- `from` is an absolute path or a path relative to `{repoRoot}`.
- `to` is relative to the new `{worktree}` and must stay inside it.
- Only files are supported in this release; directories, glob patterns, and templates are not expanded.
- Existing destination files are skipped unless `overwrite` is `true`.
- Missing files warn and continue unless `required` is `true`.

Post-create hooks:

- `command` is an argv array and is executed without a shell.
- Hooks run in order after successful worktree creation and file copy.
- `cwd` defaults to `{worktree}`. Relative `cwd` values are resolved inside the new worktree.
- Hook execution stops at the first failure and returns a non-zero exit code.

Bootstrap placeholders:

- `{repoRoot}`
- `{worktree}`
- `{branch}`

Docker Compose setup can be modeled later with bootstrap templates and post-create hooks, but `wtm` does not provide Docker-specific commands.

### `wtm doctor`

Run environment diagnostics.

```sh
wtm doctor
```

Checks:

- `git` availability
- current repository availability
- global and repo config validity
- update prerequisites

### `wtm update`

Update `wtm` from GitHub Releases.

```sh
wtm update
wtm update --version vX.Y.Z
```

### `wtm version`

Print the current version.

```sh
wtm version
wtm --version
```

## Migration From `git-worktree-opener`

`v0.5.0` is the first release under the `worktree-manager` name.

- Replace `wto` with `wtm`.
- Replace `wto new` with `wtm create`.
- Replace `wto rm` with `wtm remove`.
- Replace editor or terminal launch workflows with `wtm path`, then pass the printed path to your shell/editor tooling.
- Recreate config with `wtm config init`; old `open` and `tmux` config fields are no longer supported.

## Troubleshooting

### `wtm list` fails outside a Git repository

Run `wtm` inside a Git repository or linked worktree.

### `wtm path` finds no valid worktrees

Run `wtm list` to inspect stale or missing entries. Use `wtm remove <branch>` or interactive `wtm remove` to clean stale metadata or remove registered worktree entries when appropriate.

### `wtm remove` refuses to remove

Move to another directory outside the target worktree, then run `wtm remove` again.

### `wtm update` fails

Run `wtm doctor`, then verify network access, the requested release tag, and checksum availability.
