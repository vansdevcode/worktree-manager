package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vansdevcode/worktree-manager/internal/config"
	"github.com/vansdevcode/worktree-manager/internal/git"
	"github.com/vansdevcode/worktree-manager/internal/hook"
	"github.com/vansdevcode/worktree-manager/internal/pr"
	"github.com/vansdevcode/worktree-manager/internal/template"
	"github.com/vansdevcode/worktree-manager/internal/worktree"
	"github.com/vansdevcode/worktree-manager/pkg/ui"
)

var addCmd = &cobra.Command{
	Use:   "add <base-branch> [new-branch] [directory]",
	Short: "Add a new worktree",
	Long: `Add a new worktree for an existing or new branch.

If the branch doesn't exist, it will be created from the base branch.
Supports PR syntax: pr/<number> or pr/<number>/<custom-name>

Examples:
  gh wt add main feature-x          # Create feature-x from main
  gh wt add main                    # Create worktree for main branch
  gh wt add main feature-y my-dir   # Create in custom directory
  gh wt add pr/123                  # Checkout PR #123
  gh wt add pr/123 custom-name      # PR #123 in custom directory`,
	Args: cobra.RangeArgs(1, 3),
	RunE: runAdd,
}

var addNoHooks bool

func init() {
	addCmd.Flags().BoolVar(&addNoHooks, "no-hooks", false, "Skip running post-create hooks")
}

func runAdd(cmd *cobra.Command, args []string) error {
	// Find root directory
	rootDir, err := config.FindRoot()
	if err != nil {
		return fmt.Errorf("not in a worktree-managed repository (no .bare directory found)")
	}

	bareDir := config.GetBareDir(rootDir)
	baseBranch := args[0]
	newBranch := ""
	directory := ""

	// Parse arguments
	if len(args) > 1 {
		newBranch = args[1]
	}
	if len(args) > 2 {
		directory = args[2]
	}

	// Check if base branch is PR syntax
	isPR := false
	prNumber := 0
	if strings.HasPrefix(baseBranch, "pr/") {
		isPR = true
		re := regexp.MustCompile(`^pr/(\d+)(?:/(.+))?$`)
		matches := re.FindStringSubmatch(baseBranch)
		if matches == nil {
			return fmt.Errorf("invalid PR syntax, use pr/<number> or pr/<number>/<name>")
		}

		prNumber, _ = strconv.Atoi(matches[1])
		if matches[2] != "" {
			directory = matches[2]
		}
		if directory == "" && newBranch != "" {
			directory = newBranch
			newBranch = ""
		}
		if directory == "" {
			directory = fmt.Sprintf("pr-%d", prNumber)
		}
	}

	// If no new branch specified, use base branch
	if newBranch == "" && !isPR {
		newBranch = baseBranch
	}

	// Determine directory name
	if directory == "" {
		if isPR {
			directory = fmt.Sprintf("pr-%d", prNumber)
		} else {
			directory = worktree.GenerateWorktreeDirectory(newBranch)
		}
	}

	worktreePath := filepath.Join(rootDir, directory)

	// Check if directory already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("directory '%s' already exists", directory)
	}

	// Handle PR checkout
	if isPR {
		ui.Info("Fetching PR #%d...", prNumber)

		branchName, err := pr.FetchPR(bareDir, prNumber, directory)
		if err != nil {
			return fmt.Errorf("failed to fetch PR: %w", err)
		}

		ui.Info("Creating worktree for PR #%d (branch: %s)", prNumber, branchName)
		if err := git.AddWorktree(bareDir, branchName, worktreePath, ""); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}

		newBranch = branchName
	} else {
		// Check if branch exists
		branchExists, err := git.BranchExists(bareDir, newBranch)
		if err != nil {
			return fmt.Errorf("failed to check if branch exists: %w", err)
		}

		if branchExists {
			ui.Info("Creating worktree for existing branch: %s", newBranch)
			if err := git.AddWorktree(bareDir, newBranch, worktreePath, ""); err != nil {
				return fmt.Errorf("failed to create worktree: %w", err)
			}
		} else {
			ui.Info("Creating new branch '%s' from '%s'", newBranch, baseBranch)
			if err := git.AddWorktree(bareDir, newBranch, worktreePath, baseBranch); err != nil {
				return fmt.Errorf("failed to create worktree: %w", err)
			}
		}
	}

	// Process files
	filesDir := config.GetFilesDir(rootDir)
	if _, err := os.Stat(filesDir); err == nil {
		ui.Info("Processing files...")
		data := template.TemplateData{
			Branch:        newBranch,
			Directory:     worktreePath,
			RootDirectory: rootDir,
		}
		if err := template.ProcessTemplates(filesDir, worktreePath, data); err != nil {
			ui.Warning("Failed to process files: %v", err)
		}
	}

	// Run post-create hook
	if !addNoHooks {
		hookPath := config.GetHookPath(rootDir, "post-create")
		if _, err := os.Stat(hookPath); err == nil {
			ui.Info("Running post-create hook...")
			hookData := hook.HookData{
				RootDirectory: rootDir,
				Directory:     worktreePath,
				Branch:        newBranch,
			}
			if err := hook.RunHook(hookPath, hookData); err != nil {
				ui.Warning("Post-create hook failed: %v", err)
			}
		}
	}

	ui.Success("✓ Worktree created successfully")
	ui.Info("  Branch: %s", newBranch)
	ui.Info("  Directory: %s", worktreePath)

	return nil
}
