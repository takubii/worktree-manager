package cli

import (
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/takubii/git-worktree-opener/internal/buildinfo"
	"github.com/takubii/git-worktree-opener/internal/config"
	"github.com/takubii/git-worktree-opener/internal/doctor"
	"github.com/takubii/git-worktree-opener/internal/git"
	"github.com/takubii/git-worktree-opener/internal/opener"
	"github.com/takubii/git-worktree-opener/internal/selector"
	"github.com/takubii/git-worktree-opener/internal/updater"
)

// Dependencies holds external dependencies for command execution.
type Dependencies struct {
	Stdout   io.Writer
	Stderr   io.Writer
	Version  string
	LookPath func(file string) (string, error)
	Git      git.Client
	Opener   opener.Opener
	Enter    EnterRunner
	After    AfterRunner
	Selector selector.Selector
	Config   config.Provider
	Updater  updater.Service
	Doctor   doctor.Service
}

// NewRootCmd creates the root command for wto.
func NewRootCmd(deps Dependencies) *cobra.Command {
	deps = withDefaults(deps)
	var showVersion bool
	var verbose bool

	cmd := &cobra.Command{
		Use:           "wto",
		Short:         "Git worktree helper CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			cmd.SetContext(withVerbose(cmd.Context(), cmd.ErrOrStderr(), verbose))
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if showVersion {
				_, err := cmd.OutOrStdout().Write([]byte(deps.Version + "\n"))
				return err
			}
			return cmd.Help()
		},
	}
	cmd.SetOut(deps.Stdout)
	cmd.SetErr(deps.Stderr)
	cmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "print verbose trace logs to stderr")
	cmd.Flags().BoolVar(&showVersion, "version", false, "print wto version")

	cmd.AddCommand(newListCmd(deps))
	cmd.AddCommand(newOpenCmd(deps))
	cmd.AddCommand(newNewCmd(deps))
	cmd.AddCommand(newRmCmd(deps))
	cmd.AddCommand(newConfigCmd(deps))
	cmd.AddCommand(newEnterCmd(deps))
	cmd.AddCommand(newUpdateCmd(deps))
	cmd.AddCommand(newVersionCmd(deps))
	cmd.AddCommand(newDoctorCmd(deps))

	return cmd
}

func withDefaults(deps Dependencies) Dependencies {
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}
	if deps.Version == "" {
		deps.Version = buildinfo.Version
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
	if deps.Enter == nil {
		deps.Enter = newDefaultEnterRunner()
	}
	if deps.After == nil {
		deps.After = newDefaultAfterRunner()
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
	if deps.Doctor == nil {
		deps.Doctor = doctor.NewService(doctor.Options{
			LookPath: deps.LookPath,
			Git:      deps.Git,
		})
	}
	return deps
}
