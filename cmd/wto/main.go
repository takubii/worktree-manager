package main

import (
	"fmt"
	"os"

	"github.com/takubii/git-worktree-opener/internal/cli"
)

func main() {
	rootCmd := cli.NewRootCmd(cli.Dependencies{})
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
