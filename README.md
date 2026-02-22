# git-worktree-opener

`git-worktree-opener` is a CLI tool whose binary name is `wto`.

## Current status

Implemented commands:

- `wto --help`
- `wto list`
- `wto new`
- `wto open`
- `wto rm`
- `wto config init`
- `wto config show`

`wto list` executes:

```text
git worktree list --porcelain
```

and renders a readable table by default.

## Usage

Show help:

```sh
go run ./cmd/wto --help
```

List worktrees in the current repository:

```sh
go run ./cmd/wto list
go run ./cmd/wto list --format table
go run ./cmd/wto list --format raw
go run ./cmd/wto list --format json
```

`wto list` format behavior:

- default (`table`): `<marker> | BRANCH | STATUS | HEAD | PATH`
- `raw`: original `git worktree list --porcelain` output (backward-compatible)
- `json`: machine-readable array output

`<marker>` header is intentionally blank and shows `*` for the current working tree row.

`STATUS` values:

- `active`: normal entry and path exists
- `stale`: marked as `prunable` by Git
- `missing`: not prunable, but local path does not exist

Create a new worktree:

```sh
go run ./cmd/wto new
go run ./cmd/wto new feature/my-task
go run ./cmd/wto new feature/my-task --base main --open vscode
```

By default, `wto new` runs `git worktree prune --expire now`, then `git fetch origin --prune`, uses `main` as the base when creating a new branch, creates the worktree under `<repo-parent>/worktrees/<branch>`, and opens it with the `system` opener in a new window.

When running `wto new` without a branch argument:

- with `fzf`, you can type a branch name and press Enter to create it if no existing branch is selected
- with `promptui`, choose `Create a new branch` and enter the branch name
- the entered name is validated with `git check-ref-format --branch`

Select and open an existing worktree (uses `fzf` if installed, otherwise `promptui`, and finally numeric selection when interactive UI is unavailable):

```sh
go run ./cmd/wto open
```

By default, `wto open` prefers opening in a new window where supported.

Before listing candidates, `wto open` runs `git worktree prune --expire now` and skips entries marked as `prunable` (stale metadata).

Choose opener explicitly:

```sh
go run ./cmd/wto open --open vscode
go run ./cmd/wto open --open cursor
go run ./cmd/wto open --open vim
go run ./cmd/wto open --open system
```

Choose window behavior:

```sh
go run ./cmd/wto open --window new
go run ./cmd/wto open --window reuse
```

Current note: `--window` is applied to `system`, `vscode`, and `cursor`. `vim` currently uses best-effort behavior.

Remove an existing worktree:

```sh
go run ./cmd/wto rm
go run ./cmd/wto rm feature/my-task
go run ./cmd/wto rm feature/my-task --delete-branch none
go run ./cmd/wto rm feature/my-task --force
```

By default, `wto rm` removes the selected worktree and then safely deletes the local branch with `git branch -d`.

`wto rm` also shows stale entries (marked `prunable`) in selection as `[stale]`.
When a stale entry is selected, it is cleaned up via `git worktree prune --expire now`.
Selection rows use suffix status labels:

- `<branch>\t<path>\t[active]`
- `<branch>\t<path>\t[stale]`

- `--delete-branch none` skips branch deletion
- `--delete-branch safe` uses `git branch -d`
- `--delete-branch force` uses `git branch -D`
- `--force` forces `git worktree remove` and, when `--delete-branch` is not explicitly set, also switches branch deletion to force

Initialize global config:

```sh
go run ./cmd/wto config init
go run ./cmd/wto config init --force
```

Show effective config (merged defaults + global + repo override):

```sh
go run ./cmd/wto config show
```

Config is optional. If no config files exist, `wto` keeps using built-in defaults (0-config behavior).

Config precedence:

```text
flag > repo (.wtoconfig.json) > global (config.json) > built-in defaults
```

Config file locations:

- Global: `<os.UserConfigDir()>/git-worktree-opener/config.json`
- Repo override: `<repo-root>/.wtoconfig.json`

Supported config keys:

```json
{
  "remote": "origin",
  "baseBranch": "main",
  "worktreeDirTemplate": "{repoParent}/worktrees/{branch}",
  "open": {
    "default": "system",
    "window": "new"
  },
  "rm": {
    "deleteBranch": "safe"
  }
}
```

`worktreeDirTemplate` placeholders:

- `{repoParent}`
- `{repoRoot}`
- `{branch}`

If global/repo config is invalid (unknown keys or invalid values), `wto` prints a warning to `stderr` and continues with lower-priority values.

## Error example

If you run `wto list` outside a Git repository, the command exits with non-zero status and prints guidance to `stderr`.

```sh
cd <repo-root>
go build -o ./wto ./cmd/wto

cd <non-git-directory>
<repo-root>/wto list
echo $?
```

Move to a Git repository directory and run the command again.

On Windows `cmd.exe`, use `wto.exe` and `echo %ERRORLEVEL%`.
