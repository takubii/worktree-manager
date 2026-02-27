package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takubii/git-worktree-opener/internal/config"
	"github.com/takubii/git-worktree-opener/internal/git"
	"github.com/takubii/git-worktree-opener/internal/opener"
)

const errNoValidWorktreesForOpen = "no valid worktrees found after filtering stale/missing entries. Run `wto list` to inspect current state, then retry"

func newOpenCmd(deps Dependencies) *cobra.Command {
	var openerName string
	var windowModeRaw string
	var targetBranch string
	var printCD bool
	var afterCommand string
	var noPrune bool
	var outputRaw string

	cmd := &cobra.Command{
		Use:   "open",
		Short: "Select and open an existing worktree",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			outputMode, err := parseOutputMode(outputRaw)
			if err != nil {
				return err
			}
			if printCD && outputMode != outputModeNone {
				return fmt.Errorf("`--print-cd` and `--output` cannot be used together. Use one mode and retry")
			}
			tracef(cmd.Context(), "open: branch=%q opener=%q window=%q output=%s printCD=%v noPrune=%v", targetBranch, openerName, windowModeRaw, outputMode, printCD, noPrune)

			cfg := deps.Config.Load(cmd.Context())
			resolvedNoPrune := noPrune
			if !cmd.Flags().Changed("no-prune") {
				resolvedNoPrune = !cfg.Open.Prune
			}

			if !resolvedNoPrune {
				tracef(cmd.Context(), "open: running `git worktree prune --expire now`")
				if err := deps.Git.WorktreePrune(cmd.Context()); err != nil {
					return err
				}
				tracef(cmd.Context(), "open: prune completed")
			}

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

			tracef(cmd.Context(), "open: running `git worktree list --porcelain`")
			raw, err := deps.Git.WorktreeListPorcelain(cmd.Context())
			if err != nil {
				return err
			}

			worktrees, err := git.ParseWorktreeListPorcelain(raw)
			if err != nil {
				return fmt.Errorf("failed to parse git worktree output: %w", err)
			}
			tracef(cmd.Context(), "open: parsed %d worktrees", len(worktrees))
			if len(worktrees) == 0 {
				return fmt.Errorf("no worktrees found. Create one first, then run `wto open`")
			}

			activeWorktrees, prunable, missing := splitUnavailableWorktreesForOpen(worktrees)
			warnSkippedPrunableWorktrees(cmd.ErrOrStderr(), "wto open", prunable)
			warnSkippedMissingWorktrees(cmd.ErrOrStderr(), "wto open", missing)
			if len(activeWorktrees) == 0 {
				return errors.New(errNoValidWorktreesForOpen)
			}
			tracef(cmd.Context(), "open: active candidates=%d", len(activeWorktrees))

			selected, err := selectWorktreeForOpen(cmd, deps, activeWorktrees, targetBranch)
			if err != nil {
				return err
			}
			tracef(cmd.Context(), "open: invoking opener kind=%s path=%s window=%s", openerName, selected.Path, windowMode)
			if err := deps.Opener.Open(cmd.Context(), openerName, selected.Path, windowMode); err != nil {
				return err
			}
			if strings.TrimSpace(afterCommand) != "" {
				tracef(cmd.Context(), "open: running after command")
				if err := deps.After.Run(cmd.Context(), afterCommand, selected.Path); err != nil {
					return err
				}
			}
			if printCD {
				tracef(cmd.Context(), "open: printing cd hints")
				hints := deps.Enter.FormatCDHints(selected.Path)
				for _, hint := range hints {
					if _, err := fmt.Fprintln(cmd.OutOrStdout(), hint); err != nil {
						return fmt.Errorf("failed to write cd hint output: %w", err)
					}
				}
			}

			branch := ""
			if value, ok := worktreeLocalBranch(selected); ok {
				branch = value
			}

			return writeCommandOutput(cmd.OutOrStdout(), outputMode, commandOutput{
				Command: "open",
				Path:    selected.Path,
				Branch:  branch,
				Created: false,
				Opened:  true,
			})
		},
	}

	cmd.Flags().StringVar(&openerName, "open", config.DefaultOpenKind, "opener to use: "+config.SupportedOpenKindsText)
	cmd.Flags().StringVar(&windowModeRaw, "window", config.DefaultOpenWindow, "window behavior: "+config.SupportedWindowModesText)
	cmd.Flags().StringVar(&targetBranch, "branch", "", "open worktree linked to this local branch")
	cmd.Flags().BoolVar(&printCD, "print-cd", false, "print cd command hints for the opened worktree")
	cmd.Flags().StringVar(&afterCommand, "after", "", "run a follow-up command after open (`{path}` is replaced with selected path)")
	cmd.Flags().BoolVar(&noPrune, "no-prune", false, "skip running git worktree prune --expire now before listing candidates")
	cmd.Flags().StringVar(&outputRaw, "output", string(outputModeNone), "output mode: "+supportedOutputModesText)
	return cmd
}

func selectWorktreeForOpen(cmd *cobra.Command, deps Dependencies, worktrees []git.Worktree, targetBranch string) (git.Worktree, error) {
	targetBranch = normalizeBranch(targetBranch)
	if targetBranch != "" {
		tracef(cmd.Context(), "open: selecting by branch=%s", targetBranch)
		return findActiveWorktreeByBranch(worktrees, targetBranch, "wto open")
	}
	tracef(cmd.Context(), "open: selecting interactively")

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

func formatWorktreeOption(wt git.Worktree) string {
	state := branchLabelUnknown
	if wt.Detached {
		state = branchLabelDetached
	} else if branch, ok := worktreeLocalBranch(wt); ok {
		state = branch
	}

	return fmt.Sprintf("%s\t%s", state, wt.Path)
}
