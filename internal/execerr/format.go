package execerr

import (
	"errors"
	"fmt"
	"strings"
)

// Build returns a standardized external-command failure message.
func Build(command string, reason string, nextAction string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		command = "command"
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "unknown error"
	}

	message := fmt.Sprintf("failed to run %s: %s", command, reason)
	nextAction = strings.TrimSpace(nextAction)
	if nextAction != "" {
		message = fmt.Sprintf("%s. %s", message, nextAction)
	}

	return errors.New(message)
}
