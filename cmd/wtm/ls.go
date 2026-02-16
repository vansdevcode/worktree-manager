package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vansdevcode/worktree-manager/internal/config"
	"github.com/vansdevcode/worktree-manager/internal/git"
	"github.com/vansdevcode/worktree-manager/pkg/ui"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all worktrees",
	Long:  `List all worktrees in the repository.`,
	Args:  cobra.NoArgs,
	RunE:  runLs,
}

func runLs(cmd *cobra.Command, args []string) error {
	// Find root directory
	rootDir, err := config.FindRoot()
	if err != nil {
		return fmt.Errorf("not in a worktree-managed repository (no .bare directory found)")
	}

	bareDir := config.GetBareDir(rootDir)

	// List worktrees
	output, err := git.ListWorktrees(bareDir)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	ui.Plain(output)
	return nil
}
