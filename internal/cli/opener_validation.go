package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takubii/git-worktree-opener/internal/opener"
)

func validateExplicitOpenerAvailability(cmd *cobra.Command, lookPath func(file string) (string, error), openerName string) error {
	if cmd == nil || !cmd.Flags().Changed("open") {
		return nil
	}
	if lookPath == nil {
		return fmt.Errorf("failed to validate opener availability: lookPath is not configured")
	}

	normalized := strings.ToLower(strings.TrimSpace(openerName))
	switch normalized {
	case opener.KindVSCode:
		if _, err := lookPath("code"); err != nil {
			return fmt.Errorf("`--open vscode` was requested but `code` command was not found. Install VS Code CLI and retry, or use `--open system`")
		}
	case opener.KindCursor:
		if _, err := lookPath("cursor"); err != nil {
			return fmt.Errorf("`--open cursor` was requested but `cursor` command was not found. Install Cursor CLI and retry, or use `--open system`")
		}
	}

	return nil
}
