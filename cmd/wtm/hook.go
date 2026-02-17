package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/vansdevcode/worktree-manager/internal/hook"
)

var hookCmd = &cobra.Command{
	Use:   "hook <hook-name>",
	Short: "Process and execute a templated hook script",
	Long: `Process a hook script as a Go template with gomplate functions, then execute it.

Looks up the hook by name in .worktree/hooks/ directory.

The script has access to template variables:
  - .Branch: The branch name (e.g., "feature/user-auth")
  - .Directory: Absolute path to worktree directory
  - .RootDirectory: Absolute path to repository root

And all gomplate functions (https://docs.gomplate.ca/functions/):
  - strings.Slug: Convert to URL-friendly slug
  - strings.ReplaceAll: String replacement
  - And many more...

This command must be run from within a branch directory (direct child of the root directory).
It will automatically infer the worktree context from the current working directory.

Example:
  cd /path/to/root/my-branch && wtm hook post-create`,
	Args: cobra.ExactArgs(1),
	RunE: runHook,
}

func init() {
	rootCmd.AddCommand(hookCmd)
}

func runHook(cmd *cobra.Command, args []string) error {
	hookName := args[0]

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Infer context from current working directory
	ctx, err := inferWorktreeContext(cwd)
	if err != nil {
		return err
	}

	return hook.RunHookByName(ctx.RootDirectory, hookName, ctx.Branch, ctx.Directory)
}

// inferWorktreeContext determines the worktree context from the current working directory
func inferWorktreeContext(cwd string) (struct {
	Branch        string
	Directory     string
	RootDirectory string
}, error) {
	result := struct {
		Branch        string
		Directory     string
		RootDirectory string
	}{}

	// Find the root directory by looking for .worktree directory
	absPath, err := filepath.Abs(cwd)
	if err != nil {
		return result, fmt.Errorf("failed to resolve current directory: %w", err)
	}

	currentDir := absPath
	var rootDirectory string
	for {
		worktreeDir := filepath.Join(currentDir, ".worktree")
		if info, err := os.Stat(worktreeDir); err == nil && info.IsDir() {
			rootDirectory = currentDir
			break
		}
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			// Reached filesystem root without finding .worktree
			return result, fmt.Errorf("not in a worktree directory: .worktree directory not found")
		}
		currentDir = parent
	}

	result.RootDirectory = rootDirectory

	// Check if cwd is directly in a branch directory
	// Branch directories are children of root directory
	if filepath.Dir(absPath) != rootDirectory {
		return result, fmt.Errorf("must be run from a branch directory (direct child of root), not a subdirectory")
	}

	// The branch directory is the cwd itself
	result.Directory = absPath
	result.Branch = filepath.Base(absPath)

	return result, nil
}
