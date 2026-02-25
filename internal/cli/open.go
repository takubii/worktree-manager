package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takubii/git-worktree-opener/internal/config"
	"github.com/takubii/git-worktree-opener/internal/git"
	"github.com/takubii/git-worktree-opener/internal/opener"
)

func newOpenCmd(deps Dependencies) *cobra.Command {
	var openerName string
	var windowModeRaw string
	var targetBranch string
	var printCD bool
	var afterCommand string
	var noPrune bool

	cmd := &cobra.Command{
		Use:   "open",
		Short: "Select and open an existing worktree",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !noPrune {
				if err := deps.Git.WorktreePrune(cmd.Context()); err != nil {
					return err
				}
			}

			cfg := deps.Config.Load(cmd.Context())

			if !cmd.Flags().Changed("open") {
				openerName = cfg.Open.Default
			}
			if !cmd.Flags().Changed("window") {
				windowModeRaw = cfg.Open.Window
			}

			if err := validateExplicitOpenerAvailability(cmd, deps.LookPath, openerName); err != nil {
				return err
			}

			windowMode, err := opener.ParseWindowMode(windowModeRaw)
			if err != nil {
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
				return fmt.Errorf("no worktrees found. Create one first, then run `wto open`")
			}

			activeWorktrees, prunable, missing := splitUnavailableWorktreesForOpen(worktrees)
			warnSkippedPrunableWorktrees(cmd.ErrOrStderr(), "wto open", prunable)
			warnSkippedMissingWorktrees(cmd.ErrOrStderr(), "wto open", missing)
			if len(activeWorktrees) == 0 {
				return fmt.Errorf("no valid worktrees found after filtering stale/missing entries. Run `wto list` to inspect current state, then retry")
			}

			selected, err := selectWorktreeForOpen(cmd, deps, activeWorktrees, targetBranch)
			if err != nil {
				return err
			}
			if err := deps.Opener.Open(cmd.Context(), openerName, selected.Path, windowMode); err != nil {
				return err
			}
			if strings.TrimSpace(afterCommand) != "" {
				if err := deps.After.Run(cmd.Context(), afterCommand, selected.Path); err != nil {
					return err
				}
			}
			if printCD {
				hints := deps.Enter.FormatCDHints(selected.Path)
				for _, hint := range hints {
					if _, err := fmt.Fprintln(cmd.OutOrStdout(), hint); err != nil {
						return fmt.Errorf("failed to write cd hint output: %w", err)
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&openerName, "open", config.DefaultOpenKind, "opener to use: "+config.SupportedOpenKindsText)
	cmd.Flags().StringVar(&windowModeRaw, "window", config.DefaultOpenWindow, "window behavior: "+config.SupportedWindowModesText)
	cmd.Flags().StringVar(&targetBranch, "branch", "", "open worktree linked to this local branch")
	cmd.Flags().BoolVar(&printCD, "print-cd", false, "print cd command hints for the opened worktree")
	cmd.Flags().StringVar(&afterCommand, "after", "", "run a follow-up command after open (`{path}` is replaced with selected path)")
	cmd.Flags().BoolVar(&noPrune, "no-prune", false, "skip running git worktree prune --expire now before listing candidates")
	return cmd
}

func selectWorktreeForOpen(cmd *cobra.Command, deps Dependencies, worktrees []git.Worktree, targetBranch string) (git.Worktree, error) {
	targetBranch = normalizeBranch(targetBranch)
	if targetBranch != "" {
		return findWorktreeForOpenByBranch(worktrees, targetBranch)
	}

	options := make([]string, len(worktrees))
	for i, wt := range worktrees {
		options[i] = formatWorktreeOption(wt)
	}

	selectedIndex, err := deps.Selector.Select(cmd.Context(), "Select a worktree to open:", options)
	if err != nil {
		return git.Worktree{}, err
	}
	if selectedIndex < 0 || selectedIndex >= len(worktrees) {
		return git.Worktree{}, fmt.Errorf("invalid worktree selection index: %d", selectedIndex)
	}

	return worktrees[selectedIndex], nil
}

func findWorktreeForOpenByBranch(worktrees []git.Worktree, targetBranch string) (git.Worktree, error) {
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
		return git.Worktree{}, fmt.Errorf("multiple worktrees matched branch %q. Run `wto open` without --branch and choose the exact path", targetBranch)
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

func formatWorktreeOption(wt git.Worktree) string {
	state := "(detached)"
	if !wt.Detached && wt.Branch != "" {
		state = strings.TrimPrefix(wt.Branch, "refs/heads/")
	}

	return fmt.Sprintf("%s\t%s", state, wt.Path)
}
