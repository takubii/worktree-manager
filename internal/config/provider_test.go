package config

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

type fakeRepoRootFinder struct {
	root string
	err  error
}

func (f fakeRepoRootFinder) RepoRoot(context.Context) (string, error) {
	return f.root, f.err
}

func TestFileProviderLoad_MergesGlobalAndRepoConfig(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	repoRoot := filepath.Join(tmp, "repo")
	globalPath := filepath.Join(tmp, "worktree-manager", "config.json")
	repoPath := filepath.Join(repoRoot, ".wtmconfig.json")

	files := map[string]string{
		globalPath: `{"create":{"fetch":false},"remove":{"deleteBranch":"none"}}`,
		repoPath:   `{"baseBranch":"develop","create":{"prune":false}}`,
	}
	provider := NewFileProvider(FileProviderOptions{
		Git:           fakeRepoRootFinder{root: repoRoot},
		UserConfigDir: func() (string, error) { return tmp, nil },
		ReadFile: func(path string) ([]byte, error) {
			body, ok := files[path]
			if !ok {
				return nil, errors.New("not found")
			}
			return []byte(body), nil
		},
	})

	got := provider.Load(context.Background())
	if got.BaseBranch != "develop" {
		t.Fatalf("unexpected baseBranch: %q", got.BaseBranch)
	}
	if got.Create.Fetch {
		t.Fatalf("unexpected create.fetch: %v", got.Create.Fetch)
	}
	if got.Create.Prune {
		t.Fatalf("unexpected create.prune: %v", got.Create.Prune)
	}
	if got.Remove.DeleteBranch != DeleteBranchNone {
		t.Fatalf("unexpected remove.deleteBranch: %q", got.Remove.DeleteBranch)
	}
}

func TestConfigPathsUseWorktreeManagerNames(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	globalPath, err := GlobalConfigPath(func() (string, error) { return tmp, nil })
	if err != nil {
		t.Fatalf("GlobalConfigPath() returned error: %v", err)
	}
	if globalPath != filepath.Join(tmp, "worktree-manager", "config.json") {
		t.Fatalf("unexpected global path: %q", globalPath)
	}

	repoPath := RepoConfigPath(filepath.Join(tmp, "repo"))
	if repoPath != filepath.Join(tmp, "repo", ".wtmconfig.json") {
		t.Fatalf("unexpected repo path: %q", repoPath)
	}
}
