package cli

import "github.com/spf13/cobra"

func newEnterCmd(_ Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "enter",
		Short: "Select a worktree and print its path for terminal workflows",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return nil
		},
	}
}
