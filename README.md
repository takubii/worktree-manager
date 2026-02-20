# git-worktree-opener

`git-worktree-opener` is a CLI tool whose binary name is `wto`.

## Current status

Implemented commands:

- `wto --help`
- `wto list`
- `wto open`

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

Select and open an existing worktree (uses `fzf` if installed, otherwise numeric selection):

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
