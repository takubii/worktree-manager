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
	defaultRemoteName = config.DefaultRemote
	defaultBaseBranch = config.DefaultBaseBranch
)

func newNewCmd(deps Dependencies) *cobra.Command {
	var baseBranch string
	var openerName string

	cmd := &cobra.Command{
		Use:   "new [branch]",
		Short: "Create and open a new worktree",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := deps.Config.Load(cmd.Context())

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

			resolvedOpener := strings.TrimSpace(openerName)
			if !cmd.Flags().Changed("open") {
				resolvedOpener = cfg.Open.Default
			}

			windowMode, err := opener.ParseWindowMode(cfg.Open.Window)
			if err != nil {
				return fmt.Errorf("invalid config open.window value: %w", err)
			}

			targetBranch := ""
			if len(args) == 1 {
				targetBranch = args[0]
			}

			if err := deps.Git.FetchPrune(cmd.Context(), remoteName); err != nil {
				return err
			}

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

			if err := deps.Opener.Open(cmd.Context(), resolvedOpener, worktreePath, windowMode); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&baseBranch, "base", defaultBaseBranch, "base branch used when creating a new branch")
	cmd.Flags().StringVar(&openerName, "open", "system", "opener to use: system|vscode|cursor|vim")

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
	branchArg = normalizeBranchForRemote(branchArg, remoteName)

	localSet := asBranchSet(localBranches, remoteName)
	remoteSet := asRemoteBranchSet(remoteBranches, remoteName)

	if branchArg == "" {
		candidates := branchCandidates(localBranches, remoteBranches, remoteName)
		if len(candidates) == 0 {
			return "", "", fmt.Errorf("no branches available. Create or fetch branches, then run `wto new` again")
		}

		creator, supportsCreate := deps.Selector.(selector.SelectOrCreator)
		if supportsCreate {
			result, err := creator.SelectOrCreate(cmd.Context(), "Select or enter a branch for the new worktree:", candidates)
			if err != nil {
				return "", "", err
			}

			branchArg = normalizeBranchForRemote(result.Value, remoteName)
			if branchArg == "" {
				return "", "", fmt.Errorf("branch name is empty. Enter a branch name and retry")
			}

			if result.IsNew {
				if err := deps.Git.CheckBranchName(cmd.Context(), branchArg); err != nil {
					return "", "", err
				}
			}
		} else {
			selectedIndex, err := deps.Selector.Select(cmd.Context(), "Select a branch for the new worktree:", candidates)
			if err != nil {
				return "", "", err
			}
			if selectedIndex < 0 || selectedIndex >= len(candidates) {
				return "", "", fmt.Errorf("invalid branch selection index: %d", selectedIndex)
			}
			branchArg = candidates[selectedIndex]
		}
	}

	if _, ok := localSet[branchArg]; ok {
		return branchArg, "", nil
	}
	if _, ok := remoteSet[branchArg]; ok {
		return branchArg, remoteName + "/" + branchArg, nil
	}

	baseBranch = normalizeBranchForRemote(baseBranch, remoteName)
	if _, ok := localSet[baseBranch]; ok {
		return branchArg, baseBranch, nil
	}
	if _, ok := remoteSet[baseBranch]; ok {
		return branchArg, remoteName + "/" + baseBranch, nil
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

func normalizeBranch(branch string) string {
	return normalizeBranchForRemote(branch, defaultRemoteName)
}

func normalizeBranchForRemote(branch string, remote string) string {
	branch = strings.TrimSpace(branch)
	branch = strings.TrimPrefix(branch, "refs/heads/")
	remote = strings.TrimSpace(remote)
	if remote != "" {
		return strings.TrimPrefix(branch, remote+"/")
	}
	return branch
}

func asBranchSet(branches []string, remote string) map[string]struct{} {
	set := make(map[string]struct{}, len(branches))
	for _, branch := range branches {
		normalized := normalizeBranchForRemote(branch, remote)
		if normalized == "" {
			continue
		}
		set[normalized] = struct{}{}
	}
	return set
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

func branchCandidates(localBranches []string, remoteBranches []string, remote string) []string {
	candidates := make([]string, 0, len(localBranches)+len(remoteBranches))
	seen := make(map[string]struct{}, len(localBranches)+len(remoteBranches))

	for _, branch := range localBranches {
		normalized := normalizeBranchForRemote(branch, remote)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		candidates = append(candidates, normalized)
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
		candidates = append(candidates, short)
	}

	return candidates
}
