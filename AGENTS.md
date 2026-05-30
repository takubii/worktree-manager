# AGENTS.md

Repository rules for human/AI contributors.

## Scope

- Applies to the whole repository unless a deeper `AGENTS.md` overrides it.

## Project Facts

- Project: `worktree-manager`
- Binary name: `wtm`
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
3. Keep `cmd/wtm/main.go` minimal (wiring + process exit handling only).
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
   - `go run ./cmd/wtm --help`
   - `go run ./cmd/wtm list` (run inside a Git repository, if applicable)
5. Keep CI parity with `.github/workflows/ci.yaml` (lint + tests + cross-platform build expectations).

## Dependency Rules

- Do not add new dependencies unless required by the task.

## Commit Rules

- Follow `CONTRIBUTING.md` (Conventional Commits).
- Use English commit messages.

## Documentation Rules

- Update `README.md` when user-facing behavior changes.

## Review Guidelines

- Prioritize review comments that have clear impact on correctness, safety, compatibility, diagnosability, or maintenance cost.
- Do not make purely stylistic or school-of-thought preferences blocking when `golangci-lint` or existing repository rules already cover them.

1. Check correctness and operational safety first.
   Review for wrong-repository/worktree/branch handling, destructive behavior, path handling mistakes, and OS-specific regressions.
2. Preserve public CLI contracts.
   Review for unintended changes to flags, defaults, stdout/stderr usage, exit codes, output formats, config precedence, and documented command behavior.
3. Require actionable error handling.
   Review for errors that hide root cause, lose context, ignore partial-failure paths, or fail to tell the user what to do next.
4. Avoid unnecessary coupling and maintenance burden.
   Review for changes that unnecessarily mix unrelated concerns, spread one behavior across too many places, or make future modification and testing materially harder.
5. Prefer simple, testable designs.
   Review for unnecessary abstraction, hidden coupling, avoidable new dependencies, or control flow that becomes harder to reason about than the problem requires.
6. Maintain readability at the point of change.
   Review for naming, function boundaries, branching, and data flow that make intent hard to recover, especially around command behavior and platform-specific code.
7. Expect tests for behavior changes and regressions.
   Review for missing or weak tests when behavior changes, bugs are fixed, or edge cases are added. Prefer environment-independent tests using stubs over host-installed tools.
8. Call out breaking-change risk explicitly.
   Review for impact on existing users, scripts, config files, machine-readable output, install/update flows, and README-documented workflows. Intentional changes should be documented.
