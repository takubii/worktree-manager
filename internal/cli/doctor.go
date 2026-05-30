package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/takubii/worktree-manager/internal/doctor"
)

func newDoctorCmd(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run environment diagnostics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if deps.Doctor == nil {
				return fmt.Errorf("doctor service is not configured")
			}

			report := deps.Doctor.Run(cmd.Context())
			for _, result := range report.Results {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), doctor.FormatLine(result)); err != nil {
					return fmt.Errorf("failed to write doctor output: %w", err)
				}
			}

			if report.HasCritical {
				return fmt.Errorf("doctor found critical issues. Resolve [CRIT] items and retry")
			}

			return nil
		},
	}
}
