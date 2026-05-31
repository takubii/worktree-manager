package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/takubii/worktree-manager/internal/config"
)

func TestRemoveCommand_RemovesSelectedWorktree(t *testing.T) {
	t.Parallel()

	gitClient := &fakeGitClient{
		output: "worktree C:/repo\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree C:/worktrees/feature-x\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}
	cmd := NewRootCmd(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Git:    gitClient,
		Config: config.NewStaticProvider(config.DefaultConfig()),
	})
	cmd.SetArgs([]string{"remove", "feature/x", "--delete-branch", "none"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if len(gitClient.worktreeRemove) != 1 {
		t.Fatalf("expected one WorktreeRemove call, got %d", len(gitClient.worktreeRemove))
	}
	if got := gitClient.worktreeRemove[0].path; got != "C:/worktrees/feature-x" {
		t.Fatalf("unexpected worktree remove path: %q", got)
	}
}

func TestRemoveCommand_DryRunUsesNewCommandName(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	gitClient := &fakeGitClient{
		output: "worktree C:/worktrees/feature-x\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}
	cmd := NewRootCmd(Dependencies{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Git:    gitClient,
		Config: config.NewStaticProvider(config.DefaultConfig()),
	})
	cmd.SetArgs([]string{"remove", "feature/x", "--dry-run"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), `git worktree remove "C:/worktrees/feature-x"`) {
		t.Fatalf("expected planned worktree remove command, got: %s", stdout.String())
	}
	if len(gitClient.worktreeRemove) != 0 {
		t.Fatalf("WorktreeRemove should not be called in dry-run, got %+v", gitClient.worktreeRemove)
	}
}

func TestRemoveCommand_DryRunExplainsStalePrunePlan(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	gitClient := &fakeGitClient{
		output: "worktree C:/worktrees/stale\nHEAD def\nbranch refs/heads/feature/stale\nprunable gitdir file points to non-existent location\n\n",
	}
	cmd := NewRootCmd(Dependencies{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Git:    gitClient,
		Config: config.NewStaticProvider(config.DefaultConfig()),
	})
	cmd.SetArgs([]string{"remove", "feature/stale", "--dry-run"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "git worktree prune --expire now") {
		t.Fatalf("expected planned prune command, got: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "selected worktree is stale") {
		t.Fatalf("expected stale guidance note, got: %s", stdout.String())
	}
	if gitClient.worktreePruneCall != 0 {
		t.Fatalf("WorktreePrune should not be called in dry-run, got %d", gitClient.worktreePruneCall)
	}
	if len(gitClient.worktreeRemove) != 0 {
		t.Fatalf("WorktreeRemove should not be called in dry-run, got %+v", gitClient.worktreeRemove)
	}
}
