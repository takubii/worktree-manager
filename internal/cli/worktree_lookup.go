package cli

import (
	"fmt"
	"os"

	"github.com/takubii/git-worktree-opener/internal/git"
)

func findActiveWorktreeByBranch(worktrees []git.Worktree, targetBranch string, commandName string) (git.Worktree, error) {
	if targetBranch == "" {
		return git.Worktree{}, fmt.Errorf("branch name is empty. Set `--branch <name>` and retry")
	}

	matches := make([]git.Worktree, 0, 1)
	for _, wt := range worktrees {
		branch, ok := worktreeLocalBranch(wt)
		if !ok {
			continue
		}
		if branch == targetBranch {
			matches = append(matches, wt)
		}
	}

	if len(matches) == 0 {
		return git.Worktree{}, fmt.Errorf(
			"branch %q does not have a linked active worktree. Run `wto new %s` to create one, or run `wto list` to inspect available worktrees, then retry",
			targetBranch,
			targetBranch,
		)
	}
	if len(matches) > 1 {
		return git.Worktree{}, fmt.Errorf("multiple worktrees matched branch %q. Run `%s` without --branch and choose the exact path", targetBranch, commandName)
	}

	match := matches[0]
	if _, err := os.Stat(match.Path); err != nil {
		if os.IsNotExist(err) {
			return git.Worktree{}, fmt.Errorf("worktree path for branch %q does not exist locally: %s. Run `wto list` to inspect entries and `wto rm` to clean stale entries, then retry", targetBranch, match.Path)
		}
		return git.Worktree{}, fmt.Errorf("failed to inspect worktree path %q for branch %q: %w", match.Path, targetBranch, err)
	}

	return match, nil
}
