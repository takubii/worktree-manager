package git

import (
	"fmt"
	"strings"
)

// ParseWorktreeListPorcelain parses output from `git worktree list --porcelain`.
func ParseWorktreeListPorcelain(raw string) ([]Worktree, error) {
	lines := strings.Split(raw, "\n")
	worktrees := make([]Worktree, 0)

	var current *Worktree
	flushCurrent := func(lineNo int) error {
		if current == nil {
			return nil
		}
		if strings.TrimSpace(current.Path) == "" {
			return fmt.Errorf("invalid worktree entry near line %d: missing worktree path", lineNo)
		}
		worktrees = append(worktrees, *current)
		current = nil
		return nil
	}

	for i, rawLine := range lines {
		lineNo := i + 1
		line := strings.TrimRight(rawLine, "\r")

		if line == "" {
			if err := flushCurrent(lineNo); err != nil {
				return nil, err
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			if err := flushCurrent(lineNo); err != nil {
				return nil, err
			}

			current = &Worktree{
				Path: strings.TrimSpace(strings.TrimPrefix(line, "worktree ")),
			}
			continue
		}

		if current == nil {
			return nil, fmt.Errorf("invalid worktree list format near line %d: %s", lineNo, line)
		}

		switch {
		case strings.HasPrefix(line, "branch "):
			current.Branch = strings.TrimSpace(strings.TrimPrefix(line, "branch "))
		case line == "detached":
			current.Detached = true
		case strings.HasPrefix(line, "prunable"):
			current.Prunable = true
		default:
			// Intentionally ignore lines such as HEAD, bare, and locked.
		}
	}

	if err := flushCurrent(len(lines)); err != nil {
		return nil, err
	}

	return worktrees, nil
}
