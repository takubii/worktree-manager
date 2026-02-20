package git

import (
	"bytes"
	"context"
	"errors"
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
	cmd := c.execCommand(ctx, "git", "worktree", "list", "--porcelain")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", buildWorktreeListError(err, stderr.String())
	}

	return stdout.String(), nil
}

func buildWorktreeListError(runErr error, stderrOutput string) error {
	var execErr *exec.Error
	if errors.As(runErr, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
		return fmt.Errorf("`git` command was not found. Install Git and ensure it is available in PATH, then retry")
	}

	stderrOutput = strings.TrimSpace(stderrOutput)
	if stderrOutput == "" {
		return fmt.Errorf("failed to run `git worktree list --porcelain`: %w. Run this command inside a Git repository", runErr)
	}

	return fmt.Errorf("failed to run `git worktree list --porcelain`: %s. Run this command inside a Git repository", stderrOutput)
}
