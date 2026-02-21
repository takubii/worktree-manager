package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takubii/git-worktree-opener/internal/git"
	"github.com/takubii/git-worktree-opener/internal/opener"
)

const (
	defaultRemoteName = "origin"
	defaultBaseBranch = "main"
)

func newNewCmd(deps Dependencies) *cobra.Command {
	var baseBranch string
	var openerName string

	cmd := &cobra.Command{
		Use:   "new [branch]",
		Short: "Create and open a new worktree",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetBranch := ""
			if len(args) == 1 {
				targetBranch = args[0]
			}

			baseBranch = strings.TrimSpace(baseBranch)
			if baseBranch == "" {
				return fmt.Errorf("base branch is empty. Set --base to a valid branch and retry")
			}

			if err := deps.Git.FetchPrune(cmd.Context(), defaultRemoteName); err != nil {
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
			remoteBranches, err := deps.Git.RemoteBranches(cmd.Context(), defaultRemoteName)
			if err != nil {
				return err
			}

			resolvedBranch, startPoint, err := resolveTargetBranch(
				cmd,
				deps,
				targetBranch,
				baseBranch,
				localBranches,
				remoteBranches,
			)
			if err != nil {
				return err
			}

			worktreePath, err := defaultWorktreePath(repoRoot, resolvedBranch)
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

			if err := deps.Opener.Open(cmd.Context(), openerName, worktreePath, opener.WindowNew); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&baseBranch, "base", defaultBaseBranch, "base branch used when creating a new branch")
	cmd.Flags().StringVar(&openerName, "open", "system", "opener to use: system|vscode|cursor|vim")

	return cmd
}

func resolveTargetBranch(cmd *cobra.Command, deps Dependencies, branchArg string, baseBranch string, localBranches []string, remoteBranches []string) (string, string, error) {
	branchArg = normalizeBranch(branchArg)

	localSet := asBranchSet(localBranches)
	remoteSet := asRemoteBranchSet(remoteBranches, defaultRemoteName)

	if branchArg == "" {
		candidates := branchCandidates(localBranches, remoteBranches, defaultRemoteName)
		if len(candidates) == 0 {
			return "", "", fmt.Errorf("no branches available. Create or fetch branches, then run `wto new` again")
		}

		selectedIndex, err := deps.Selector.Select(cmd.Context(), "Select a branch for the new worktree:", candidates)
		if err != nil {
			return "", "", err
		}
		if selectedIndex < 0 || selectedIndex >= len(candidates) {
			return "", "", fmt.Errorf("invalid branch selection index: %d", selectedIndex)
		}
		branchArg = candidates[selectedIndex]
	}

	if _, ok := localSet[branchArg]; ok {
		return branchArg, "", nil
	}
	if _, ok := remoteSet[branchArg]; ok {
		return branchArg, defaultRemoteName + "/" + branchArg, nil
	}

	baseBranch = normalizeBranch(baseBranch)
	if _, ok := localSet[baseBranch]; ok {
		return branchArg, baseBranch, nil
	}
	if _, ok := remoteSet[baseBranch]; ok {
		return branchArg, defaultRemoteName + "/" + baseBranch, nil
	}

	return branchArg, baseBranch, nil
}

func defaultWorktreePath(repoRoot string, branch string) (string, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return "", fmt.Errorf("repository root is empty. Run this command inside a Git repository, then retry")
	}

	branch = normalizeBranch(branch)
	if branch == "" {
		return "", fmt.Errorf("branch name is empty. Specify a branch and retry")
	}

	return filepath.Join(filepath.Dir(repoRoot), "worktrees", filepath.FromSlash(branch)), nil
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
	branch = strings.TrimSpace(branch)
	branch = strings.TrimPrefix(branch, "refs/heads/")
	return strings.TrimPrefix(branch, defaultRemoteName+"/")
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
		normalized := normalizeBranch(branch)
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
