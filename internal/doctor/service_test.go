package doctor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeRepoRootFinder struct {
	root string
	err  error
}

func (f fakeRepoRootFinder) RepoRoot(context.Context) (string, error) {
	return f.root, f.err
}

func TestServiceRun_ReportsGitRepositoryConfigAndUpdateChecks(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	repoRoot := filepath.Join(tmp, "repo")
	globalPath := filepath.Join(tmp, "worktree-manager", "config.json")
	repoPath := filepath.Join(repoRoot, ".wtmconfig.json")

	service := NewService(Options{
		LookPath: func(file string) (string, error) {
			switch file {
			case "git", "curl":
				return "/bin/" + file, nil
			default:
				return "", errors.New(file + " not found")
			}
		},
		Git:           fakeRepoRootFinder{root: repoRoot},
		UserConfigDir: func() (string, error) { return tmp, nil },
		Stat: func(path string) (os.FileInfo, error) {
			switch path {
			case globalPath, repoPath:
				return configFileInfo{}, nil
			default:
				return configFileInfo{}, errors.New("not found")
			}
		},
		ReadFile: func(string) ([]byte, error) {
			return []byte(`{"create":{"fetch":false},"remove":{"deleteBranch":"safe"}}`), nil
		},
		GOOS: "linux",
	})

	report := service.Run(context.Background())
	if report.HasCritical {
		t.Fatal("did not expect critical issues")
	}
	for _, name := range []string{"git", "repository", "config/global", "config/repo", "update/prerequisites"} {
		if _, ok := findResult(report.Results, name); !ok {
			t.Fatalf("expected result %q in %+v", name, report.Results)
		}
	}
	if _, ok := findResult(report.Results, "terminal/windows-terminal"); ok {
		t.Fatal("terminal checks should not be reported")
	}
}

func TestServiceRun_GitMissingIsCritical(t *testing.T) {
	t.Parallel()

	service := NewService(Options{
		LookPath: func(string) (string, error) { return "", errors.New("not found") },
		Git:      fakeRepoRootFinder{err: errors.New("not in repo")},
		GOOS:     "linux",
	})

	report := service.Run(context.Background())
	if !report.HasCritical {
		t.Fatal("expected critical issue")
	}
	result, ok := findResult(report.Results, "git")
	if !ok {
		t.Fatal("expected git result")
	}
	if result.Level != LevelCritical {
		t.Fatalf("unexpected git level: %s", result.Level)
	}
}

type configFileInfo struct{}

func (configFileInfo) Name() string       { return "config.json" }
func (configFileInfo) Size() int64        { return 1 }
func (configFileInfo) Mode() os.FileMode  { return 0o644 }
func (configFileInfo) ModTime() time.Time { return time.Time{} }
func (configFileInfo) IsDir() bool        { return false }
func (configFileInfo) Sys() any           { return nil }

func findResult(results []Result, name string) (Result, bool) {
	for _, result := range results {
		if result.Name == name {
			return result, true
		}
	}
	return Result{}, false
}
