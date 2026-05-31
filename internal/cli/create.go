package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takubii/worktree-manager/internal/config"
	"github.com/takubii/worktree-manager/internal/git"
	"github.com/takubii/worktree-manager/internal/selector"
)

const (
	defaultBaseBranch = config.DefaultBaseBranch
	branchLinkedLabel = " [worktree]"
)

type branchCandidate struct {
	Name    string
	Display string
}

func newCreateCmd(deps Dependencies) *cobra.Command {
	var baseBranch string
	var noFetch bool
	var noPrune bool
	var noBootstrap bool
	var dryRun bool
	var outputRaw string

	cmd := &cobra.Command{
		Use:   "create [branch]",
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
				resolvedNoPrune = !cfg.Create.Prune
			}
			resolvedNoFetch := noFetch
			if !cmd.Flags().Changed("no-fetch") {
				resolvedNoFetch = !cfg.Create.Fetch
			}

			if dryRun {
				tracef(cmd.Context(), "create: dry-run mode skips git worktree prune")
			} else if !resolvedNoPrune {
				tracef(cmd.Context(), "create: running `git worktree prune --expire now`")
				if err := deps.Git.WorktreePrune(cmd.Context()); err != nil {
					return err
				}
				tracef(cmd.Context(), "create: prune completed")
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

			tracef(cmd.Context(), "create: output=%s base=%q noFetch=%v noPrune=%v", outputMode, baseBranch, noFetch, noPrune)

			targetBranch := ""
			if len(args) == 1 {
				targetBranch = args[0]
			}

			if dryRun {
				tracef(cmd.Context(), "create: dry-run mode skips git fetch")
			} else if !resolvedNoFetch {
				tracef(cmd.Context(), "create: running `git fetch %s --prune`", remoteName)
				if err := deps.Git.FetchPrune(cmd.Context(), remoteName); err != nil {
					return err
				}
				tracef(cmd.Context(), "create: fetch completed")
			}

			tracef(cmd.Context(), "create: resolving repository root and branches")
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
			tracef(cmd.Context(), "create: resolved branch=%s startPoint=%s", resolvedBranch, startPoint)

			worktreePath, err := config.RenderWorktreeDir(cfg.WorktreeDirTemplate, repoRoot, resolvedBranch)
			if err != nil {
				return err
			}

			if err := ensureWorktreePathAvailable(worktreePath); err != nil {
				return err
			}

			bootstrapCtx := createBootstrapContext{
				repoRoot:     repoRoot,
				worktreePath: worktreePath,
				branch:       resolvedBranch,
				dryRun:       dryRun,
			}
			bootstrapEnabled := hasBootstrapActions(cfg.Create.Bootstrap) && !noBootstrap

			if dryRun {
				return writeCreateDryRunPlan(
					cmd.OutOrStdout(),
					resolvedNoPrune,
					resolvedNoFetch,
					remoteName,
					resolvedBranch,
					startPoint,
					worktreePath,
					cfg.Create.Bootstrap,
					bootstrapEnabled,
					bootstrapCtx,
				)
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
			tracef(cmd.Context(), "create: worktree added path=%s", worktreePath)

			if bootstrapEnabled {
				tracef(cmd.Context(), "create: running bootstrap actions")
				if err := runCreateBootstrap(
					cmd.Context(),
					cfg.Create.Bootstrap,
					bootstrapCtx,
					deps.CommandRunner,
					cmd.OutOrStdout(),
					cmd.ErrOrStderr(),
				); err != nil {
					return err
				}
			} else if noBootstrap && hasBootstrapActions(cfg.Create.Bootstrap) {
				tracef(cmd.Context(), "create: bootstrap skipped by --no-bootstrap")
			}

			return writeCommandOutput(cmd.OutOrStdout(), outputMode, commandOutput{
				Command: "create",
				Path:    worktreePath,
				Branch:  resolvedBranch,
				Created: true,
			})
		},
	}

	cmd.Flags().StringVar(&baseBranch, "base", defaultBaseBranch, "base branch used when creating a new branch")
	cmd.Flags().BoolVar(&noFetch, "no-fetch", false, "skip running git fetch <remote> --prune before branch resolution")
	cmd.Flags().BoolVar(&noPrune, "no-prune", false, "skip running git worktree prune --expire now before processing")
	cmd.Flags().BoolVar(&noBootstrap, "no-bootstrap", false, "skip configured create.bootstrap copy and post-create actions")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print planned create and bootstrap actions without changing files or running commands")
	cmd.Flags().StringVar(&outputRaw, "output", string(outputModeNone), "output mode: "+supportedOutputModesText)

	return cmd
}

func writeCreateDryRunPlan(
	w io.Writer,
	noPrune bool,
	noFetch bool,
	remoteName string,
	branch string,
	startPoint string,
	worktreePath string,
	bootstrap config.Bootstrap,
	bootstrapEnabled bool,
	bootstrapCtx createBootstrapContext,
) error {
	if _, err := fmt.Fprintln(w, "dry-run: planned create actions (no changes made)"); err != nil {
		return fmt.Errorf("failed to write dry-run output: %w", err)
	}
	if !noPrune {
		if _, err := fmt.Fprintln(w, "- git worktree prune --expire now"); err != nil {
			return fmt.Errorf("failed to write dry-run output: %w", err)
		}
	}
	if !noFetch {
		if _, err := fmt.Fprintf(w, "- git fetch %s --prune\n", remoteName); err != nil {
			return fmt.Errorf("failed to write dry-run output: %w", err)
		}
	}
	addCommand := "git worktree add"
	if strings.TrimSpace(startPoint) != "" {
		addCommand += fmt.Sprintf(" -b %s %q %s", branch, worktreePath, startPoint)
	} else {
		addCommand += fmt.Sprintf(" %q %s", worktreePath, branch)
	}
	if _, err := fmt.Fprintf(w, "- %s\n", addCommand); err != nil {
		return fmt.Errorf("failed to write dry-run output: %w", err)
	}
	if !hasBootstrapActions(bootstrap) {
		return nil
	}
	if !bootstrapEnabled {
		if _, err := fmt.Fprintln(w, "- skip configured bootstrap actions (--no-bootstrap)"); err != nil {
			return fmt.Errorf("failed to write dry-run output: %w", err)
		}
		return nil
	}
	for _, action := range bootstrap.CopyFiles {
		source, err := resolveBootstrapSourcePath(action.From, bootstrapCtx)
		if err != nil {
			return err
		}
		destination, err := resolveWorktreeContainedPath(action.To, bootstrapCtx)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "- copy %q -> %q\n", source, destination); err != nil {
			return fmt.Errorf("failed to write dry-run output: %w", err)
		}
	}
	for _, action := range bootstrap.PostCreate {
		command, err := renderCommandArgs(action.Command, bootstrapCtx)
		if err != nil {
			return err
		}
		cwd, err := resolveHookCWD(action.CWD, bootstrapCtx)
		if err != nil {
			return err
		}
		label := strings.TrimSpace(action.Name)
		if label == "" {
			label = strings.Join(command, " ")
		}
		if _, err := fmt.Fprintf(w, "- run hook %q in %q: %s\n", label, cwd, strings.Join(command, " ")); err != nil {
			return fmt.Errorf("failed to write dry-run output: %w", err)
		}
	}
	return nil
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
			tracef(cmd.Context(), "create: failed to resolve linked worktree branches for candidate labels: %v", err)
		}

		candidates := branchCandidates(localBranches, remoteBranches, remoteName, linkedBranches)
		if len(candidates) == 0 {
			return "", "", fmt.Errorf("no branches available. Create or fetch branches, then run `wtm create` again")
		}
		tracef(cmd.Context(), "create: interactive branch selection from %d candidates", len(candidates))
		candidateDisplays := make([]string, 0, len(candidates))
		displayToBranch := make(map[string]string, len(candidates))
		for _, candidate := range candidates {
			candidateDisplays = append(candidateDisplays, candidate.Display)
			displayToBranch[candidate.Display] = candidate.Name
		}

		creator, supportsCreate := deps.Selector.(selector.SelectOrCreator)
		if supportsCreate {
			tracef(cmd.Context(), "create: selector supports create flow")
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
			tracef(cmd.Context(), "create: selector fallback select-only flow")
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
