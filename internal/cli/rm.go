package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takubii/git-worktree-opener/internal/git"
)

type deleteBranchMode string

const (
	deleteBranchNone  deleteBranchMode = "none"
	deleteBranchSafe  deleteBranchMode = "safe"
	deleteBranchForce deleteBranchMode = "force"
)

func newRmCmd(deps Dependencies) *cobra.Command {
	var removeForce bool
	var deleteBranchRaw string

	cmd := &cobra.Command{
		Use:   "rm [branch]",
		Short: "Remove a worktree and optionally delete its local branch",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deleteMode, err := parseDeleteBranchMode(deleteBranchRaw)
			if err != nil {
				return err
			}

			if removeForce && !cmd.Flags().Changed("delete-branch") {
				deleteMode = deleteBranchForce
			}

			raw, err := deps.Git.WorktreeListPorcelain(cmd.Context())
			if err != nil {
				return err
			}

			worktrees, err := git.ParseWorktreeListPorcelain(raw)
			if err != nil {
				return fmt.Errorf("failed to parse git worktree output: %w", err)
			}
			if len(worktrees) == 0 {
				return fmt.Errorf("no worktrees found. Create one first (for example, `wto new`), then retry")
			}

			selected, err := selectWorktreeForRemove(cmd.Context(), deps, args, worktrees)
			if err != nil {
				return err
			}

			if err := deps.Git.WorktreeRemove(cmd.Context(), selected.Path, removeForce); err != nil {
				return err
			}

			if deleteMode == deleteBranchNone {
				return nil
			}

			branch, ok := worktreeLocalBranch(selected)
			if !ok {
				return nil
			}

			return deps.Git.DeleteLocalBranch(cmd.Context(), branch, deleteMode == deleteBranchForce)
		},
	}

	cmd.Flags().BoolVar(&removeForce, "force", false, "force worktree removal; when --delete-branch is not set, branch deletion also becomes force")
	cmd.Flags().StringVar(&deleteBranchRaw, "delete-branch", string(deleteBranchSafe), "local branch deletion policy: none|safe|force")

	return cmd
}

func parseDeleteBranchMode(raw string) (deleteBranchMode, error) {
	switch deleteBranchMode(strings.ToLower(strings.TrimSpace(raw))) {
	case deleteBranchNone:
		return deleteBranchNone, nil
	case deleteBranchSafe:
		return deleteBranchSafe, nil
	case deleteBranchForce:
		return deleteBranchForce, nil
	default:
		return "", fmt.Errorf("invalid --delete-branch value %q. Use one of: none, safe, force", raw)
	}
}

func selectWorktreeForRemove(ctx context.Context, deps Dependencies, args []string, worktrees []git.Worktree) (git.Worktree, error) {
	if len(args) == 1 {
		targetBranch := normalizeBranch(args[0])
		if targetBranch == "" {
			return git.Worktree{}, fmt.Errorf("branch name is empty. Specify a branch and retry")
		}

		match, err := findWorktreeByBranch(worktrees, targetBranch)
		if err != nil {
			return git.Worktree{}, err
		}
		return match, nil
	}

	options := make([]string, len(worktrees))
	for i, wt := range worktrees {
		options[i] = formatWorktreeOption(wt)
	}

	selectedIndex, err := deps.Selector.Select(ctx, "Select a worktree to remove:", options)
	if err != nil {
		return git.Worktree{}, err
	}
	if selectedIndex < 0 || selectedIndex >= len(worktrees) {
		return git.Worktree{}, fmt.Errorf("invalid worktree selection index: %d", selectedIndex)
	}

	return worktrees[selectedIndex], nil
}

func findWorktreeByBranch(worktrees []git.Worktree, targetBranch string) (git.Worktree, error) {
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
		return git.Worktree{}, fmt.Errorf("branch %q does not have a linked worktree. Run `wto list` to inspect available worktrees, then retry", targetBranch)
	}
	if len(matches) > 1 {
		return git.Worktree{}, fmt.Errorf("multiple worktrees matched branch %q. Run `wto rm` without arguments and choose the exact path", targetBranch)
	}

	return matches[0], nil
}

func worktreeLocalBranch(wt git.Worktree) (string, bool) {
	if wt.Detached {
		return "", false
	}

	branch := strings.TrimSpace(wt.Branch)
	if branch == "" {
		return "", false
	}

	branch = normalizeBranch(branch)
	if branch == "" {
		return "", false
	}

	return branch, true
}
