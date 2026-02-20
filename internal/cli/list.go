package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newListCmd(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List git worktrees",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			output, err := deps.Git.WorktreeListPorcelain(cmd.Context())
			if err != nil {
				return err
			}

			if _, err := io.WriteString(cmd.OutOrStdout(), output); err != nil {
				return fmt.Errorf("failed to write command output: %w", err)
			}
			return nil
		},
	}
}
