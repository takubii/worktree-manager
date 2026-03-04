package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/takubii/git-worktree-opener/internal/config"
)

type repoRootFinder interface {
	RepoRoot(ctx context.Context) (string, error)
}

// Options customizes doctor service behavior.
type Options struct {
	LookPath       func(file string) (string, error)
	Git            repoRootFinder
	ConfigProvider config.Provider
	UserConfigDir  func() (string, error)
	ReadFile       func(path string) ([]byte, error)
	Stat           func(name string) (os.FileInfo, error)
	GetEnv         func(key string) string
	GOOS           string
}

type service struct {
	lookPath       func(file string) (string, error)
	git            repoRootFinder
	configProvider config.Provider
	userConfigDir  func() (string, error)
	readFile       func(path string) ([]byte, error)
	stat           func(name string) (os.FileInfo, error)
	getEnv         func(key string) string
	goos           string
}

// NewService returns the default doctor service.
func NewService(opts Options) Service {
	if opts.UserConfigDir == nil {
		opts.UserConfigDir = os.UserConfigDir
	}
	if opts.ReadFile == nil {
		opts.ReadFile = os.ReadFile
	}
	if opts.Stat == nil {
		opts.Stat = os.Stat
	}
	if opts.GetEnv == nil {
		opts.GetEnv = os.Getenv
	}
	if strings.TrimSpace(opts.GOOS) == "" {
		opts.GOOS = runtime.GOOS
	}

	return &service{
		lookPath:       opts.LookPath,
		git:            opts.Git,
		configProvider: opts.ConfigProvider,
		userConfigDir:  opts.UserConfigDir,
		readFile:       opts.ReadFile,
		stat:           opts.Stat,
		getEnv:         opts.GetEnv,
		goos:           strings.TrimSpace(opts.GOOS),
	}
}

func (s *service) Run(ctx context.Context) Report {
	results := make([]Result, 0, 24)
	hasCritical := false

	gitOK := s.lookPath != nil
	if s.lookPath == nil {
		results = append(results, Result{
			Name:       "git",
			Level:      LevelCritical,
			Message:    "cannot check `git` command because lookPath is not configured",
			NextAction: "configure lookPath dependency, then run `wto doctor` again",
		})
		hasCritical = true
		gitOK = false
	} else if _, err := s.lookPath("git"); err != nil {
		results = append(results, Result{
			Name:       "git",
			Level:      LevelCritical,
			Message:    "`git` command was not found in PATH",
			NextAction: "install Git and ensure `git --version` works in this shell, then retry",
		})
		hasCritical = true
		gitOK = false
	} else {
		results = append(results, Result{
			Name:       "git",
			Level:      LevelOK,
			Message:    "`git` command is available",
			NextAction: "no action required",
		})
	}

	repoRoot, repoErr := s.repoRoot(ctx, gitOK)
	terminalSelection := s.selectedTerminalSelection(ctx)
	results = append(results, s.repoResult(repoRoot, repoErr)...)
	results = append(results, s.openerResults()...)
	results = append(results, s.terminalProviderResults(terminalSelection.Provider)...)
	results = append(results, s.terminalRuntimeResults(terminalSelection)...)
	results = append(results, s.configResults(repoRoot, repoErr)...)
	results = append(results, s.updatePrerequisiteResult()...)

	return Report{
		Results:     results,
		HasCritical: hasCritical,
	}
}

type terminalSelection struct {
	Enabled  bool
	Provider string
}

func (s *service) selectedTerminalSelection(ctx context.Context) terminalSelection {
	if s.configProvider == nil {
		return terminalSelection{}
	}

	cfg := s.configProvider.Load(ctx)
	openDefault := strings.ToLower(strings.TrimSpace(cfg.Open.Default))
	if openDefault != config.OpenKindTerminal {
		return terminalSelection{}
	}

	provider := strings.ToLower(strings.TrimSpace(cfg.Open.TerminalProvider))
	if provider == "" {
		provider = config.TerminalProviderAuto
	}

	return terminalSelection{
		Enabled:  true,
		Provider: provider,
	}
}

type terminalProviderCheck struct {
	Provider string
	Command  string
	OnlyOS   string
	AllowWSL bool
	AppPaths []string
}

func (s *service) terminalProviderResults(selectedProvider string) []Result {
	checks := []terminalProviderCheck{
		{Provider: config.TerminalProviderWindowsTerminal, Command: "wt", OnlyOS: "windows", AllowWSL: true},
		{Provider: config.TerminalProviderCMD, Command: "cmd", OnlyOS: "windows"},
		{Provider: config.TerminalProviderPowerShell, Command: "powershell", OnlyOS: "windows"},
		{
			Provider: config.TerminalProviderMacTerminal,
			Command:  "open",
			OnlyOS:   "darwin",
			AppPaths: []string{
				"/System/Applications/Utilities/Terminal.app",
				"/Applications/Utilities/Terminal.app",
			},
		},
		{Provider: config.TerminalProviderGNOMETerminal, Command: "gnome-terminal", OnlyOS: "linux"},
		{Provider: config.TerminalProviderWezTerm, Command: "wezterm"},
		{
			Provider: config.TerminalProviderITerm2,
			Command:  "open",
			OnlyOS:   "darwin",
			AppPaths: []string{
				"/Applications/iTerm.app",
			},
		},
		{
			Provider: config.TerminalProviderGhostty,
			Command:  "ghostty",
			AppPaths: []string{
				"/Applications/Ghostty.app",
			},
		},
		{
			Provider: config.TerminalProviderWarp,
			Command:  "warp",
			AppPaths: []string{
				"/Applications/Warp.app",
			},
		},
		{
			Provider: config.TerminalProviderTabby,
			Command:  "tabby",
			AppPaths: []string{
				"/Applications/Tabby.app",
			},
		},
	}

	results := make([]Result, 0, len(checks))
	for _, check := range checks {
		name := "terminal/" + check.Provider
		if !s.isTerminalProviderApplicable(check) {
			level := LevelOK
			message := fmt.Sprintf("provider is not applicable on %s (optional)", s.goos)
			nextAction := "no action required"
			if selectedProvider == check.Provider {
				level = LevelWarn
				message = fmt.Sprintf("provider is configured but not supported on %s", s.goos)
				nextAction = "choose a supported terminal provider for this OS or set open.terminalProvider to auto"
			}
			results = append(results, Result{
				Name:       name,
				Level:      level,
				Message:    message,
				NextAction: nextAction,
			})
			continue
		}

		available, reason := s.isTerminalProviderAvailable(check)
		if available {
			results = append(results, Result{
				Name:       name,
				Level:      LevelOK,
				Message:    "provider command/app is available",
				NextAction: "no action required",
			})
			continue
		}

		level := LevelOK
		message := fmt.Sprintf("provider is not installed (optional): %s", reason)
		nextAction := "no action required"
		if selectedProvider == check.Provider {
			level = LevelWarn
			message = fmt.Sprintf("configured provider is not available: %s", reason)
			nextAction = "install the configured terminal provider or switch open.terminalProvider to auto"
		}

		results = append(results, Result{
			Name:       name,
			Level:      level,
			Message:    message,
			NextAction: nextAction,
		})
	}

	return results
}

func (s *service) terminalRuntimeResults(selection terminalSelection) []Result {
	return []Result{
		s.linuxGUISessionResult(selection),
		s.wsl2BridgeResult(selection),
	}
}

func (s *service) linuxGUISessionResult(selection terminalSelection) Result {
	if s.goos != "linux" {
		return Result{
			Name:       "terminal/linux-gui-session",
			Level:      LevelOK,
			Message:    fmt.Sprintf("check is not applicable on %s (optional)", s.goos),
			NextAction: "no action required",
		}
	}

	if s.hasLinuxGUISession() {
		return Result{
			Name:       "terminal/linux-gui-session",
			Level:      LevelOK,
			Message:    "Linux GUI session is available (`DISPLAY` or `WAYLAND_DISPLAY` is set)",
			NextAction: "no action required",
		}
	}

	if !selection.Enabled {
		return Result{
			Name:       "terminal/linux-gui-session",
			Level:      LevelOK,
			Message:    "Linux GUI session is not detected (optional)",
			NextAction: "no action required",
		}
	}

	provider := strings.ToLower(strings.TrimSpace(selection.Provider))
	if s.isWSL2() && terminalProviderCanUseWSLBridge(provider) {
		return Result{
			Name:       "terminal/linux-gui-session",
			Level:      LevelOK,
			Message:    "Linux GUI session is not detected, but WSL2 Windows Terminal bridge can be used",
			NextAction: "ensure `wt.exe` is available for bridge mode, or enable WSLg/X11 for Linux GUI terminals",
		}
	}

	return Result{
		Name:       "terminal/linux-gui-session",
		Level:      LevelWarn,
		Message:    "Linux GUI session is not detected (`DISPLAY`/`WAYLAND_DISPLAY` are empty)",
		NextAction: "start a Linux GUI terminal session, or use `wto enter --shell` / `wto enter --print-cd`",
	}
}

func (s *service) wsl2BridgeResult(selection terminalSelection) Result {
	if s.goos != "linux" {
		return Result{
			Name:       "terminal/wsl2-bridge",
			Level:      LevelOK,
			Message:    fmt.Sprintf("check is not applicable on %s (optional)", s.goos),
			NextAction: "no action required",
		}
	}

	if !s.isWSL2() {
		return Result{
			Name:       "terminal/wsl2-bridge",
			Level:      LevelOK,
			Message:    "WSL2 bridge check is not applicable outside WSL2 (optional)",
			NextAction: "no action required",
		}
	}

	bridgeRequired := selection.Enabled && terminalProviderCanUseWSLBridge(selection.Provider)
	if s.lookPath == nil {
		if bridgeRequired {
			return Result{
				Name:       "terminal/wsl2-bridge",
				Level:      LevelWarn,
				Message:    "cannot verify WSL2 bridge commands because lookPath is not configured",
				NextAction: "configure lookPath dependency, then run `wto doctor` again",
			}
		}
		return Result{
			Name:       "terminal/wsl2-bridge",
			Level:      LevelOK,
			Message:    "WSL2 bridge commands were not checked (optional): lookPath is not configured",
			NextAction: "no action required",
		}
	}

	missing := make([]string, 0, 2)
	for _, cmd := range []string{"wt.exe", "wsl.exe"} {
		if _, err := s.lookPath(cmd); err != nil {
			missing = append(missing, cmd)
		}
	}
	if len(missing) == 0 {
		return Result{
			Name:       "terminal/wsl2-bridge",
			Level:      LevelOK,
			Message:    "WSL2 bridge commands are available (`wt.exe`, `wsl.exe`)",
			NextAction: "no action required",
		}
	}

	message := fmt.Sprintf("WSL2 bridge commands are missing (optional): %s", strings.Join(missing, ", "))
	nextAction := "no action required"
	level := LevelOK
	if bridgeRequired {
		level = LevelWarn
		message = fmt.Sprintf("WSL2 bridge commands are missing: %s", strings.Join(missing, ", "))
		nextAction = "install Windows Terminal and ensure both `wt.exe` and `wsl.exe` are available in WSL2 PATH"
	}

	return Result{
		Name:       "terminal/wsl2-bridge",
		Level:      level,
		Message:    message,
		NextAction: nextAction,
	}
}

func terminalProviderCanUseWSLBridge(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", config.TerminalProviderAuto, config.TerminalProviderWindowsTerminal:
		return true
	default:
		return false
	}
}

func (s *service) isTerminalProviderApplicable(check terminalProviderCheck) bool {
	if check.Provider == config.TerminalProviderWindowsTerminal {
		if s.goos == "windows" {
			return true
		}
		return check.AllowWSL && s.goos == "linux" && s.isWSL2()
	}

	if check.OnlyOS == "" {
		return true
	}
	return check.OnlyOS == s.goos
}

func (s *service) isWSL2() bool {
	if s.goos != "linux" {
		return false
	}
	return strings.TrimSpace(s.env("WSL_INTEROP")) != "" || strings.TrimSpace(s.env("WSL_DISTRO_NAME")) != ""
}

func (s *service) hasLinuxGUISession() bool {
	return strings.TrimSpace(s.env("DISPLAY")) != "" || strings.TrimSpace(s.env("WAYLAND_DISPLAY")) != ""
}

func (s *service) env(key string) string {
	if s.getEnv == nil {
		return ""
	}
	return s.getEnv(key)
}

func (s *service) isTerminalProviderAvailable(check terminalProviderCheck) (bool, string) {
	if s.lookPath == nil {
		return false, "lookPath is not configured"
	}

	command := check.Command
	if check.Provider == config.TerminalProviderWindowsTerminal && s.goos == "linux" && s.isWSL2() {
		command = "wt.exe"
	}

	if check.Command != "" {
		if _, err := s.lookPath(command); err == nil {
			return true, "command is available"
		}
	}

	for _, path := range check.AppPaths {
		if path == "" {
			continue
		}
		if _, err := s.stat(path); err == nil {
			return true, "application path exists"
		}
	}

	if command != "" {
		return false, fmt.Sprintf("command `%s` was not found in PATH", command)
	}

	return false, "provider command is unknown"
}

func (s *service) repoRoot(ctx context.Context, gitOK bool) (string, error) {
	if !gitOK {
		return "", errors.New("git command is unavailable")
	}
	if s.git == nil {
		return "", errors.New("git client is not configured")
	}
	return s.git.RepoRoot(ctx)
}

func (s *service) repoResult(repoRoot string, repoErr error) []Result {
	if repoErr == nil {
		return []Result{{
			Name:       "repository",
			Level:      LevelOK,
			Message:    fmt.Sprintf("Git repository is available at %s", repoRoot),
			NextAction: "no action required",
		}}
	}

	return []Result{{
		Name:       "repository",
		Level:      LevelWarn,
		Message:    fmt.Sprintf("Git repository check failed: %v", repoErr),
		NextAction: "run this command inside a Git repository when you want repo-specific checks",
	}}
}

func (s *service) openerResults() []Result {
	results := make([]Result, 0, 2)
	for _, cmd := range []string{"code", "cursor"} {
		if s.lookPath == nil {
			results = append(results, Result{
				Name:       "opener/" + cmd,
				Level:      LevelWarn,
				Message:    "cannot check command availability because lookPath is not configured",
				NextAction: "configure lookPath dependency, then run `wto doctor` again",
			})
			continue
		}
		if _, err := s.lookPath(cmd); err != nil {
			results = append(results, Result{
				Name:       "opener/" + cmd,
				Level:      LevelWarn,
				Message:    fmt.Sprintf("`%s` command is not available", cmd),
				NextAction: fmt.Sprintf("install `%s` CLI or use `--open system`", cmd),
			})
			continue
		}
		results = append(results, Result{
			Name:       "opener/" + cmd,
			Level:      LevelOK,
			Message:    fmt.Sprintf("`%s` command is available", cmd),
			NextAction: "no action required",
		})
	}
	return results
}

func (s *service) configResults(repoRoot string, repoErr error) []Result {
	results := make([]Result, 0, 2)

	globalPath, err := config.GlobalConfigPath(s.userConfigDir)
	if err != nil {
		results = append(results, Result{
			Name:       "config/global",
			Level:      LevelWarn,
			Message:    fmt.Sprintf("failed to resolve global config path: %v", err),
			NextAction: "ensure the user config directory is accessible, then retry",
		})
	} else {
		results = append(results, s.validateConfigFile("config/global", globalPath))
	}

	if repoErr != nil || strings.TrimSpace(repoRoot) == "" {
		results = append(results, Result{
			Name:       "config/repo",
			Level:      LevelWarn,
			Message:    "repository config check was skipped (repository root unavailable)",
			NextAction: "run `wto doctor` inside a Git repository to validate .wtoconfig.json",
		})
		return results
	}

	repoPath := config.RepoConfigPath(repoRoot)
	results = append(results, s.validateConfigFile("config/repo", repoPath))
	return results
}

func (s *service) validateConfigFile(name string, path string) Result {
	info, err := s.stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Result{
				Name:       name,
				Level:      LevelOK,
				Message:    fmt.Sprintf("config file not found at %s (0-config mode)", path),
				NextAction: "no action required",
			}
		}
		return Result{
			Name:       name,
			Level:      LevelWarn,
			Message:    fmt.Sprintf("failed to inspect config file %s: %v", path, err),
			NextAction: "check file permissions and retry",
		}
	}

	if info.IsDir() {
		return Result{
			Name:       name,
			Level:      LevelWarn,
			Message:    fmt.Sprintf("config path points to a directory: %s", path),
			NextAction: "replace the directory with a JSON config file and retry",
		}
	}

	if err := config.ValidateFile(path, s.readFile); err != nil {
		return Result{
			Name:       name,
			Level:      LevelWarn,
			Message:    fmt.Sprintf("config file is invalid at %s: %v", path, err),
			NextAction: "fix JSON keys/values and retry",
		}
	}

	return Result{
		Name:       name,
		Level:      LevelOK,
		Message:    fmt.Sprintf("config file is valid at %s", path),
		NextAction: "no action required",
	}
}

func (s *service) updatePrerequisiteResult() []Result {
	if s.lookPath == nil {
		return []Result{{
			Name:       "update/prerequisites",
			Level:      LevelWarn,
			Message:    "cannot check update prerequisites because lookPath is not configured",
			NextAction: "configure lookPath dependency, then run `wto doctor` again",
		}}
	}

	if s.goos == "windows" {
		missing := make([]string, 0, 3)
		for _, cmd := range []string{"curl", "tar", "certutil"} {
			if _, err := s.lookPath(cmd); err != nil {
				missing = append(missing, cmd)
			}
		}
		if len(missing) > 0 {
			return []Result{{
				Name:       "update/prerequisites",
				Level:      LevelWarn,
				Message:    fmt.Sprintf("missing commands for Windows update flow: %s", strings.Join(missing, ", ")),
				NextAction: "install missing commands or use an environment where they are available",
			}}
		}
		return []Result{{
			Name:       "update/prerequisites",
			Level:      LevelOK,
			Message:    "required commands for Windows update flow are available (curl, tar, certutil)",
			NextAction: "no action required",
		}}
	}

	if _, err := s.lookPath("curl"); err == nil {
		return []Result{{
			Name:       "update/prerequisites",
			Level:      LevelOK,
			Message:    "update prerequisite is available (`curl`)",
			NextAction: "no action required",
		}}
	}
	if _, err := s.lookPath("wget"); err == nil {
		return []Result{{
			Name:       "update/prerequisites",
			Level:      LevelOK,
			Message:    "update prerequisite is available (`wget`)",
			NextAction: "no action required",
		}}
	}

	return []Result{{
		Name:       "update/prerequisites",
		Level:      LevelWarn,
		Message:    "missing update prerequisites (`curl` or `wget`)",
		NextAction: "install curl or wget and retry",
	}}
}
