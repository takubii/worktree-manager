package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/takubii/git-worktree-opener/internal/git"
)

func splitPrunableWorktrees(worktrees []git.Worktree) ([]git.Worktree, []git.Worktree) {
	active := make([]git.Worktree, 0, len(worktrees))
	prunable := make([]git.Worktree, 0)

	for _, wt := range worktrees {
		if wt.Prunable {
			prunable = append(prunable, wt)
			continue
		}
		active = append(active, wt)
	}

	return active, prunable
}

func warnSkippedPrunableWorktrees(w io.Writer, commandName string, prunable []git.Worktree) {
	if len(prunable) == 0 || w == nil {
		return
	}

	paths := make([]string, 0, len(prunable))
	for _, wt := range prunable {
		paths = append(paths, wt.Path)
	}

	_, _ = fmt.Fprintf(
		w,
		"warning: skipped %d stale worktree entries in `%s` (marked `prunable`): %s\n",
		len(prunable),
		commandName,
		strings.Join(paths, ", "),
	)
}
