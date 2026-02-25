package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/takubii/git-worktree-opener/internal/git"
)

func newEnterCmd(deps Dependencies) *cobra.Command {
	var useShell bool
	var printCD bool
	var targetBranch string

	cmd := &cobra.Command{
		Use:   "enter",
		Short: "Select a worktree for terminal workflows",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if useShell && printCD {
				return fmt.Errorf("`--shell` and `--print-cd` cannot be used together. Choose one mode and retry")
			}

			if err := deps.Git.WorktreePrune(cmd.Context()); err != nil {
				return err
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
				return fmt.Errorf("no worktrees found. Create one first, then run `wto enter`")
			}

			activeWorktrees, prunable := splitPrunableWorktrees(worktrees)
			warnSkippedPrunableWorktrees(cmd.ErrOrStderr(), "wto enter", prunable)
			if len(activeWorktrees) == 0 {
				return fmt.Errorf("no valid worktrees found after pruning stale entries. Run `wto new` to create one, then retry")
			}

			targetBranch = normalizeBranch(targetBranch)
			var selected git.Worktree
			if targetBranch != "" {
				selected, err = findActiveWorktreeByBranch(activeWorktrees, targetBranch, "wto enter")
				if err != nil {
					return err
				}
			} else {
				options := make([]string, len(activeWorktrees))
				for i, wt := range activeWorktrees {
					options[i] = formatWorktreeOption(wt)
				}

				selectedIndex, err := deps.Selector.Select(cmd.Context(), "Select a worktree to enter:", options)
				if err != nil {
					return err
				}
				if selectedIndex < 0 || selectedIndex >= len(activeWorktrees) {
					return fmt.Errorf("invalid worktree selection index: %d", selectedIndex)
				}

				selected = activeWorktrees[selectedIndex]
			}
			if _, err := os.Stat(selected.Path); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("selected worktree path does not exist locally: %s. Run `wto list` to inspect entries, then retry", selected.Path)
				}
				return fmt.Errorf("failed to inspect selected worktree path %q: %w", selected.Path, err)
			}

			if printCD {
				hints := deps.Enter.FormatCDHints(selected.Path)
				if len(hints) == 0 {
					return fmt.Errorf("failed to generate cd hints for %s", selected.Path)
				}

				for _, hint := range hints {
					if _, err := fmt.Fprintln(cmd.OutOrStdout(), hint); err != nil {
						return fmt.Errorf("failed to write cd hint output: %w", err)
					}
				}
				return nil
			}

			if useShell {
				return deps.Enter.StartShell(cmd.Context(), selected.Path)
			}

			if _, err := fmt.Fprintln(cmd.OutOrStdout(), selected.Path); err != nil {
				return fmt.Errorf("failed to write selected path output: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&useShell, "shell", false, "start a subshell in the selected worktree")
	cmd.Flags().BoolVar(&printCD, "print-cd", false, "print cd command hints for the selected worktree")
	cmd.Flags().StringVar(&targetBranch, "branch", "", "enter worktree linked to this local branch")

	return cmd
}
