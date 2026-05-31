package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takubii/worktree-manager/internal/config"
	"github.com/takubii/worktree-manager/internal/git"
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
			tracef(cmd.Context(), "list: format=%s", listFormat)

			tracef(cmd.Context(), "list: running `git worktree list --porcelain`")
			output, err := deps.Git.WorktreeListPorcelain(cmd.Context())
			if err != nil {
				return err
			}
			tracef(cmd.Context(), "list: received git worktree output")

			if listFormat == config.ListFormatRaw {
				if _, err := io.WriteString(cmd.OutOrStdout(), output); err != nil {
					return fmt.Errorf("failed to write command output: %w", err)
				}
				tracef(cmd.Context(), "list: wrote raw output")
				return nil
			}

			worktrees, err := git.ParseWorktreeListPorcelain(output)
			if err != nil {
				return fmt.Errorf("failed to parse git worktree output: %w", err)
			}
			tracef(cmd.Context(), "list: parsed %d worktrees", len(worktrees))
			unavailable := classifyUnavailableWorktrees(worktrees)
			warnUnavailableWorktreesForList(cmd.ErrOrStderr(), unavailable)

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to resolve current working directory: %w", err)
			}

			rows := buildListRows(worktrees, cwd)
			tracef(cmd.Context(), "list: rendering %d rows as %s", len(rows), listFormat)
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
