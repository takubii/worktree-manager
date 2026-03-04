package opener

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/takubii/git-worktree-opener/internal/execerr"
)

type execCommandFunc func(ctx context.Context, name string, args ...string) *exec.Cmd
type lookPathFunc func(file string) (string, error)

const (
	KindSystem   = "system"
	KindVSCode   = "vscode"
	KindCursor   = "cursor"
	KindVim      = "vim"
	KindTerminal = "terminal"
	KindCustom   = "custom"
)

const (
	TerminalProviderAuto            = "auto"
	TerminalProviderWindowsTerminal = "windows-terminal"
	TerminalProviderCMD             = "cmd"
	TerminalProviderPowerShell      = "powershell"
	TerminalProviderMacTerminal     = "terminal"
	TerminalProviderGNOMETerminal   = "gnome-terminal"
	TerminalProviderWezTerm         = "wezterm"
	TerminalProviderITerm2          = "iterm2"
	TerminalProviderGhostty         = "ghostty"
	TerminalProviderWarp            = "warp"
	TerminalProviderTabby           = "tabby"
)

const (
	terminalProviderXTerminalEmulator = "x-terminal-emulator"
	terminalProviderXTerm             = "xterm"
)

// WindowMode controls whether opener should prefer a new or reused window.
type WindowMode string

const (
	WindowNew   WindowMode = "new"
	WindowReuse WindowMode = "reuse"
)

// TmuxMode controls tmux optimization mode for terminal workflows.
type TmuxMode string

const (
	TmuxModeAuto   TmuxMode = "auto"
	TmuxModeOff    TmuxMode = "off"
	TmuxModeSplit  TmuxMode = "split"
	TmuxModeWindow TmuxMode = "window"
)

// Opener opens a path with a selected tool.
type Opener interface {
	Open(ctx context.Context, kind string, path string, window WindowMode) error
}

// OpenRequest contains opener execution parameters.
type OpenRequest struct {
	Kind             string
	Path             string
	Window           WindowMode
	TerminalProvider string
	TmuxMode         TmuxMode
}

// OpenResult contains opener execution details.
type OpenResult struct {
	Provider string
	Warnings []string
}

type defaultOpener struct {
	goos        string
	getEnv      func(key string) string
	lookPath    lookPathFunc
	execCommand execCommandFunc
}

// NewDefault returns the default opener implementation.
func NewDefault() Opener {
	return &defaultOpener{
		goos:        runtime.GOOS,
		getEnv:      os.Getenv,
		lookPath:    exec.LookPath,
		execCommand: exec.CommandContext,
	}
}

func (o *defaultOpener) Open(ctx context.Context, kind string, path string, window WindowMode) error {
	_, err := o.OpenWithResult(ctx, OpenRequest{
		Kind:             kind,
		Path:             path,
		Window:           window,
		TerminalProvider: TerminalProviderAuto,
		TmuxMode:         TmuxModeAuto,
	})
	return err
}

// OpenWithResult opens a path and returns provider metadata and warnings.
func (o *defaultOpener) OpenWithResult(ctx context.Context, req OpenRequest) (OpenResult, error) {
	path := req.Path
	path = o.normalizePath(path)

	normalized := strings.ToLower(strings.TrimSpace(req.Kind))
	if normalized == "" {
		normalized = KindSystem
	}

	switch normalized {
	case KindSystem:
		if err := o.openSystem(ctx, path, req.Window); err != nil {
			return OpenResult{}, err
		}
		return OpenResult{Provider: KindSystem}, nil
	case KindVSCode:
		if err := o.openVSCode(ctx, path, req.Window); err != nil {
			return OpenResult{}, err
		}
		return OpenResult{Provider: KindVSCode}, nil
	case KindCursor:
		if err := o.openCursor(ctx, path, req.Window); err != nil {
			return OpenResult{}, err
		}
		return OpenResult{Provider: KindCursor}, nil
	case KindVim:
		if err := o.openVim(ctx, path, req.Window); err != nil {
			return OpenResult{}, err
		}
		return OpenResult{Provider: KindVim}, nil
	case KindTerminal:
		return o.openTerminal(ctx, path, req.Window, req.TerminalProvider, req.TmuxMode)
	default:
		return OpenResult{}, fmt.Errorf("unknown opener %q. Use one of: system, vscode, cursor, vim, terminal", req.Kind)
	}
}

func ParseWindowMode(raw string) (WindowMode, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(WindowNew):
		return WindowNew, nil
	case string(WindowReuse):
		return WindowReuse, nil
	default:
		return "", fmt.Errorf("invalid window mode %q. Use one of: new, reuse", raw)
	}
}

func ParseTmuxMode(raw string) (TmuxMode, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(TmuxModeAuto):
		return TmuxModeAuto, nil
	case string(TmuxModeOff):
		return TmuxModeOff, nil
	case string(TmuxModeSplit):
		return TmuxModeSplit, nil
	case string(TmuxModeWindow):
		return TmuxModeWindow, nil
	default:
		return "", fmt.Errorf("invalid tmux mode %q. Use one of: auto, off, split, window", raw)
	}
}

func (o *defaultOpener) openSystem(ctx context.Context, path string, window WindowMode) error {
	name, args, err := systemOpenCommand(path, window)
	if err != nil {
		return err
	}
	return o.run(ctx, name, args...)
}

func (o *defaultOpener) run(ctx context.Context, name string, args ...string) error {
	return o.runInDir(ctx, "", name, args...)
}

func (o *defaultOpener) runInDir(ctx context.Context, dir string, name string, args ...string) error {
	cmd := o.execCommand(ctx, name, args...)
	if strings.TrimSpace(dir) != "" {
		cmd.Dir = dir
	}
	if err := cmd.Run(); err != nil {
		command := strings.TrimSpace(name)
		if len(args) > 0 {
			command = strings.TrimSpace(command + " " + strings.Join(args, " "))
		}
		return execerr.Build(command, err.Error(), "verify opener command availability and arguments, then retry")
	}
	return nil
}

func (o *defaultOpener) normalizePath(path string) string {
	if o.goos == "windows" {
		return filepath.Clean(filepath.FromSlash(path))
	}
	return path
}
