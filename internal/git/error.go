package git

import (
	"errors"
	"os/exec"
	"strings"

	"github.com/takubii/git-worktree-opener/internal/execerr"
)

func buildGitCommandError(runErr error, stderrOutput string, command string, nextAction string) error {
	var execErr *exec.Error
	if errors.As(runErr, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
		return execerr.Build(
			"git",
			"command was not found in PATH",
			"Install Git and ensure `git --version` works in this shell, then retry",
		)
	}

	command = strings.TrimSpace(command)
	if command == "" {
		command = "git"
	} else {
		command = "git " + command
	}

	stderrOutput = strings.TrimSpace(stderrOutput)
	if stderrOutput == "" && runErr != nil {
		stderrOutput = runErr.Error()
	}

	return execerr.Build(command, stderrOutput, nextAction)
}
