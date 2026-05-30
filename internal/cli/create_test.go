package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takubii/worktree-manager/internal/config"
)

func TestCreateCommand_CreatesWorktreeFromLocalBranch(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	expectedPath := filepath.Clean(filepath.Join(repoRoot, "..", "worktrees", "feature", "local"))
	gitClient := &fakeGitClient{
		repoRoot:      repoRoot,
		localBranches: []string{"main", "feature/local"},
		output:        "worktree " + strings.ReplaceAll(repoRoot, "\\", "/") + "\nHEAD abc\nbranch refs/heads/main\n\n",
	}

	var stdout bytes.Buffer
	cmd := NewRootCmd(Dependencies{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Git:    gitClient,
		Config: config.NewStaticProvider(config.DefaultConfig()),
	})
	cmd.SetArgs([]string{"create", "feature/local", "--output", "path"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != expectedPath {
		t.Fatalf("unexpected path output: want=%q got=%q", expectedPath, got)
	}
	if len(gitClient.worktreeAddCalls) != 1 {
		t.Fatalf("expected one WorktreeAdd call, got %d", len(gitClient.worktreeAddCalls))
	}
	if got := gitClient.worktreeAddCalls[0].Path; got != expectedPath {
		t.Fatalf("unexpected worktree path: want=%q got=%q", expectedPath, got)
	}
}

func TestCreateCommand_UsesCreateConfigDefaults(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.Create.Fetch = false
	cfg.Create.Prune = false

	gitClient := &fakeGitClient{
		repoRoot:      t.TempDir(),
		localBranches: []string{"main", "feature/local"},
		output:        "worktree C:/repo\nHEAD abc\nbranch refs/heads/main\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Git:    gitClient,
		Config: config.NewStaticProvider(cfg),
	})
	cmd.SetArgs([]string{"create", "feature/local"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if gitClient.fetchRemote != "" {
		t.Fatalf("FetchPrune should not be called, got remote=%q", gitClient.fetchRemote)
	}
	if gitClient.worktreePruneCall != 0 {
		t.Fatalf("WorktreePrune should not be called, got %d", gitClient.worktreePruneCall)
	}
}
