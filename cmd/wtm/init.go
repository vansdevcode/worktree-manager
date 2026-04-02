package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vansdevcode/worktree-manager/internal/git"
	"github.com/vansdevcode/worktree-manager/internal/template"
	"github.com/vansdevcode/worktree-manager/internal/wtmconfig"
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
  wtm init myorg/myrepo
  wtm init myorg/myrepo my-project
  wtm init https://github.com/myorg/myrepo.git
  wtm init myorg/myrepo --new
  wtm init myorg/myrepo --no-hooks`,
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

	bareDir := filepath.Join(directory, ".git")

	ui.Info("Initializing repository in %s", directory)

	// Clone or init bare repository
	if initNew {
		ui.Info("Creating new bare repository...")
		if err := git.InitBare(bareDir); err != nil {
			return fmt.Errorf("failed to initialize bare repository: %w", err)
		}

		// Create initial branch with empty commit
		defaultBranch := "main"
		ui.Info("Creating initial branch: %s", defaultBranch)
		if err := git.CreateInitialBranch(bareDir, defaultBranch); err != nil {
			return fmt.Errorf("failed to create initial branch: %w", err)
		}
	} else {
		// Convert GitHub format if needed
		repoURL := git.ConvertGitHubFormat(repo)
		ui.Info("Cloning repository: %s", repoURL)

		if err := git.CloneBare(repoURL, bareDir); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}

		// Enable fetch for the bare repository
		if err := git.ConfigSet(bareDir, "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*"); err != nil {
			return fmt.Errorf("failed to configure remote.origin.fetch: %w", err)
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

	// Create .wtm.toml scaffold if it doesn't already exist in the repo
	wtmTomlPath := filepath.Join(worktreePath, ".wtm.toml")
	if _, err := os.Stat(wtmTomlPath); os.IsNotExist(err) {
		scaffold := `[hooks]
# post-create = "npm install"
# pre-delete = "echo 'cleaning up'"

[files]
# ".env" = { template = ".wtm/env.tmpl" }
# "config/local.yml" = { copy = ".wtm/local.yml" }

[vars]
# db_host = "localhost"
# db_port = "5432"
`
		if err := os.WriteFile(wtmTomlPath, []byte(scaffold), 0644); err != nil {
			ui.Warning("Failed to create .wtm.toml: %v", err)
		}
	}

	// Add .wtm.local.toml to .git/info/exclude
	excludePath := filepath.Join(bareDir, "info", "exclude")
	if err := addToExclude(excludePath, ".wtm.local.toml"); err != nil {
		ui.Warning("Failed to add .wtm.local.toml to .git/info/exclude: %v", err)
	}

	// Load config and process files/hooks
	cfg, err := wtmconfig.Load(worktreePath, directory)
	if err != nil {
		ui.Warning("Failed to load config: %v", err)
		cfg = &wtmconfig.Config{}
	}

	// Process files from config
	if len(cfg.Files) > 0 {
		ui.Info("Processing files...")
		files := make(map[string]template.FileSpec, len(cfg.Files))
		for k, v := range cfg.Files {
			files[k] = template.FileSpec{Template: v.Template, Copy: v.Copy}
		}
		data := template.TemplateData{
			Branch:        defaultBranch,
			Directory:     worktreePath,
			RootDirectory: directory,
			Vars:          cfg.Vars,
		}
		if err := template.ProcessFiles(files, worktreePath, worktreePath, data); err != nil {
			ui.Warning("Failed to process files: %v", err)
		}
	}

	// Run post-create hook
	if !initNoHooks {
		if err := runConfigHook(cfg, "post-create", defaultBranch, worktreePath, directory, cfg.Vars); err != nil {
			return fmt.Errorf("post-create hook failed: %w", err)
		}
	}

	ui.Success("✓ Repository initialized successfully")
	ui.Info("  Root directory: %s", directory)
	ui.Info("  Default branch worktree: %s", worktreePath)

	return nil
}

// addToExclude appends a pattern to .git/info/exclude if not already present.
func addToExclude(excludePath, pattern string) error {
	if err := os.MkdirAll(filepath.Dir(excludePath), 0755); err != nil {
		return err
	}

	content, err := os.ReadFile(excludePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Check if already present
	for _, line := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(line) == pattern {
			return nil
		}
	}

	// Append
	f, err := os.OpenFile(excludePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	_, err = fmt.Fprintf(f, "\n%s\n", pattern)
	return err
}
