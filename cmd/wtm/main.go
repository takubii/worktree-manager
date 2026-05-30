package main

import (
	"fmt"
	"os"

	"github.com/takubii/worktree-manager/internal/buildinfo"
	"github.com/takubii/worktree-manager/internal/cli"
	"github.com/takubii/worktree-manager/internal/config"
	"github.com/takubii/worktree-manager/internal/git"
)

func main() {
	gitClient := git.NewClient()
	rootCmd := cli.NewRootCmd(cli.Dependencies{
		Git:     gitClient,
		Version: buildinfo.Version,
		Config: config.NewFileProvider(config.FileProviderOptions{
			Git:    gitClient,
			Stderr: os.Stderr,
		}),
	})
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
