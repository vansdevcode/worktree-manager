package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vansdevcode/worktree-manager/internal/config"
	"github.com/vansdevcode/worktree-manager/internal/git"
	"github.com/vansdevcode/worktree-manager/internal/hook"
	"github.com/vansdevcode/worktree-manager/pkg/ui"
)

var rmCmd = &cobra.Command{
	Use:   "rm <directory>",
	Short: "Remove a worktree",
	Long:  `Remove a worktree and optionally delete its branch.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRm,
}

var (
	rmForce        bool
	rmDeleteBranch bool
	rmNoHooks      bool
)

func init() {
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "Force removal even with uncommitted changes")
	rmCmd.Flags().BoolVarP(&rmDeleteBranch, "delete-branch", "d", false, "Also delete the branch")
	rmCmd.Flags().BoolVar(&rmNoHooks, "no-hooks", false, "Skip running post-delete hooks")
}

func runRm(cmd *cobra.Command, args []string) error {
	directory := args[0]

	// Find root directory
	rootDir, err := config.FindRoot()
	if err != nil {
		return fmt.Errorf("not in a worktree-managed repository (no .bare directory found)")
	}

	bareDir := config.GetBareDir(rootDir)

	// Resolve directory path
	var worktreePath string
	if filepath.IsAbs(directory) {
		worktreePath = directory
	} else {
		worktreePath = filepath.Join(rootDir, directory)
	}

	// Check if directory exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return fmt.Errorf("directory '%s' does not exist", directory)
	}

	// Get current directory and check if we're inside the worktree to be removed
	currentDir, err := os.Getwd()
	if err == nil {
		// Normalize paths for comparison
		currentDirAbs, err := filepath.Abs(currentDir)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
		worktreePathAbs, err := filepath.Abs(worktreePath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}

		if currentDirAbs == worktreePathAbs || strings.HasPrefix(currentDirAbs, worktreePathAbs+string(filepath.Separator)) {
			return fmt.Errorf("cannot remove worktree you're currently in")
		}
	}

	// Safety checks
	if !rmForce {
		hasChanges, err := git.HasUncommittedChanges(worktreePath)
		if err == nil && hasChanges {
			return fmt.Errorf("worktree has uncommitted changes, use --force to remove anyway")
		}

		hasUntracked, err := git.HasUntrackedFiles(worktreePath)
		if err == nil && hasUntracked {
			ui.Warning("⚠ Worktree has untracked files")
		}
	}

	// Get branch name before removal (for hook)
	branchName := filepath.Base(worktreePath) // Simplified, could be improved

	// Run post-delete hook before removal
	if !rmNoHooks {
		hookPath := config.GetHookPath(rootDir, "post-delete")
		if _, err := os.Stat(hookPath); err == nil {
			ui.Info("Running post-delete hook...")
			hookData := hook.HookData{
				RootDirectory: rootDir,
				Directory:     worktreePath,
				Branch:        branchName,
			}
			if err := hook.RunHook(hookPath, hookData); err != nil {
				ui.Warning("Post-delete hook failed: %v", err)
			}
		}
	}

	// Remove worktree
	ui.Info("Removing worktree...")
	if rmForce {
		if err := git.RemoveWorktreeForce(bareDir, worktreePath); err != nil {
			return fmt.Errorf("failed to remove worktree: %w", err)
		}
	} else {
		if err := git.RemoveWorktree(bareDir, worktreePath); err != nil {
			return fmt.Errorf("failed to remove worktree: %w", err)
		}
	}

	// Delete branch if requested
	if rmDeleteBranch {
		ui.Info("Deleting branch '%s'...", branchName)
		if err := git.DeleteBranch(bareDir, branchName); err != nil {
			ui.Warning("Failed to delete branch: %v", err)
		} else {
			ui.Success("✓ Branch deleted")
		}
	}

	ui.Success("✓ Worktree removed successfully")
	return nil
}
