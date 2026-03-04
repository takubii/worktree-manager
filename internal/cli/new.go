package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takubii/git-worktree-opener/internal/config"
	"github.com/takubii/git-worktree-opener/internal/git"
	"github.com/takubii/git-worktree-opener/internal/opener"
	"github.com/takubii/git-worktree-opener/internal/selector"
)

const (
	defaultBaseBranch = config.DefaultBaseBranch
	newOpenNone       = "none"
	branchLinkedLabel = " [worktree]"
)

type branchCandidate struct {
	Name    string
	Display string
}

func newNewCmd(deps Dependencies) *cobra.Command {
	var baseBranch string
	var openerName string
	var terminalProvider string
	var tmuxModeRaw string
	var noFetch bool
	var noPrune bool
	var outputRaw string

	cmd := &cobra.Command{
		Use:   "new [branch]",
		Short: "Create a new worktree",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputMode, err := parseOutputMode(outputRaw)
			if err != nil {
				return err
			}

			cfg := deps.Config.Load(cmd.Context())
			resolvedNoPrune := noPrune
			if !cmd.Flags().Changed("no-prune") {
				resolvedNoPrune = !cfg.New.Prune
			}
			resolvedNoFetch := noFetch
			if !cmd.Flags().Changed("no-fetch") {
				resolvedNoFetch = !cfg.New.Fetch
			}

			if !resolvedNoPrune {
				tracef(cmd.Context(), "new: running `git worktree prune --expire now`")
				if err := deps.Git.WorktreePrune(cmd.Context()); err != nil {
					return err
				}
				tracef(cmd.Context(), "new: prune completed")
			}

			remoteName := cfg.Remote
			if strings.TrimSpace(remoteName) == "" {
				return fmt.Errorf("remote name is empty. Set a valid `remote` in config and retry")
			}

			resolvedBaseBranch := strings.TrimSpace(baseBranch)
			if !cmd.Flags().Changed("base") {
				resolvedBaseBranch = cfg.BaseBranch
			}
			resolvedBaseBranch = strings.TrimSpace(resolvedBaseBranch)
			if resolvedBaseBranch == "" {
				return fmt.Errorf("base branch is empty. Set --base or `baseBranch` in config to a valid branch and retry")
			}

			resolvedOpener := strings.ToLower(strings.TrimSpace(openerName))
			resolvedTerminalProvider := strings.ToLower(strings.TrimSpace(terminalProvider))
			if !cmd.Flags().Changed("terminal-provider") {
				resolvedTerminalProvider = cfg.Open.TerminalProvider
			}
			if !cmd.Flags().Changed("tmux-mode") {
				tmuxModeRaw = cfg.Tmux.Mode
			}
			tracef(cmd.Context(), "new: opener=%q terminalProvider=%q tmuxMode=%q output=%s base=%q noFetch=%v noPrune=%v", openerName, terminalProvider, tmuxModeRaw, outputMode, baseBranch, noFetch, noPrune)
			if cmd.Flags().Changed("terminal-provider") && resolvedOpener != opener.KindTerminal {
				return fmt.Errorf("`--terminal-provider` can only be used with `--open terminal`. Set `--open terminal` and retry")
			}
			if cmd.Flags().Changed("tmux-mode") && resolvedOpener != opener.KindTerminal {
				return fmt.Errorf("`--tmux-mode` can only be used with `--open terminal`. Set `--open terminal` and retry")
			}
			tmuxMode, err := opener.ParseTmuxMode(tmuxModeRaw)
			if err != nil {
				return err
			}
			if resolvedOpener != newOpenNone {
				if err := validateExplicitOpenerAvailability(cmd, deps.LookPath, resolvedOpener); err != nil {
					return err
				}
			}

			targetBranch := ""
			if len(args) == 1 {
				targetBranch = args[0]
			}

			if !resolvedNoFetch {
				tracef(cmd.Context(), "new: running `git fetch %s --prune`", remoteName)
				if err := deps.Git.FetchPrune(cmd.Context(), remoteName); err != nil {
					return err
				}
				tracef(cmd.Context(), "new: fetch completed")
			}

			tracef(cmd.Context(), "new: resolving repository root and branches")
			repoRoot, err := deps.Git.RepoRoot(cmd.Context())
			if err != nil {
				return err
			}

			localBranches, err := deps.Git.LocalBranches(cmd.Context())
			if err != nil {
				return err
			}
			remoteBranches, err := deps.Git.RemoteBranches(cmd.Context(), remoteName)
			if err != nil {
				return err
			}

			resolvedBranch, startPoint, err := resolveTargetBranch(
				cmd,
				deps,
				targetBranch,
				resolvedBaseBranch,
				remoteName,
				localBranches,
				remoteBranches,
			)
			if err != nil {
				return err
			}
			tracef(cmd.Context(), "new: resolved branch=%s startPoint=%s", resolvedBranch, startPoint)

			worktreePath, err := config.RenderWorktreeDir(cfg.WorktreeDirTemplate, repoRoot, resolvedBranch)
			if err != nil {
				return err
			}

			if err := ensureWorktreePathAvailable(worktreePath); err != nil {
				return err
			}

			if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
				return fmt.Errorf("failed to create worktree parent directory: %w", err)
			}

			if err := deps.Git.WorktreeAdd(cmd.Context(), git.WorktreeAddParams{
				Path:       worktreePath,
				Branch:     resolvedBranch,
				StartPoint: startPoint,
			}); err != nil {
				return err
			}
			tracef(cmd.Context(), "new: worktree added path=%s", worktreePath)

			opened := false
			if resolvedOpener != newOpenNone {
				windowMode, err := opener.ParseWindowMode(cfg.Open.Window)
				if err != nil {
					return fmt.Errorf("invalid config open.window value: %w", err)
				}

				tracef(cmd.Context(), "new: invoking opener kind=%s terminalProvider=%s tmuxMode=%s path=%s window=%s", resolvedOpener, resolvedTerminalProvider, tmuxMode, worktreePath, windowMode)
				openResult, err := openPathWithResult(cmd.Context(), deps.Opener, resolvedOpener, worktreePath, windowMode, resolvedTerminalProvider, tmuxMode)
				if err != nil {
					return err
				}
				for _, warning := range openResult.Warnings {
					if _, warnErr := fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", warning); warnErr != nil {
						return fmt.Errorf("failed to write opener warning: %w", warnErr)
					}
				}
				opened = true
			}

			return writeCommandOutput(cmd.OutOrStdout(), outputMode, commandOutput{
				Command: "new",
				Path:    worktreePath,
				Branch:  resolvedBranch,
				Created: true,
				Opened:  opened,
			})
		},
	}

	cmd.Flags().StringVar(&baseBranch, "base", defaultBaseBranch, "base branch used when creating a new branch")
	cmd.Flags().StringVar(&openerName, "open", newOpenNone, "opener to use after creation: none|"+config.SupportedOpenKindsText)
	cmd.Flags().StringVar(&terminalProvider, "terminal-provider", config.DefaultOpenTerminalProvider, "terminal provider: "+config.SupportedTerminalProvidersText)
	cmd.Flags().StringVar(&tmuxModeRaw, "tmux-mode", config.DefaultTmuxMode, "tmux mode: "+config.SupportedTmuxModesText)
	cmd.Flags().BoolVar(&noFetch, "no-fetch", false, "skip running git fetch <remote> --prune before branch resolution")
	cmd.Flags().BoolVar(&noPrune, "no-prune", false, "skip running git worktree prune --expire now before processing")
	cmd.Flags().StringVar(&outputRaw, "output", string(outputModeNone), "output mode: "+supportedOutputModesText)

	return cmd
}

func resolveTargetBranch(
	cmd *cobra.Command,
	deps Dependencies,
	branchArg string,
	baseBranch string,
	remoteName string,
	localBranches []string,
	remoteBranches []string,
) (string, string, error) {
	branchArg = normalizeBranch(branchArg)

	localSet := asBranchSet(localBranches)
	remoteSet := asRemoteBranchSet(remoteBranches, remoteName)

	if branchArg == "" {
		linkedBranches, err := linkedWorktreeBranchSet(cmd, deps)
		if err != nil {
			tracef(cmd.Context(), "new: failed to resolve linked worktree branches for candidate labels: %v", err)
		}

		candidates := branchCandidates(localBranches, remoteBranches, remoteName, linkedBranches)
		if len(candidates) == 0 {
			return "", "", fmt.Errorf("no branches available. Create or fetch branches, then run `wto new` again")
		}
		tracef(cmd.Context(), "new: interactive branch selection from %d candidates", len(candidates))
		candidateDisplays := make([]string, 0, len(candidates))
		displayToBranch := make(map[string]string, len(candidates))
		for _, candidate := range candidates {
			candidateDisplays = append(candidateDisplays, candidate.Display)
			displayToBranch[candidate.Display] = candidate.Name
		}

		creator, supportsCreate := deps.Selector.(selector.SelectOrCreator)
		if supportsCreate {
			tracef(cmd.Context(), "new: selector supports create flow")
			result, err := creator.SelectOrCreate(
				cmd.Context(),
				"Select or enter a branch for the new worktree:",
				candidateDisplays,
			)
			if err != nil {
				return "", "", err
			}

			branchArg = normalizeBranch(result.Value)
			if !result.IsNew {
				if mapped, ok := displayToBranch[branchArg]; ok {
					branchArg = mapped
				}
			}
			if branchArg == "" {
				return "", "", fmt.Errorf("branch name is empty. Enter a branch name and retry")
			}

			if result.IsNew {
				if err := deps.Git.CheckBranchName(cmd.Context(), branchArg); err != nil {
					return "", "", err
				}
			}
		} else {
			tracef(cmd.Context(), "new: selector fallback select-only flow")
			selectedIndex, err := deps.Selector.Select(
				cmd.Context(),
				"Select a branch for the new worktree:",
				candidateDisplays,
			)
			if err != nil {
				return "", "", err
			}
			if selectedIndex < 0 || selectedIndex >= len(candidateDisplays) {
				return "", "", fmt.Errorf("invalid branch selection index: %d", selectedIndex)
			}
			branchArg = candidates[selectedIndex].Name
		}
	}

	if _, ok := localSet[branchArg]; ok {
		return branchArg, "", nil
	}
	if remoteBranch, ok := findRemoteBranchKey(branchArg, remoteName, remoteSet); ok {
		return branchArg, remoteName + "/" + remoteBranch, nil
	}

	baseBranch = normalizeBranch(baseBranch)
	if _, ok := localSet[baseBranch]; ok {
		return branchArg, baseBranch, nil
	}
	if remoteBaseBranch, ok := findRemoteBranchKey(baseBranch, remoteName, remoteSet); ok {
		return branchArg, remoteName + "/" + remoteBaseBranch, nil
	}

	return branchArg, baseBranch, nil
}

func ensureWorktreePathAvailable(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("worktree path already exists: %s. Remove it or choose another branch, then retry", path)
		}
		return fmt.Errorf("worktree path already exists as a file: %s. Remove it or choose another branch, then retry", path)
	}

	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return fmt.Errorf("failed to inspect worktree path %q: %w", path, err)
}

func asBranchSet(branches []string) map[string]struct{} {
	set := make(map[string]struct{}, len(branches))
	for _, branch := range branches {
		normalized := normalizeBranch(branch)
		if normalized == "" {
			continue
		}
		set[normalized] = struct{}{}
	}
	return set
}

func findRemoteBranchKey(branch string, remote string, remoteSet map[string]struct{}) (string, bool) {
	branch = normalizeBranch(branch)
	if branch == "" {
		return "", false
	}

	if _, ok := remoteSet[branch]; ok {
		return branch, true
	}

	remote = strings.TrimSpace(remote)
	if remote == "" {
		return "", false
	}

	prefixed := remote + "/"
	if !strings.HasPrefix(branch, prefixed) {
		return "", false
	}

	trimmed := strings.TrimPrefix(branch, prefixed)
	if _, ok := remoteSet[trimmed]; ok {
		return trimmed, true
	}

	return "", false
}

func asRemoteBranchSet(remoteBranches []string, remote string) map[string]struct{} {
	set := make(map[string]struct{}, len(remoteBranches))
	prefix := remote + "/"
	for _, branch := range remoteBranches {
		branch = strings.TrimSpace(branch)
		if !strings.HasPrefix(branch, prefix) {
			continue
		}

		short := strings.TrimPrefix(branch, prefix)
		if short == "" || short == "HEAD" {
			continue
		}
		set[short] = struct{}{}
	}
	return set
}

func branchCandidates(
	localBranches []string,
	remoteBranches []string,
	remote string,
	linkedBranches map[string]struct{},
) []branchCandidate {
	candidates := make([]branchCandidate, 0, len(localBranches)+len(remoteBranches))
	seen := make(map[string]struct{}, len(localBranches)+len(remoteBranches))

	for _, branch := range localBranches {
		normalized := normalizeBranch(branch)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		candidates = append(candidates, decorateBranchCandidate(normalized, linkedBranches))
	}

	prefix := remote + "/"
	for _, branch := range remoteBranches {
		branch = strings.TrimSpace(branch)
		if !strings.HasPrefix(branch, prefix) {
			continue
		}
		short := strings.TrimPrefix(branch, prefix)
		if short == "" || short == "HEAD" {
			continue
		}
		if _, ok := seen[short]; ok {
			continue
		}
		seen[short] = struct{}{}
		candidates = append(candidates, decorateBranchCandidate(short, linkedBranches))
	}

	return candidates
}

func decorateBranchCandidate(branch string, linkedBranches map[string]struct{}) branchCandidate {
	display := branch
	if _, linked := linkedBranches[branch]; linked {
		display += branchLinkedLabel
	}

	return branchCandidate{
		Name:    branch,
		Display: display,
	}
}

func linkedWorktreeBranchSet(cmd *cobra.Command, deps Dependencies) (map[string]struct{}, error) {
	raw, err := deps.Git.WorktreeListPorcelain(cmd.Context())
	if err != nil {
		return nil, err
	}

	worktrees, err := git.ParseWorktreeListPorcelain(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse git worktree output while building branch candidates: %w", err)
	}

	activeWorktrees, _ := splitPrunableWorktrees(worktrees)
	linked := make(map[string]struct{}, len(activeWorktrees))
	for _, wt := range activeWorktrees {
		branch, ok := worktreeLocalBranch(wt)
		if !ok {
			continue
		}
		linked[branch] = struct{}{}
	}

	return linked, nil
}
