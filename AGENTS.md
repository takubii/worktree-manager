# AGENTS.md

Repository rules for human/AI contributors.

## Scope

- Applies to the whole repository unless a deeper `AGENTS.md` overrides it.

## Project Facts

- Project: `git-worktree-opener`
- Binary name: `wto`
- Language: Go

## Core Rules

1. Keep changes small and task-focused (no drive-by refactors).
2. Prefer standard library + existing dependencies.
3. Use external `git` commands for Git operations (do not add libgit2 bindings).
4. Error messages must be actionable (tell the user what to do next).
5. Preserve CLI contracts (errors to stderr, non-zero exit code).

## Go Coding Rules

1. Use `golangci-lint` as the primary lint/format toolchain.
2. Prefer `golangci-lint fmt` for Go formatting (including import organization configured in `.golangci.yaml`).
3. Keep `cmd/wto/main.go` minimal (wiring + process exit handling only).
4. Keep CLI wiring in `internal/cli`.
5. Keep Git command execution in `internal/git`.
6. Prefer dependency injection via interfaces for testability.
7. Use `context.Context` for operations that may block or call external commands.
8. Keep tests environment-independent by stubbing external dependencies (`LookPath`, `Selector`, `Opener`, `Git`) instead of relying on host-installed tools.

## Testing Rules

1. Add or update tests for behavior changes.
2. Prefer unit tests near changed packages.
3. Before finalizing, run in this order:
   - `golangci-lint fmt`
   - `golangci-lint run`
   - `go test ./...`
4. For CLI-impacting changes, also smoke test:
   - `go run ./cmd/wto --help`
   - `go run ./cmd/wto list` (run inside a Git repository, if applicable)
5. Keep CI parity with `.github/workflows/ci.yaml` (lint + tests + cross-platform build expectations).

## Dependency Rules

- Do not add new dependencies unless required by the task.

## Commit Rules

- Follow `CONTRIBUTING.md` (Conventional Commits).
- Use English commit messages.

## Documentation Rules

- Update `README.md` when user-facing behavior changes.
