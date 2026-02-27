package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/takubii/git-worktree-opener/internal/updater"
)

func newUpdateCmd(deps Dependencies) *cobra.Command {
	var targetVersion string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update wto from GitHub Releases",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if deps.Updater == nil {
				return fmt.Errorf("update service is not configured")
			}
			tracef(cmd.Context(), "update: requested version=%q", targetVersion)

			result, err := deps.Updater.Update(cmd.Context(), updater.Request{
				Version: targetVersion,
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
			})
			if err != nil {
				return err
			}
			tracef(cmd.Context(), "update: completed async=%v", result.Async)

			if result.Async {
				if _, err := fmt.Fprintln(
					cmd.OutOrStdout(),
					"update started in background. Wait until it finishes, then run `wto --help`.",
				); err != nil {
					return fmt.Errorf("failed to write update status: %w", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&targetVersion, "version", "", "install a specific release tag (for example: v0.1.0)")

	return cmd
}
