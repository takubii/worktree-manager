package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print wtm version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), deps.Version); err != nil {
				return fmt.Errorf("failed to write version output: %w", err)
			}
			return nil
		},
	}
}
