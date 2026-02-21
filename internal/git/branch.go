package git

import (
	"context"
	"fmt"
	"strings"
)

func (c *execClient) RepoRoot(ctx context.Context) (string, error) {
	stdout, stderr, err := c.runGit(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", buildGitCommandError(
			err,
			stderr,
			"rev-parse --show-toplevel",
			"Run this command inside a Git repository, then retry",
		)
	}

	repoRoot := strings.TrimSpace(stdout)
	if repoRoot == "" {
		return "", fmt.Errorf("failed to detect repository root. Run this command inside a Git repository, then retry")
	}

	return repoRoot, nil
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
	if strings.HasPrefix(branch, "refs/heads/") {
		return strings.TrimPrefix(branch, "refs/heads/")
	}
	return strings.TrimPrefix(branch, "origin/")
}
