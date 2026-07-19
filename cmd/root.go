package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "th",
	Short:   "Treehouse a Git Worktree manager to make your life easier",
	Aliases: []string{"th"},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
