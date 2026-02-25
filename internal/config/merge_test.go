package config

import (
	"path/filepath"
	"reflect"
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

func TestNormalizeOverride_ReadsBooleanFields(t *testing.T) {
	t.Parallel()

	fetch := false
	prune := false
	openPrune := false
	override, err := normalizeOverride(rawConfig{
		New: &rawNew{
			Fetch: &fetch,
			Prune: &prune,
		},
		Open: &rawOpen{
			Prune: &openPrune,
		},
	})
	if err != nil {
		t.Fatalf("normalizeOverride() returned error: %v", err)
	}

	if override.NewFetch == nil || *override.NewFetch {
		t.Fatalf("unexpected NewFetch override: %+v", override.NewFetch)
	}
	if override.NewPrune == nil || *override.NewPrune {
		t.Fatalf("unexpected NewPrune override: %+v", override.NewPrune)
	}
	if override.OpenPrune == nil || *override.OpenPrune {
		t.Fatalf("unexpected OpenPrune override: %+v", override.OpenPrune)
	}
}

func TestMergeConfig_AppliesBooleanOverrides(t *testing.T) {
	t.Parallel()

	fetch := false
	prune := false
	openPrune := false

	got := mergeConfig(DefaultConfig(), configOverride{
		NewFetch:  &fetch,
		NewPrune:  &prune,
		OpenPrune: &openPrune,
	})
	want := DefaultConfig()
	want.New.Fetch = false
	want.New.Prune = false
	want.Open.Prune = false

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected merged config:\nwant=%+v\ngot=%+v", want, got)
	}
}
