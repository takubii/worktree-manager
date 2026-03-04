package cli

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/takubii/git-worktree-opener/internal/execerr"
	"github.com/takubii/git-worktree-opener/internal/opener"
)

type enterCommandContextFunc func(ctx context.Context, name string, args ...string) *exec.Cmd

// EnterRunner provides shell-specific behavior for `wto enter`.
type EnterRunner interface {
	FormatCDHints(path string) []string
	StartShell(ctx context.Context, path string, tmuxMode opener.TmuxMode) error
}

type defaultEnterRunner struct {
	goos           string
	getEnv         func(key string) string
	commandContext enterCommandContextFunc
	runCommand     func(cmd *exec.Cmd) error
}

func newDefaultEnterRunner() EnterRunner {
	return &defaultEnterRunner{
		goos:           runtime.GOOS,
		getEnv:         os.Getenv,
		commandContext: exec.CommandContext,
		runCommand:     (*exec.Cmd).Run,
	}
}

func (r *defaultEnterRunner) FormatCDHints(path string) []string {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}

	if r.goos == "windows" {
		return []string{
			`cmd.exe: cd /d "` + strings.ReplaceAll(path, `"`, `""`) + `"`,
			`PowerShell: Set-Location -LiteralPath '` + strings.ReplaceAll(path, `'`, `''`) + `'`,
		}
	}

	return []string{
		"cd " + shellSingleQuote(path),
	}
}

func (r *defaultEnterRunner) StartShell(ctx context.Context, path string, _ opener.TmuxMode) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return execerr.Build("interactive shell", "selected worktree path is empty", "select a valid worktree and retry")
	}

	var shell string
	switch r.goos {
	case "windows":
		shell = strings.TrimSpace(r.getEnv("ComSpec"))
		if shell == "" {
			shell = "cmd.exe"
		}
	default:
		shell = strings.TrimSpace(r.getEnv("SHELL"))
		if shell == "" {
			shell = "sh"
		}
	}

	cmd := r.commandContext(ctx, shell)
	cmd.Dir = path
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := r.runCommand(cmd); err != nil {
		return execerr.Build(
			shell,
			err.Error(),
			"use `wto enter --print-cd` to get a manual cd command, then retry",
		)
	}

	return nil
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
