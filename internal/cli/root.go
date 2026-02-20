package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/takubii/git-worktree-opener/internal/git"
)

// Dependencies holds external dependencies for command execution.
type Dependencies struct {
	Stdout io.Writer
	Stderr io.Writer
	Git    git.Client
}

// NewRootCmd creates the root command for wto.
func NewRootCmd(deps Dependencies) *cobra.Command {
	deps = withDefaults(deps)

	cmd := &cobra.Command{
		Use:           "wto",
		Short:         "Git worktree helper CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(deps.Stdout)
	cmd.SetErr(deps.Stderr)

	cmd.AddCommand(newListCmd(deps))

	return cmd
}

func withDefaults(deps Dependencies) Dependencies {
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}
	if deps.Git == nil {
		deps.Git = git.NewClient()
	}
	return deps
}
