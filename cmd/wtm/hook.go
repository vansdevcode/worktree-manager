package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/vansdevcode/worktree-manager/internal/state"
)

var hookCmd = &cobra.Command{
	Use:   "hook <hook-name>",
	Short: "Process and execute a templated hook script",
	Long: `Process a hook script as a Go template with gomplate functions, then execute it.

Looks up the hook by name in .wtm.toml config.

The script has access to template variables:
  - .Branch: The branch name (e.g., "feature/user-auth")
  - .Directory: Absolute path to worktree directory
  - .RootDirectory: Absolute path to repository root
  - .Vars: Custom variables from config and state

And all gomplate functions (https://docs.gomplate.ca/functions/):
  - strings.Slug: Convert to URL-friendly slug
  - strings.ReplaceAll: String replacement
  - And many more...

This command must be run from within a branch directory (direct child of the root directory).
It will automatically infer the worktree context from the current working directory.

Example:
  cd /path/to/root/my-branch && wtm hook post-create`,
	Args: cobra.ExactArgs(1),
	RunE: runHookCmd,
}

func init() {
	rootCmd.AddCommand(hookCmd)
}

func runHookCmd(cmd *cobra.Command, args []string) error {
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

	// Load config
	cfg, err := loadConfigFromRoot(ctx.RootDirectory)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Load custom variables from state and merge with config vars
	var vars map[string]string
	if s, err := state.Load(ctx.RootDirectory, ctx.Directory); err == nil && s.Vars != nil {
		vars = mergeVars(cfg.Vars, s.Vars)
	} else {
		vars = cfg.Vars
	}

	return runConfigHook(cfg, hookName, ctx.Branch, ctx.Directory, ctx.RootDirectory, vars)
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

	// Find the root directory by looking for .wtm.toml
	absPath, err := filepath.Abs(cwd)
	if err != nil {
		return result, fmt.Errorf("failed to resolve current directory: %w", err)
	}

	currentDir := absPath
	var rootDirectory string
	for {
		configPath := filepath.Join(currentDir, ".wtm.toml")
		if _, err := os.Stat(configPath); err == nil {
			rootDirectory = currentDir
			break
		}
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			return result, fmt.Errorf("not in a worktree directory: .wtm.toml not found")
		}
		currentDir = parent
	}

	result.RootDirectory = rootDirectory

	// Check if cwd is directly in a branch directory
	if filepath.Dir(absPath) != rootDirectory {
		return result, fmt.Errorf("must be run from a branch directory (direct child of root), not a subdirectory")
	}

	result.Directory = absPath
	result.Branch = filepath.Base(absPath)

	return result, nil
}
