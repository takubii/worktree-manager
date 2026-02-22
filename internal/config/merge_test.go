package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderWorktreeDir_ReplacesPlaceholders(t *testing.T) {
	t.Parallel()

	repoRoot := filepath.Join(t.TempDir(), "repo")
	got, err := RenderWorktreeDir("{repoParent}/worktrees/{branch}", repoRoot, "feature/x")
	if err != nil {
		t.Fatalf("RenderWorktreeDir() returned error: %v", err)
	}

	expected := filepath.Clean(filepath.Join(filepath.Dir(repoRoot), "worktrees", "feature", "x"))
	if filepath.Clean(got) != expected {
		t.Fatalf("unexpected rendered path: want=%q got=%q", expected, filepath.Clean(got))
	}
}

func TestRenderWorktreeDir_ReturnsErrorForUnknownPlaceholder(t *testing.T) {
	t.Parallel()

	_, err := RenderWorktreeDir("{repoRoot}/worktrees/{unknown}", "/repo/project", "feature/x")
	if err == nil {
		t.Fatal("expected RenderWorktreeDir() to return error")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRenderWorktreeDir_ReturnsErrorForPlaceholderWithHyphen(t *testing.T) {
	t.Parallel()

	_, err := RenderWorktreeDir("{repoRoot}/worktrees/{repo-parent}", "/repo/project", "feature/x")
	if err == nil {
		t.Fatal("expected RenderWorktreeDir() to return error")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}
