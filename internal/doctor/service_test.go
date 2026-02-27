package doctor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRepoRootFinder struct {
	root string
	err  error
}

func (f fakeRepoRootFinder) RepoRoot(context.Context) (string, error) {
	return f.root, f.err
}

func TestServiceRun_GitMissingIsCritical(t *testing.T) {
	t.Parallel()

	svc := NewService(Options{
		LookPath: func(file string) (string, error) {
			return "", errors.New(file + " not found")
		},
		Git: fakeRepoRootFinder{
			root: "C:/repo",
		},
	})

	report := svc.Run(context.Background())
	if !report.HasCritical {
		t.Fatal("expected HasCritical=true")
	}

	result, ok := findResult(report.Results, "git")
	if !ok {
		t.Fatal("expected git result")
	}
	if result.Level != LevelCritical {
		t.Fatalf("expected CRIT, got %s", result.Level)
	}
}

func TestServiceRun_InvalidGlobalConfigWarns(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	repoRoot := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(repoRoot) returned error: %v", err)
	}

	globalPath := filepath.Join(tmp, "git-worktree-opener", "config.json")
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(globalPath) returned error: %v", err)
	}
	if err := os.WriteFile(globalPath, []byte(`{"unknown":"value"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile(globalPath) returned error: %v", err)
	}

	svc := NewService(Options{
		LookPath: func(file string) (string, error) {
			if file == "git" || file == "curl" {
				return "/usr/bin/" + file, nil
			}
			return "", errors.New("not found")
		},
		Git: fakeRepoRootFinder{
			root: repoRoot,
		},
		UserConfigDir: func() (string, error) {
			return tmp, nil
		},
	})

	report := svc.Run(context.Background())
	result, ok := findResult(report.Results, "config/global")
	if !ok {
		t.Fatal("expected config/global result")
	}
	if result.Level != LevelWarn {
		t.Fatalf("expected WARN, got %s", result.Level)
	}
	if !strings.Contains(result.Message, "invalid") {
		t.Fatalf("unexpected message: %s", result.Message)
	}
}

func TestServiceRun_RepositoryWarningOutsideRepo(t *testing.T) {
	t.Parallel()

	svc := NewService(Options{
		LookPath: func(file string) (string, error) {
			return "ok", nil
		},
		Git: fakeRepoRootFinder{
			err: errors.New("not in repo"),
		},
	})

	report := svc.Run(context.Background())
	result, ok := findResult(report.Results, "repository")
	if !ok {
		t.Fatal("expected repository result")
	}
	if result.Level != LevelWarn {
		t.Fatalf("expected WARN, got %s", result.Level)
	}
}

func TestServiceRun_WindowsUpdatePrerequisitesWarnWhenMissing(t *testing.T) {
	t.Parallel()

	svc := NewService(Options{
		LookPath: func(file string) (string, error) {
			if file == "git" {
				return "C:/Program Files/Git/cmd/git.exe", nil
			}
			return "", errors.New("not found")
		},
		Git: fakeRepoRootFinder{
			err: errors.New("not in repo"),
		},
		GOOS: "windows",
	})

	report := svc.Run(context.Background())
	result, ok := findResult(report.Results, "update/prerequisites")
	if !ok {
		t.Fatal("expected update/prerequisites result")
	}
	if result.Level != LevelWarn {
		t.Fatalf("expected WARN, got %s", result.Level)
	}
	if !strings.Contains(result.Message, "curl") || !strings.Contains(result.Message, "tar") || !strings.Contains(result.Message, "certutil") {
		t.Fatalf("unexpected message: %s", result.Message)
	}
}

func findResult(results []Result, name string) (Result, bool) {
	for _, r := range results {
		if r.Name == name {
			return r, true
		}
	}
	return Result{}, false
}
