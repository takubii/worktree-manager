package cli

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/takubii/git-worktree-opener/internal/execerr"
)

// AfterRunner executes a follow-up command for `wto open --after`.
type AfterRunner interface {
	Run(ctx context.Context, commandTemplate string, path string) error
}

type defaultAfterRunner struct {
	goos           string
	commandContext enterCommandContextFunc
	runCommand     func(cmd *exec.Cmd) error
}

func newDefaultAfterRunner() AfterRunner {
	return &defaultAfterRunner{
		goos:           runtime.GOOS,
		commandContext: exec.CommandContext,
		runCommand:     (*exec.Cmd).Run,
	}
}

func (r *defaultAfterRunner) Run(ctx context.Context, commandTemplate string, path string) error {
	commandTemplate = strings.TrimSpace(commandTemplate)
	path = strings.TrimSpace(path)
	if commandTemplate == "" {
		return execerr.Build("follow-up command", "command template is empty", "set `--after <command>` and retry")
	}
	if path == "" {
		return execerr.Build("follow-up command", "selected path is empty", "select a valid worktree and retry")
	}

	finalCommand := commandTemplate
	quotedPath := quoteForShell(r.goos, path)
	if strings.Contains(finalCommand, "{path}") {
		finalCommand = strings.ReplaceAll(finalCommand, "{path}", quotedPath)
	} else {
		finalCommand += " " + quotedPath
	}

	var name string
	var args []string
	if r.goos == "windows" {
		name = "cmd"
		args = []string{"/c", finalCommand}
	} else {
		name = "sh"
		args = []string{"-c", finalCommand}
	}

	cmd := r.commandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := r.runCommand(cmd); err != nil {
		return execerr.Build(commandTemplate, err.Error(), "verify `--after` command syntax and retry")
	}

	return nil
}

func quoteForShell(goos string, path string) string {
	if goos == "windows" {
		return `"` + strings.ReplaceAll(path, `"`, `""`) + `"`
	}
	return shellSingleQuote(path)
}
