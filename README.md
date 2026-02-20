# git-worktree-opener

`git-worktree-opener` is a CLI tool whose binary name is `wto`.

## Phase 1 status

Implemented commands:

- `wto --help`
- `wto list`

`wto list` executes:

```text
git worktree list --porcelain
```

and prints the result as-is.

## Usage (Windows cmd.exe)

Show help:

```cmd
go run .\cmd\wto --help
```

List worktrees in the current repository:

```cmd
go run .\cmd\wto list
```

## Error example

If you run `wto list` outside a Git repository, the command exits with non-zero status and prints guidance to `stderr`.

```cmd
cd C:\Work\Repos\git-worktree-opener
go build -o wto.exe .\cmd\wto
cd C:\
C:\Work\Repos\git-worktree-opener\wto.exe list
echo %ERRORLEVEL%
```

Move to a Git repository directory and run the command again.
