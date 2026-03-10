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
	"github.com/vansdevcode/worktree-manager/internal/state"
	"github.com/vansdevcode/worktree-manager/pkg/ui"
)

var rmCmd = &cobra.Command{
	Use:   "rm <directory>",
	Short: "Remove a worktree",
	Long: `Remove a worktree and optionally delete its branch.

The branch name is determined from the worktree itself, not the directory name.
If the worktree is in detached HEAD state, use --force to remove anyway.`,
	Args: cobra.ExactArgs(1),
	RunE: runRm,
}

var (
	rmForce        bool
	rmDeleteBranch bool
	rmNoHooks      bool
)

func init() {
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "Force removal even with uncommitted changes")
	rmCmd.Flags().BoolVarP(&rmDeleteBranch, "delete-branch", "d", false, "Also delete the branch")
	rmCmd.Flags().BoolVar(&rmNoHooks, "no-hooks", false, "Skip running hooks")
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
		if err != nil {
			return fmt.Errorf("failed to check for uncommitted changes: %w", err)
		}
		if hasChanges {
			return fmt.Errorf("worktree has uncommitted changes, use --force to remove anyway")
		}

		hasUntracked, err := git.HasUntrackedFiles(worktreePath)
		if err != nil {
			return fmt.Errorf("failed to check for untracked files: %w", err)
		}
		if hasUntracked {
			ui.Warning("⚠ Worktree has untracked files")
		}
	}

	// Get actual branch name from worktree before removal (for hooks and branch deletion)
	branchName, err := git.GetWorktreeBranch(worktreePath)
	if err != nil {
		if !rmForce {
			return fmt.Errorf("failed to determine branch name: %w (use --force to remove anyway)", err)
		}
		// With --force, proceed without hooks/branch deletion
		ui.Warning("⚠ Could not determine branch name: %v", err)
		ui.Warning("⚠ Skipping hooks and branch deletion")
		branchName = "" // Clear branch name to skip hooks
	}

	// Load state for hook vars
	var vars map[string]string
	if s, err := state.Load(rootDir, worktreePath); err == nil && s.Vars != nil {
		vars = s.Vars
	}

	// Run pre-delete hook
	if !rmNoHooks && branchName != "" {
		if err := hook.RunHookByName(rootDir, "pre-delete", branchName, worktreePath, vars); err != nil {
			return fmt.Errorf("pre-delete hook failed: %w", err)
		}
	}

	// Run post-delete hook before removal (while worktree still exists)
	if !rmNoHooks && branchName != "" {
		if err := hook.RunHookByName(rootDir, "post-delete", branchName, worktreePath, vars); err != nil {
			return fmt.Errorf("post-delete hook failed: %w", err)
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
	if rmDeleteBranch && branchName != "" {
		ui.Info("Deleting branch '%s'...", branchName)
		if err := git.DeleteBranch(bareDir, branchName); err != nil {
			ui.Warning("Failed to delete branch: %v", err)
		} else {
			ui.Success("✓ Branch deleted")
		}
	} else if rmDeleteBranch && branchName == "" {
		ui.Warning("⚠ Cannot delete branch: branch name could not be determined")
	}

	// Clean up state file
	if err := state.Remove(rootDir, worktreePath); err != nil {
		ui.Warning("Failed to remove state file: %v", err)
	}

	ui.Success("✓ Worktree removed successfully")
	return nil
}
