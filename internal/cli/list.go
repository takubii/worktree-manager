package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takubii/git-worktree-opener/internal/config"
	"github.com/takubii/git-worktree-opener/internal/git"
)

func newListCmd(deps Dependencies) *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List git worktrees",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			listFormat, err := parseListFormat(outputFormat)
			if err != nil {
				return err
			}

			output, err := deps.Git.WorktreeListPorcelain(cmd.Context())
			if err != nil {
				return err
			}

			if listFormat == config.ListFormatRaw {
				if _, err := io.WriteString(cmd.OutOrStdout(), output); err != nil {
					return fmt.Errorf("failed to write command output: %w", err)
				}
				return nil
			}

			worktrees, err := git.ParseWorktreeListPorcelain(output)
			if err != nil {
				return fmt.Errorf("failed to parse git worktree output: %w", err)
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to resolve current working directory: %w", err)
			}

			rows := buildListRows(worktrees, cwd)
			switch listFormat {
			case config.ListFormatTable:
				if err := renderListTable(cmd.OutOrStdout(), rows); err != nil {
					return err
				}
			case config.ListFormatJSON:
				if err := renderListJSON(cmd.OutOrStdout(), rows); err != nil {
					return fmt.Errorf("failed to write list JSON: %w", err)
				}
			default:
				return fmt.Errorf("unsupported list format %q. Use one of: %s", listFormat, config.ListSupportedFormatsText)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&outputFormat, "format", config.ListDefaultFormat, "output format: "+config.ListSupportedFormatsText)
	return cmd
}

func parseListFormat(raw string) (string, error) {
	format := strings.ToLower(strings.TrimSpace(raw))
	switch format {
	case config.ListFormatTable, config.ListFormatRaw, config.ListFormatJSON:
		return format, nil
	default:
		return "", fmt.Errorf("invalid --format value %q. Use one of: %s", raw, config.ListSupportedFormatsText)
	}
}

func renderListJSON(w io.Writer, rows []listRow) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(rows)
}
