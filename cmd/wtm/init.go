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
	"github.com/vansdevcode/worktree-manager/internal/template"
	"github.com/vansdevcode/worktree-manager/pkg/ui"
)

var initCmd = &cobra.Command{
	Use:   "init <repo> [directory]",
	Short: "Initialize a new worktree-managed repository",
	Long: `Initialize a new worktree-managed repository with bare repository structure.

The repository can be:
  - A GitHub repository (owner/repo)
  - A Git URL (https://github.com/owner/repo.git)
  - A local path (when using --new flag)

Examples:
  gh wt init myorg/myrepo
  gh wt init myorg/myrepo my-project
  gh wt init https://github.com/myorg/myrepo.git
  gh wt init myorg/myrepo --new
  gh wt init myorg/myrepo --no-hooks`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runInit,
}

var (
	initNew     bool
	initNoHooks bool
)

func init() {
	initCmd.Flags().BoolVar(&initNew, "new", false, "Create a new repository instead of cloning")
	initCmd.Flags().BoolVar(&initNoHooks, "no-hooks", false, "Skip running post-create hooks")
}

func runInit(cmd *cobra.Command, args []string) error {
	repo := args[0]
	directory := ""
	if len(args) > 1 {
		directory = args[1]
	}

	// If no directory specified, derive from repo
	if directory == "" {
		directory = filepath.Base(repo)
		directory = strings.TrimSuffix(directory, ".git")
	}

	// Check if directory already exists
	if _, err := os.Stat(directory); err == nil {
		return fmt.Errorf("directory '%s' already exists", directory)
	}

	// Create directory
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	bareDir := filepath.Join(directory, ".bare")
	worktreeDir := filepath.Join(directory, ".worktree")

	// Create .worktree directory structure
	if err := os.MkdirAll(filepath.Join(worktreeDir, "files"), 0755); err != nil {
		return fmt.Errorf("failed to create .worktree directory: %w", err)
	}

	ui.Info("Initializing repository in %s", directory)

	// Clone or init bare repository
	if initNew {
		ui.Info("Creating new bare repository...")
		if err := git.InitBare(bareDir); err != nil {
			return fmt.Errorf("failed to initialize bare repository: %w", err)
		}
	} else {
		// Convert GitHub format if needed
		repoURL := git.ConvertGitHubFormat(repo)
		ui.Info("Cloning repository: %s", repoURL)

		if err := git.CloneBare(repoURL, bareDir); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
	}

	// Get default branch
	defaultBranch, err := git.GetDefaultBranch(bareDir)
	if err != nil {
		return fmt.Errorf("failed to get default branch: %w", err)
	}

	// Create worktree for default branch
	worktreePath := filepath.Join(directory, defaultBranch)
	ui.Info("Creating worktree for default branch: %s", defaultBranch)

	if err := git.AddWorktree(bareDir, defaultBranch, worktreePath, ""); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Process files
	filesDir := filepath.Join(worktreeDir, "files")
	if _, err := os.Stat(filesDir); err == nil {
		ui.Info("Processing files...")
		data := template.TemplateData{
			Branch:        defaultBranch,
			Directory:     worktreePath,
			RootDirectory: directory,
		}
		if err := template.ProcessTemplates(filesDir, worktreePath, data); err != nil {
			ui.Warning("Failed to process files: %v", err)
		}
	}

	// Run post-create hook
	if !initNoHooks {
		hookPath := config.GetHookPath(directory, "post-create")
		if _, err := os.Stat(hookPath); err == nil {
			ui.Info("Running post-create hook...")
			hookData := hook.HookData{
				RootDirectory: directory,
				Directory:     worktreePath,
				Branch:        defaultBranch,
			}
			if err := hook.RunHook(hookPath, hookData); err != nil {
				ui.Warning("Post-create hook failed: %v", err)
			}
		}
	}

	ui.Success("✓ Repository initialized successfully")
	ui.Info("  Root directory: %s", directory)
	ui.Info("  Default branch worktree: %s", worktreePath)

	return nil
}
