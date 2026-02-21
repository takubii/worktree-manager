package git

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func buildGitCommandError(runErr error, stderrOutput string, command string, nextAction string) error {
	var execErr *exec.Error
	if errors.As(runErr, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
		return fmt.Errorf("`git` command was not found. Install Git and ensure it is available in PATH, then retry")
	}

	stderrOutput = strings.TrimSpace(stderrOutput)
	message := fmt.Sprintf("failed to run `git %s`", command)
	if stderrOutput != "" {
		message = fmt.Sprintf("%s: %s", message, stderrOutput)
	} else {
		message = fmt.Sprintf("%s: %v", message, runErr)
	}

	nextAction = strings.TrimSpace(nextAction)
	if nextAction == "" {
		return fmt.Errorf("%s", message)
	}

	return fmt.Errorf("%s. %s", message, nextAction)
}
