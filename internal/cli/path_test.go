package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

type fakeSelector struct {
	index int
	err   error
}

func (f fakeSelector) Select(_ context.Context, _ string, _ []string) (int, error) {
	return f.index, f.err
}

func TestPathCommand_PrintsSelectedPath(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	repoPath := strings.ReplaceAll(t.TempDir(), "\\", "/")
	worktreePath := strings.ReplaceAll(t.TempDir(), "\\", "/")
	gitClient := &fakeGitClient{
		output: "worktree " + repoPath + "\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + worktreePath + "\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}
	cmd := NewRootCmd(Dependencies{
		Stdout:   &stdout,
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: fakeSelector{index: 1},
	})
	cmd.SetArgs([]string{"path"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != worktreePath {
		t.Fatalf("unexpected path output: %q", got)
	}
	if gitClient.worktreePruneCall != 0 {
		t.Fatalf("path must not prune metadata, got %d prune calls", gitClient.worktreePruneCall)
	}
}

func TestPathCommand_BranchOutputJSON(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	repoPath := strings.ReplaceAll(t.TempDir(), "\\", "/")
	worktreePath := strings.ReplaceAll(t.TempDir(), "\\", "/")
	gitClient := &fakeGitClient{
		output: "worktree " + repoPath + "\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + worktreePath + "\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}
	cmd := NewRootCmd(Dependencies{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Git:    gitClient,
	})
	cmd.SetArgs([]string{"path", "--branch", "feature/x", "--output", "json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("failed to unmarshal json output: %v", err)
	}
	if len(payload) != 3 {
		t.Fatalf("unexpected json keys: %+v", payload)
	}
	if payload["command"] != "path" || payload["path"] != worktreePath || payload["branch"] != "feature/x" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestPathCommand_FiltersUnavailableWorktrees(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	missingPath := filepath.Join(t.TempDir(), "missing")
	gitClient := &fakeGitClient{
		output: "worktree C:/worktrees/stale\nHEAD abc\nbranch refs/heads/old\nprunable gitdir file points to non-existent location\n\n",
	}
	gitClient.output += "worktree " + strings.ReplaceAll(missingPath, "\\", "/") + "\nHEAD def\nbranch refs/heads/missing\n\n"
	cmd := NewRootCmd(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		Git:    gitClient,
	})
	cmd.SetArgs([]string{"path"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no valid worktrees found") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "1 stale and 1 missing") {
		t.Fatalf("expected unavailable counts in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "wtm remove <branch>") {
		t.Fatalf("expected cleanup guidance in error, got: %v", err)
	}
	if !strings.Contains(stderr.String(), "skipped 1 stale worktree") {
		t.Fatalf("expected stale warning, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "skipped 1 missing worktree") {
		t.Fatalf("expected missing warning, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "wtm remove <branch>") {
		t.Fatalf("expected cleanup guidance, got: %s", stderr.String())
	}
	if gitClient.worktreePruneCall != 0 {
		t.Fatalf("path must not prune metadata, got %d prune calls", gitClient.worktreePruneCall)
	}
}
