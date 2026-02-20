package opener

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type execCommandFunc func(ctx context.Context, name string, args ...string) *exec.Cmd
type lookPathFunc func(file string) (string, error)

const (
	KindSystem = "system"
	KindVSCode = "vscode"
	KindCursor = "cursor"
	KindVim    = "vim"
	KindCustom = "custom"
)

// WindowMode controls whether opener should prefer a new or reused window.
type WindowMode string

const (
	WindowNew   WindowMode = "new"
	WindowReuse WindowMode = "reuse"
)

// Opener opens a path with a selected tool.
type Opener interface {
	Open(ctx context.Context, kind string, path string, window WindowMode) error
}

type defaultOpener struct {
	goos        string
	lookPath    lookPathFunc
	execCommand execCommandFunc
}

// NewDefault returns the default opener implementation.
func NewDefault() Opener {
	return &defaultOpener{
		goos:        runtime.GOOS,
		lookPath:    exec.LookPath,
		execCommand: exec.CommandContext,
	}
}

func (o *defaultOpener) Open(ctx context.Context, kind string, path string, window WindowMode) error {
	path = o.normalizePath(path)

	normalized := strings.ToLower(strings.TrimSpace(kind))
	if normalized == "" {
		normalized = KindSystem
	}

	switch normalized {
	case KindSystem:
		return o.openSystem(ctx, path, window)
	case KindVSCode:
		return o.openVSCode(ctx, path, window)
	case KindCursor:
		return o.openCursor(ctx, path, window)
	case KindVim:
		return o.openVim(ctx, path, window)
	default:
		return fmt.Errorf("unknown opener %q. Use one of: system, vscode, cursor, vim", kind)
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

func (o *defaultOpener) openSystem(ctx context.Context, path string, window WindowMode) error {
	name, args, err := systemOpenCommand(path, window)
	if err != nil {
		return err
	}
	return o.run(ctx, name, args...)
}

func (o *defaultOpener) run(ctx context.Context, name string, args ...string) error {
	cmd := o.execCommand(ctx, name, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run opener command `%s`: %w", name, err)
	}
	return nil
}

func (o *defaultOpener) normalizePath(path string) string {
	if o.goos == "windows" {
		return filepath.Clean(filepath.FromSlash(path))
	}
	return path
}
