package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type execCommandFunc func(ctx context.Context, name string, args ...string) *exec.Cmd

type execClient struct {
	execCommand execCommandFunc
}

func newExecClient(execCommand execCommandFunc) *execClient {
	if execCommand == nil {
		execCommand = exec.CommandContext
	}
	return &execClient{execCommand: execCommand}
}

func (c *execClient) WorktreeListPorcelain(ctx context.Context) (string, error) {
	stdout, stderr, err := c.runGit(ctx, "worktree", "list", "--porcelain")
	if err != nil {
		return "", buildGitCommandError(
			err,
			stderr,
			"worktree list --porcelain",
			"Run this command inside a Git repository, then retry",
		)
	}

	return stdout, nil
}

func (c *execClient) WorktreePrune(ctx context.Context) error {
	args := []string{"worktree", "prune", "--expire", "now"}
	_, stderr, err := c.runGit(ctx, args...)
	if err != nil {
		return buildGitCommandError(
			err,
			stderr,
			strings.Join(args, " "),
			"Run this command inside a Git repository, then retry",
		)
	}

	return nil
}

func (c *execClient) WorktreeAdd(ctx context.Context, params WorktreeAddParams) error {
	path := strings.TrimSpace(params.Path)
	if path == "" {
		return fmt.Errorf("worktree path is empty. Provide a valid destination path and retry")
	}

	branch := normalizeBranchName(params.Branch)
	if branch == "" {
		return fmt.Errorf("branch name is empty. Specify a branch name and retry")
	}

	args := []string{"worktree", "add"}
	startPoint := strings.TrimSpace(params.StartPoint)
	if startPoint != "" {
		args = append(args, "-b", branch, path, startPoint)
	} else {
		args = append(args, path, branch)
	}

	_, stderr, err := c.runGit(ctx, args...)
	if err != nil {
		return buildGitCommandError(
			err,
			stderr,
			strings.Join(args, " "),
			"Check branch/path conflicts (for example, an existing worktree for that branch) and retry",
		)
	}

	return nil
}

func (c *execClient) WorktreeRemove(ctx context.Context, path string, force bool) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("worktree path is empty. Provide a valid worktree path and retry")
	}

	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)

	_, stderr, err := c.runGit(ctx, args...)
	if err != nil {
		return buildGitCommandError(
			err,
			stderr,
			strings.Join(args, " "),
			"Ensure the worktree path exists and is not in use, then retry (or use `--force` when appropriate)",
		)
	}

	return nil
}

func (c *execClient) runGit(ctx context.Context, args ...string) (string, string, error) {
	cmd := c.execCommand(ctx, "git", args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String(), stderr.String(), err
	}

	return stdout.String(), stderr.String(), nil
}
