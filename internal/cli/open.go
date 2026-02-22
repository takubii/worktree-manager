package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takubii/git-worktree-opener/internal/config"
	"github.com/takubii/git-worktree-opener/internal/git"
	"github.com/takubii/git-worktree-opener/internal/opener"
)

func newOpenCmd(deps Dependencies) *cobra.Command {
	var openerName string
	var windowModeRaw string

	cmd := &cobra.Command{
		Use:   "open",
		Short: "Select and open an existing worktree",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := deps.Git.WorktreePrune(cmd.Context()); err != nil {
				return err
			}

			cfg := deps.Config.Load(cmd.Context())

			if !cmd.Flags().Changed("open") {
				openerName = cfg.Open.Default
			}
			if !cmd.Flags().Changed("window") {
				windowModeRaw = cfg.Open.Window
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

			activeWorktrees, prunable := splitPrunableWorktrees(worktrees)
			warnSkippedPrunableWorktrees(cmd.ErrOrStderr(), "wto open", prunable)
			if len(activeWorktrees) == 0 {
				return fmt.Errorf("no valid worktrees found after pruning stale entries. Run `wto new` to create one, then retry")
			}

			options := make([]string, len(activeWorktrees))
			for i, wt := range activeWorktrees {
				options[i] = formatWorktreeOption(wt)
			}

			selectedIndex, err := deps.Selector.Select(cmd.Context(), "Select a worktree to open:", options)
			if err != nil {
				return err
			}
			if selectedIndex < 0 || selectedIndex >= len(activeWorktrees) {
				return fmt.Errorf("invalid worktree selection index: %d", selectedIndex)
			}

			selected := activeWorktrees[selectedIndex]
			if err := deps.Opener.Open(cmd.Context(), openerName, selected.Path, windowMode); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&openerName, "open", config.DefaultOpenKind, "opener to use: "+config.SupportedOpenKindsText)
	cmd.Flags().StringVar(&windowModeRaw, "window", config.DefaultOpenWindow, "window behavior: "+config.SupportedWindowModesText)
	return cmd
}

func formatWorktreeOption(wt git.Worktree) string {
	state := "(detached)"
	if !wt.Detached && wt.Branch != "" {
		state = strings.TrimPrefix(wt.Branch, "refs/heads/")
	}

	return fmt.Sprintf("%s\t%s", state, wt.Path)
}
