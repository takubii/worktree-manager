package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newConfigCmd(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage wto configuration",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newConfigInitCmd(deps))
	cmd.AddCommand(newConfigShowCmd(deps))

	return cmd
}

func newConfigInitCmd(deps Dependencies) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize global config file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := deps.Config.InitGlobal(force)
			if err != nil {
				return err
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "initialized config file: %s\n", path); err != nil {
				return fmt.Errorf("failed to write command output: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "overwrite the global config file if it already exists")
	return cmd
}

func newConfigShowCmd(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show effective configuration as JSON",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := deps.Config.Load(cmd.Context())

			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(cfg); err != nil {
				return fmt.Errorf("failed to write config output: %w", err)
			}
			return nil
		},
	}
}
