package doctor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takubii/git-worktree-opener/internal/config"
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

func TestServiceRun_TerminalProvidersAreReported(t *testing.T) {
	t.Parallel()

	svc := NewService(Options{
		LookPath: func(file string) (string, error) {
			if file == "git" || file == "curl" {
				return "ok", nil
			}
			return "", errors.New("not found")
		},
		Git:  fakeRepoRootFinder{err: errors.New("not in repo")},
		GOOS: "linux",
	})

	report := svc.Run(context.Background())
	for _, name := range []string{
		"terminal/windows-terminal",
		"terminal/cmd",
		"terminal/powershell",
		"terminal/terminal",
		"terminal/gnome-terminal",
		"terminal/wezterm",
		"terminal/iterm2",
		"terminal/ghostty",
		"terminal/warp",
		"terminal/tabby",
	} {
		if _, ok := findResult(report.Results, name); !ok {
			t.Fatalf("expected result %q", name)
		}
	}
}

func TestServiceRun_ConfiguredTerminalProviderMissingWarns(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.Open.Default = config.OpenKindTerminal
	cfg.Open.TerminalProvider = config.TerminalProviderGNOMETerminal

	svc := NewService(Options{
		LookPath: func(file string) (string, error) {
			if file == "git" || file == "curl" {
				return "ok", nil
			}
			return "", errors.New("not found")
		},
		Git:            fakeRepoRootFinder{err: errors.New("not in repo")},
		ConfigProvider: config.NewStaticProvider(cfg),
		GOOS:           "linux",
	})

	report := svc.Run(context.Background())
	gnomeResult, ok := findResult(report.Results, "terminal/gnome-terminal")
	if !ok {
		t.Fatal("expected terminal/gnome-terminal result")
	}
	if gnomeResult.Level != LevelWarn {
		t.Fatalf("expected WARN, got %s", gnomeResult.Level)
	}
	if !strings.Contains(gnomeResult.Message, "configured provider is not available") {
		t.Fatalf("unexpected message: %s", gnomeResult.Message)
	}

	warpResult, ok := findResult(report.Results, "terminal/warp")
	if !ok {
		t.Fatal("expected terminal/warp result")
	}
	if warpResult.Level != LevelOK {
		t.Fatalf("expected OK for optional missing provider, got %s", warpResult.Level)
	}
}

func TestServiceRun_LinuxGUISessionWarnsWhenTerminalDefaultIsConfigured(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.Open.Default = config.OpenKindTerminal
	cfg.Open.TerminalProvider = config.TerminalProviderGNOMETerminal

	svc := NewService(Options{
		LookPath: func(file string) (string, error) {
			if file == "git" || file == "curl" || file == "gnome-terminal" {
				return "ok", nil
			}
			return "", errors.New("not found")
		},
		Git:            fakeRepoRootFinder{err: errors.New("not in repo")},
		ConfigProvider: config.NewStaticProvider(cfg),
		GetEnv: func(string) string {
			return ""
		},
		GOOS: "linux",
	})

	report := svc.Run(context.Background())
	result, ok := findResult(report.Results, "terminal/linux-gui-session")
	if !ok {
		t.Fatal("expected terminal/linux-gui-session result")
	}
	if result.Level != LevelWarn {
		t.Fatalf("expected WARN, got %s", result.Level)
	}
	if !strings.Contains(result.Message, "not detected") {
		t.Fatalf("unexpected message: %s", result.Message)
	}
}

func TestServiceRun_WSL2BridgeWarnsWhenConfiguredAndMissing(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.Open.Default = config.OpenKindTerminal
	cfg.Open.TerminalProvider = config.TerminalProviderAuto

	svc := NewService(Options{
		LookPath: func(file string) (string, error) {
			if file == "git" || file == "curl" {
				return "ok", nil
			}
			return "", errors.New("not found")
		},
		Git:            fakeRepoRootFinder{err: errors.New("not in repo")},
		ConfigProvider: config.NewStaticProvider(cfg),
		GetEnv: func(key string) string {
			if key == "WSL_INTEROP" {
				return "/run/WSL/123.sock"
			}
			return ""
		},
		GOOS: "linux",
	})

	report := svc.Run(context.Background())
	result, ok := findResult(report.Results, "terminal/wsl2-bridge")
	if !ok {
		t.Fatal("expected terminal/wsl2-bridge result")
	}
	if result.Level != LevelWarn {
		t.Fatalf("expected WARN, got %s", result.Level)
	}
	if !strings.Contains(result.Message, "wt.exe") || !strings.Contains(result.Message, "wsl.exe") {
		t.Fatalf("unexpected message: %s", result.Message)
	}
}

func TestServiceRun_WSL2BridgeOkWhenCommandsAreAvailable(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.Open.Default = config.OpenKindTerminal
	cfg.Open.TerminalProvider = config.TerminalProviderAuto

	svc := NewService(Options{
		LookPath: func(file string) (string, error) {
			if file == "git" || file == "curl" || file == "wt.exe" || file == "wsl.exe" {
				return "ok", nil
			}
			return "", errors.New("not found")
		},
		Git:            fakeRepoRootFinder{err: errors.New("not in repo")},
		ConfigProvider: config.NewStaticProvider(cfg),
		GetEnv: func(key string) string {
			if key == "WSL_INTEROP" {
				return "/run/WSL/123.sock"
			}
			return ""
		},
		GOOS: "linux",
	})

	report := svc.Run(context.Background())
	result, ok := findResult(report.Results, "terminal/wsl2-bridge")
	if !ok {
		t.Fatal("expected terminal/wsl2-bridge result")
	}
	if result.Level != LevelOK {
		t.Fatalf("expected OK, got %s", result.Level)
	}
}

func TestServiceRun_WindowsTerminalProviderIsApplicableOnWSL2(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.Open.Default = config.OpenKindTerminal
	cfg.Open.TerminalProvider = config.TerminalProviderWindowsTerminal

	svc := NewService(Options{
		LookPath: func(file string) (string, error) {
			if file == "git" || file == "curl" || file == "wt.exe" || file == "wsl.exe" {
				return "ok", nil
			}
			return "", errors.New("not found")
		},
		Git:            fakeRepoRootFinder{err: errors.New("not in repo")},
		ConfigProvider: config.NewStaticProvider(cfg),
		GetEnv: func(key string) string {
			if key == "WSL_DISTRO_NAME" {
				return "Ubuntu"
			}
			return ""
		},
		GOOS: "linux",
	})

	report := svc.Run(context.Background())
	result, ok := findResult(report.Results, "terminal/windows-terminal")
	if !ok {
		t.Fatal("expected terminal/windows-terminal result")
	}
	if result.Level != LevelOK {
		t.Fatalf("expected OK, got %s", result.Level)
	}
	if strings.Contains(strings.ToLower(result.Message), "not applicable") {
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
