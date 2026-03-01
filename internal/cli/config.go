package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/takubii/git-worktree-opener/internal/config"
)

func newConfigCmd(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage wto configuration",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newConfigInitCmd(deps))
	cmd.AddCommand(newConfigShowCmd(deps))
	cmd.AddCommand(newConfigPathCmd(deps))

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

type configPathEntry struct {
	Path      string `json:"path"`
	Exists    bool   `json:"exists"`
	Available bool   `json:"available"`
}

type configPathOutput struct {
	Global configPathEntry `json:"global"`
	Repo   configPathEntry `json:"repo"`
}

func newConfigPathCmd(deps Dependencies) *cobra.Command {
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "path",
		Short: "Show global/repo config file paths",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			globalPath, err := config.GlobalConfigPath(os.UserConfigDir)
			if err != nil {
				return fmt.Errorf("failed to resolve global config path: %w", err)
			}

			out := configPathOutput{
				Global: configPathEntry{
					Path:      globalPath,
					Exists:    pathExists(globalPath),
					Available: true,
				},
				Repo: configPathEntry{
					Available: false,
				},
			}

			repoRoot, repoErr := deps.Git.RepoRoot(cmd.Context())
			if repoErr == nil {
				repoPath := config.RepoConfigPath(repoRoot)
				if repoPath != "" {
					out.Repo = configPathEntry{
						Path:      repoPath,
						Exists:    pathExists(repoPath),
						Available: true,
					}
				}
			}

			if outputJSON {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(out); err != nil {
					return fmt.Errorf("failed to write config path output: %w", err)
				}
				return nil
			}

			if _, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"global: %s (%s)\n",
				out.Global.Path,
				existenceLabel(out.Global.Exists),
			); err != nil {
				return fmt.Errorf("failed to write config path output: %w", err)
			}

			if out.Repo.Available {
				if _, err := fmt.Fprintf(
					cmd.OutOrStdout(),
					"repo: %s (%s)\n",
					out.Repo.Path,
					existenceLabel(out.Repo.Exists),
				); err != nil {
					return fmt.Errorf("failed to write config path output: %w", err)
				}
				return nil
			}

			if _, err := fmt.Fprintln(cmd.OutOrStdout(), "repo: not in a git repository"); err != nil {
				return fmt.Errorf("failed to write config path output: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "print config paths as JSON")
	return cmd
}

func pathExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func existenceLabel(exists bool) string {
	if exists {
		return "exists"
	}
	return "missing"
}
