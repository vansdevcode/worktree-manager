package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/vansdevcode/worktree-manager/pkg/ui"
)

var rootCmd = &cobra.Command{
	Use:   getBinaryName(),
	Short: "Git worktree manager with templates and safety features",
	Long: `Worktree Manager simplifies Git worktree management with bare repositories, 
templates, hooks, and safety checks.

Can be used as a standalone CLI (wtm) or as a GitHub CLI extension (gh wtm).`,
	SilenceErrors: true,
	SilenceUsage:  true,
}

// getBinaryName returns the name of the binary being executed
func getBinaryName() string {
	return filepath.Base(os.Args[0])
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(rmCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(prCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ui.Error("%v", err)
		os.Exit(1)
	}
}
