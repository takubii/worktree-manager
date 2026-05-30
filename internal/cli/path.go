package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takubii/worktree-manager/internal/git"
)

const errNoValidWorktreesForPath = "no valid worktrees found after filtering stale/missing entries. Run `wtm list` to inspect current state, then retry"

func newPathCmd(deps Dependencies) *cobra.Command {
	var targetBranch string
	var outputRaw string

	cmd := &cobra.Command{
		Use:   "path",
		Short: "Select and print a worktree path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			outputMode, err := parsePathOutputMode(outputRaw)
			if err != nil {
				return err
			}

			tracef(cmd.Context(), "path: branch=%q output=%s", targetBranch, outputMode)
			tracef(cmd.Context(), "path: running `git worktree list --porcelain`")
			raw, err := deps.Git.WorktreeListPorcelain(cmd.Context())
			if err != nil {
				return err
			}

			worktrees, err := git.ParseWorktreeListPorcelain(raw)
			if err != nil {
				return fmt.Errorf("failed to parse git worktree output: %w", err)
			}
			tracef(cmd.Context(), "path: parsed %d worktrees", len(worktrees))
			if len(worktrees) == 0 {
				return fmt.Errorf("no worktrees found. Create one first, then run `wtm path`")
			}

			activeWorktrees, prunable, missing := splitUnavailableWorktreesForPath(worktrees)
			warnSkippedPrunableWorktrees(cmd.ErrOrStderr(), "wtm path", prunable)
			warnSkippedMissingWorktrees(cmd.ErrOrStderr(), "wtm path", missing)
			if len(activeWorktrees) == 0 {
				return errors.New(errNoValidWorktreesForPath)
			}

			selected, err := selectWorktreeForPath(cmd, deps, activeWorktrees, targetBranch)
			if err != nil {
				return err
			}

			branch := ""
			if value, ok := worktreeLocalBranch(selected); ok {
				branch = value
			}

			return writeCommandOutput(cmd.OutOrStdout(), outputMode, commandOutput{
				Command: "path",
				Path:    selected.Path,
				Branch:  branch,
			})
		},
	}

	cmd.Flags().StringVar(&targetBranch, "branch", "", "print the path for the worktree linked to this local branch")
	cmd.Flags().StringVar(&outputRaw, "output", string(outputModePath), "output mode: path|json")
	return cmd
}

func parsePathOutputMode(raw string) (outputMode, error) {
	mode := outputMode(strings.ToLower(strings.TrimSpace(raw)))
	switch mode {
	case outputModePath, outputModeJSON:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid --output value %q. Use one of: path|json", raw)
	}
}

func selectWorktreeForPath(cmd *cobra.Command, deps Dependencies, worktrees []git.Worktree, targetBranch string) (git.Worktree, error) {
	targetBranch = normalizeBranch(targetBranch)
	if targetBranch != "" {
		tracef(cmd.Context(), "path: selecting by branch=%s", targetBranch)
		return findActiveWorktreeByBranch(worktrees, targetBranch, "wtm path")
	}
	tracef(cmd.Context(), "path: selecting interactively")

	options := make([]string, len(worktrees))
	for i, wt := range worktrees {
		options[i] = formatWorktreeOption(wt)
	}

	selectedIndex, err := deps.Selector.Select(cmd.Context(), "Select a worktree path:", options)
	if err != nil {
		return git.Worktree{}, err
	}
	if selectedIndex < 0 || selectedIndex >= len(worktrees) {
		return git.Worktree{}, fmt.Errorf("invalid worktree selection index: %d", selectedIndex)
	}

	return worktrees[selectedIndex], nil
}

func formatWorktreeOption(wt git.Worktree) string {
	state := branchLabelUnknown
	if wt.Detached {
		state = branchLabelDetached
	} else if branch, ok := worktreeLocalBranch(wt); ok {
		state = branch
	}

	return fmt.Sprintf("%s\t%s", state, wt.Path)
}
