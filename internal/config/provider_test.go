package config

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type fakeRepoRootFinder struct {
	root string
	err  error
}

func (f *fakeRepoRootFinder) RepoRoot(context.Context) (string, error) {
	return f.root, f.err
}

func TestFileProviderLoad_ReturnsDefaultsWhenNoConfigFilesExist(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	provider := NewFileProvider(FileProviderOptions{
		Stderr: &bytes.Buffer{},
		Git: &fakeRepoRootFinder{
			root: filepath.Join(tmp, "repo"),
		},
		UserConfigDir: func() (string, error) {
			return tmp, nil
		},
	})

	got := provider.Load(context.Background())
	if !reflect.DeepEqual(got, DefaultConfig()) {
		t.Fatalf("unexpected config:\nwant=%+v\ngot=%+v", DefaultConfig(), got)
	}
}

func TestFileProviderLoad_AppliesGlobalConfig(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	globalPath := filepath.Join(tmp, AppConfigDirName, GlobalConfigFileName)
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	globalBody := `{
  "remote": "upstream",
  "new": {
    "fetch": false,
    "prune": false
  },
  "open": {
    "default": "cursor",
    "prune": false,
    "terminalProvider": "warp"
  }
}`
	if err := os.WriteFile(globalPath, []byte(globalBody), 0o644); err != nil {
		t.Fatalf("failed to write global config: %v", err)
	}

	provider := NewFileProvider(FileProviderOptions{
		Stderr: &bytes.Buffer{},
		Git: &fakeRepoRootFinder{
			err: errors.New("not in repo"),
		},
		UserConfigDir: func() (string, error) {
			return tmp, nil
		},
	})

	got := provider.Load(context.Background())
	if got.Remote != "upstream" {
		t.Fatalf("unexpected remote: %q", got.Remote)
	}
	if got.Open.Default != "cursor" {
		t.Fatalf("unexpected open.default: %q", got.Open.Default)
	}
	if got.New.Fetch {
		t.Fatalf("unexpected new.fetch: %v", got.New.Fetch)
	}
	if got.New.Prune {
		t.Fatalf("unexpected new.prune: %v", got.New.Prune)
	}
	if got.Open.Prune {
		t.Fatalf("unexpected open.prune: %v", got.Open.Prune)
	}
	if got.Open.TerminalProvider != "warp" {
		t.Fatalf("unexpected open.terminalProvider: %q", got.Open.TerminalProvider)
	}
	if got.Open.Window != DefaultConfig().Open.Window {
		t.Fatalf("unexpected open.window: %q", got.Open.Window)
	}
}

func TestFileProviderLoad_RepoConfigOverridesGlobalConfig(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	repoRoot := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("failed to create repo root: %v", err)
	}

	globalPath := filepath.Join(tmp, AppConfigDirName, GlobalConfigFileName)
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	if err := os.WriteFile(globalPath, []byte(`{"open":{"default":"cursor"}}`), 0o644); err != nil {
		t.Fatalf("failed to write global config: %v", err)
	}

	repoPath := filepath.Join(repoRoot, RepoConfigFileName)
	if err := os.WriteFile(repoPath, []byte(`{"open":{"default":"vscode"}}`), 0o644); err != nil {
		t.Fatalf("failed to write repo config: %v", err)
	}

	provider := NewFileProvider(FileProviderOptions{
		Stderr: &bytes.Buffer{},
		Git: &fakeRepoRootFinder{
			root: repoRoot,
		},
		UserConfigDir: func() (string, error) {
			return tmp, nil
		},
	})

	got := provider.Load(context.Background())
	if got.Open.Default != "vscode" {
		t.Fatalf("unexpected open.default: %q", got.Open.Default)
	}
}

func TestFileProviderLoad_IgnoresInvalidGlobalConfigAndWarns(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	repoRoot := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("failed to create repo root: %v", err)
	}

	globalPath := filepath.Join(tmp, AppConfigDirName, GlobalConfigFileName)
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	if err := os.WriteFile(globalPath, []byte(`{"unknown":"value"}`), 0o644); err != nil {
		t.Fatalf("failed to write global config: %v", err)
	}

	repoPath := filepath.Join(repoRoot, RepoConfigFileName)
	if err := os.WriteFile(repoPath, []byte(`{"remote":"upstream"}`), 0o644); err != nil {
		t.Fatalf("failed to write repo config: %v", err)
	}

	var stderr bytes.Buffer
	provider := NewFileProvider(FileProviderOptions{
		Stderr: &stderr,
		Git: &fakeRepoRootFinder{
			root: repoRoot,
		},
		UserConfigDir: func() (string, error) {
			return tmp, nil
		},
	})

	got := provider.Load(context.Background())
	if got.Remote != "upstream" {
		t.Fatalf("unexpected remote: %q", got.Remote)
	}
	if !strings.Contains(stderr.String(), "ignoring invalid global config") {
		t.Fatalf("expected warning in stderr, got: %s", stderr.String())
	}
}

func TestFileProviderLoad_IgnoresInvalidRepoConfigAndWarns(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	repoRoot := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("failed to create repo root: %v", err)
	}

	globalPath := filepath.Join(tmp, AppConfigDirName, GlobalConfigFileName)
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	if err := os.WriteFile(globalPath, []byte(`{"remote":"upstream"}`), 0o644); err != nil {
		t.Fatalf("failed to write global config: %v", err)
	}

	repoPath := filepath.Join(repoRoot, RepoConfigFileName)
	if err := os.WriteFile(repoPath, []byte(`{"open":{"window":"invalid"}}`), 0o644); err != nil {
		t.Fatalf("failed to write repo config: %v", err)
	}

	var stderr bytes.Buffer
	provider := NewFileProvider(FileProviderOptions{
		Stderr: &stderr,
		Git: &fakeRepoRootFinder{
			root: repoRoot,
		},
		UserConfigDir: func() (string, error) {
			return tmp, nil
		},
	})

	got := provider.Load(context.Background())
	if got.Remote != "upstream" {
		t.Fatalf("unexpected remote: %q", got.Remote)
	}
	if !strings.Contains(stderr.String(), "ignoring invalid repo config") {
		t.Fatalf("expected warning in stderr, got: %s", stderr.String())
	}
}

func TestFileProviderInitGlobal_CreateAndForceOverwrite(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	provider := NewFileProvider(FileProviderOptions{
		Stderr: &bytes.Buffer{},
		UserConfigDir: func() (string, error) {
			return tmp, nil
		},
	})

	path, err := provider.InitGlobal(false)
	if err != nil {
		t.Fatalf("InitGlobal(false) returned error: %v", err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read initialized config: %v", err)
	}
	if !strings.Contains(string(body), `"remote": "origin"`) {
		t.Fatalf("unexpected config body: %s", string(body))
	}

	if _, err := provider.InitGlobal(false); err == nil {
		t.Fatal("expected InitGlobal(false) to return error when file exists")
	}

	if err := os.WriteFile(path, []byte(`{"remote":"broken"}`), 0o644); err != nil {
		t.Fatalf("failed to modify config file: %v", err)
	}

	if _, err := provider.InitGlobal(true); err != nil {
		t.Fatalf("InitGlobal(true) returned error: %v", err)
	}

	body, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read overwritten config: %v", err)
	}
	if !strings.Contains(string(body), `"baseBranch": "main"`) {
		t.Fatalf("unexpected overwritten config body: %s", string(body))
	}
}
