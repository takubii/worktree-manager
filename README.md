# git-worktree-opener

`git-worktree-opener` is a CLI tool whose binary name is `wto`.

## Current status

Implemented commands:

- `wto --help`
- `wto list`
- `wto new`
- `wto open`
- `wto rm`

`wto list` executes:

```text
git worktree list --porcelain
```

and prints the result as-is.

## Usage

Show help:

```sh
go run ./cmd/wto --help
```

List worktrees in the current repository:

```sh
go run ./cmd/wto list
```

Create a new worktree:

```sh
go run ./cmd/wto new
go run ./cmd/wto new feature/my-task
go run ./cmd/wto new feature/my-task --base main --open vscode
```

By default, `wto new` runs `git fetch origin --prune`, uses `main` as the base when creating a new branch, creates the worktree under `<repo-parent>/worktrees/<branch>`, and opens it with the `system` opener in a new window.

When running `wto new` without a branch argument:

- with `fzf`, you can type a branch name and press Enter to create it if no existing branch is selected
- with `promptui`, choose `Create a new branch` and enter the branch name
- the entered name is validated with `git check-ref-format --branch`

Select and open an existing worktree (uses `fzf` if installed, otherwise `promptui`, and finally numeric selection when interactive UI is unavailable):

```sh
go run ./cmd/wto open
```

By default, `wto open` prefers opening in a new window where supported.

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

- `--delete-branch none` skips branch deletion
- `--delete-branch safe` uses `git branch -d`
- `--delete-branch force` uses `git branch -D`
- `--force` forces `git worktree remove` and, when `--delete-branch` is not explicitly set, also switches branch deletion to force

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
