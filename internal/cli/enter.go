package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/takubii/git-worktree-opener/internal/config"
	"github.com/takubii/git-worktree-opener/internal/git"
	"github.com/takubii/git-worktree-opener/internal/opener"
)

func newEnterCmd(deps Dependencies) *cobra.Command {
	var useShell bool
	var printCD bool
	var targetBranch string
	var tmuxModeRaw string

	cmd := &cobra.Command{
		Use:   "enter",
		Short: "Select a worktree for terminal workflows",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if useShell && printCD {
				return fmt.Errorf("`--shell` and `--print-cd` cannot be used together. Choose one mode and retry")
			}
			if cmd.Flags().Changed("tmux-mode") && !useShell {
				return fmt.Errorf("`--tmux-mode` can only be used with `--shell`. Set `--shell` and retry")
			}
			tmuxMode := opener.TmuxModeAuto
			if useShell {
				cfg := deps.Config.Load(cmd.Context())
				if !cmd.Flags().Changed("tmux-mode") {
					tmuxModeRaw = cfg.Tmux.Mode
				}
				parsedTmuxMode, err := opener.ParseTmuxMode(tmuxModeRaw)
				if err != nil {
					return err
				}
				tmuxMode = parsedTmuxMode
			}
			tracef(cmd.Context(), "enter: branch=%q shell=%v printCD=%v tmuxMode=%q", targetBranch, useShell, printCD, tmuxMode)

			tracef(cmd.Context(), "enter: running `git worktree prune --expire now`")
			if err := deps.Git.WorktreePrune(cmd.Context()); err != nil {
				return err
			}

			tracef(cmd.Context(), "enter: running `git worktree list --porcelain`")
			raw, err := deps.Git.WorktreeListPorcelain(cmd.Context())
			if err != nil {
				return err
			}

			worktrees, err := git.ParseWorktreeListPorcelain(raw)
			if err != nil {
				return fmt.Errorf("failed to parse git worktree output: %w", err)
			}
			tracef(cmd.Context(), "enter: parsed %d worktrees", len(worktrees))
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
				tracef(cmd.Context(), "enter: selecting by branch=%s", targetBranch)
				selected, err = findActiveWorktreeByBranch(activeWorktrees, targetBranch, "wto enter")
				if err != nil {
					return err
				}
			} else {
				tracef(cmd.Context(), "enter: selecting interactively from %d candidates", len(activeWorktrees))
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
				tracef(cmd.Context(), "enter: printing cd hints")
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
				tracef(cmd.Context(), "enter: starting subshell path=%s tmuxMode=%s", selected.Path, tmuxMode)
				return deps.Enter.StartShell(cmd.Context(), selected.Path, tmuxMode)
			}

			tracef(cmd.Context(), "enter: writing selected path output")
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), selected.Path); err != nil {
				return fmt.Errorf("failed to write selected path output: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&useShell, "shell", false, "start a subshell in the selected worktree")
	cmd.Flags().BoolVar(&printCD, "print-cd", false, "print cd command hints for the selected worktree")
	cmd.Flags().StringVar(&targetBranch, "branch", "", "enter worktree linked to this local branch")
	cmd.Flags().StringVar(&tmuxModeRaw, "tmux-mode", config.DefaultTmuxMode, "tmux mode: "+config.SupportedTmuxModesText)

	return cmd
}
