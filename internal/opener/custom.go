package opener

import (
	"context"
	"fmt"
	"strings"
)

// OpenCustom executes a custom opener command where {path} is replaced.
func OpenCustom(ctx context.Context, run func(context.Context, string, ...string) error, commandTemplate, path string) error {
	commandTemplate = strings.TrimSpace(commandTemplate)
	if commandTemplate == "" {
		return fmt.Errorf("custom opener command is empty")
	}

	command := strings.ReplaceAll(commandTemplate, "{path}", path)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("custom opener command is invalid")
	}

	return run(ctx, parts[0], parts[1:]...)
}
