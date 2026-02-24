package cli

import (
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/takubii/git-worktree-opener/internal/config"
	"github.com/takubii/git-worktree-opener/internal/git"
	"github.com/takubii/git-worktree-opener/internal/opener"
	"github.com/takubii/git-worktree-opener/internal/selector"
	"github.com/takubii/git-worktree-opener/internal/updater"
)

// Dependencies holds external dependencies for command execution.
type Dependencies struct {
	Stdout   io.Writer
	Stderr   io.Writer
	LookPath func(file string) (string, error)
	Git      git.Client
	Opener   opener.Opener
	Selector selector.Selector
	Config   config.Provider
	Updater  updater.Service
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
	cmd.AddCommand(newOpenCmd(deps))
	cmd.AddCommand(newNewCmd(deps))
	cmd.AddCommand(newRmCmd(deps))
	cmd.AddCommand(newConfigCmd(deps))
	cmd.AddCommand(newUpdateCmd(deps))

	return cmd
}

func withDefaults(deps Dependencies) Dependencies {
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}
	if deps.LookPath == nil {
		deps.LookPath = exec.LookPath
	}
	if deps.Git == nil {
		deps.Git = git.NewClient()
	}
	if deps.Opener == nil {
		deps.Opener = opener.NewDefault()
	}
	if deps.Selector == nil {
		deps.Selector = selector.NewDefault(os.Stdin, deps.Stderr)
	}
	if deps.Config == nil {
		deps.Config = config.NewStaticProvider(config.DefaultConfig())
	}
	if deps.Updater == nil {
		deps.Updater = updater.NewInstaller()
	}
	return deps
}
