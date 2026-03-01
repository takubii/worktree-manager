package git

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

func (c *execClient) RepoRoot(ctx context.Context) (string, error) {
	stdout, stderr, err := c.runGit(ctx, "rev-parse", "--show-toplevel", "--git-common-dir")
	if err != nil {
		return "", buildGitCommandError(
			err,
			stderr,
			"rev-parse --show-toplevel --git-common-dir",
			"Run this command inside a Git repository, then retry",
		)
	}

	lines := parseBranchLines(stdout)
	if len(lines) == 0 {
		return "", fmt.Errorf("failed to detect repository root. Run this command inside a Git repository, then retry")
	}

	repoRoot := strings.TrimSpace(lines[0])
	if repoRoot == "" {
		return "", fmt.Errorf("failed to detect repository root. Run this command inside a Git repository, then retry")
	}

	if len(lines) >= 2 {
		commonDir := strings.TrimSpace(lines[1])
		if canonical := canonicalRepoRootFromCommonDir(commonDir); canonical != "" {
			return canonical, nil
		}
	}

	return repoRoot, nil
}

func canonicalRepoRootFromCommonDir(commonDir string) string {
	commonDir = strings.TrimSpace(commonDir)
	if commonDir == "" {
		return ""
	}

	if !filepath.IsAbs(commonDir) {
		abs, err := filepath.Abs(commonDir)
		if err != nil {
			return ""
		}
		commonDir = abs
	}

	commonDir = filepath.ToSlash(filepath.Clean(commonDir))
	if !strings.HasSuffix(commonDir, "/.git") {
		return ""
	}

	canonical := strings.TrimSuffix(commonDir, "/.git")
	if strings.TrimSpace(canonical) == "" {
		return ""
	}

	return canonical
}

func (c *execClient) FetchPrune(ctx context.Context, remote string) error {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return fmt.Errorf("remote name is empty. Specify a remote (for example, `origin`) and retry")
	}

	_, stderr, err := c.runGit(ctx, "fetch", remote, "--prune")
	if err != nil {
		return buildGitCommandError(
			err,
			stderr,
			fmt.Sprintf("fetch %s --prune", remote),
			fmt.Sprintf("Run this command inside a Git repository and ensure remote `%s` exists (`git remote -v`), then retry", remote),
		)
	}

	return nil
}

func (c *execClient) LocalBranches(ctx context.Context) ([]string, error) {
	stdout, stderr, err := c.runGit(ctx, "branch", "--format=%(refname:short)")
	if err != nil {
		return nil, buildGitCommandError(
			err,
			stderr,
			"branch --format=%(refname:short)",
			"Run this command inside a Git repository, then retry",
		)
	}

	return parseBranchLines(stdout), nil
}

func (c *execClient) RemoteBranches(ctx context.Context, remote string) ([]string, error) {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return nil, fmt.Errorf("remote name is empty. Specify a remote (for example, `origin`) and retry")
	}

	stdout, stderr, err := c.runGit(ctx, "branch", "-r", "--format=%(refname:short)")
	if err != nil {
		return nil, buildGitCommandError(
			err,
			stderr,
			"branch -r --format=%(refname:short)",
			"Run this command inside a Git repository, then retry",
		)
	}

	prefix := remote + "/"
	branches := make([]string, 0)
	for _, branch := range parseBranchLines(stdout) {
		if strings.Contains(branch, "->") {
			continue
		}
		if !strings.HasPrefix(branch, prefix) {
			continue
		}
		if branch == prefix+"HEAD" {
			continue
		}
		branches = append(branches, branch)
	}

	return branches, nil
}

func (c *execClient) CheckBranchName(ctx context.Context, branch string) error {
	branch = normalizeBranchName(branch)
	if branch == "" {
		return fmt.Errorf("branch name is empty. Enter a branch name and retry")
	}

	_, stderr, err := c.runGit(ctx, "check-ref-format", "--branch", branch)
	if err != nil {
		return buildGitCommandError(
			err,
			stderr,
			fmt.Sprintf("check-ref-format --branch %s", branch),
			"Use a valid branch name (for example, `feature/my-task`) and retry",
		)
	}

	return nil
}

func (c *execClient) DeleteLocalBranch(ctx context.Context, branch string, force bool) error {
	branch = normalizeBranchName(branch)
	if branch == "" {
		return fmt.Errorf("branch name is empty. Specify a branch to delete and retry")
	}

	args := []string{"branch"}
	if force {
		args = append(args, "-D")
	} else {
		args = append(args, "-d")
	}
	args = append(args, branch)

	_, stderr, err := c.runGit(ctx, args...)
	if err != nil {
		return buildGitCommandError(
			err,
			stderr,
			strings.Join(args, " "),
			"Ensure the branch is not checked out in any worktree and, for safe deletion, is already merged; otherwise retry with `--delete-branch force`",
		)
	}

	return nil
}

func parseBranchLines(raw string) []string {
	lines := strings.Split(raw, "\n")
	branches := make([]string, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))

	for _, line := range lines {
		branch := strings.TrimSpace(line)
		if branch == "" {
			continue
		}
		if _, ok := seen[branch]; ok {
			continue
		}
		seen[branch] = struct{}{}
		branches = append(branches, branch)
	}

	return branches
}

func normalizeBranchName(branch string) string {
	branch = strings.TrimSpace(branch)
	return strings.TrimPrefix(branch, "refs/heads/")
}
