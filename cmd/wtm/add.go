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
	"github.com/vansdevcode/worktree-manager/internal/state"
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
  wtm add main feature-x              # Create feature-x from main
  wtm add main                        # Create worktree for main branch
  wtm add main feature-y my-dir       # Create in custom directory
  wtm add pr/123                      # Checkout PR #123
  wtm add pr/123 custom-name          # PR #123 in custom directory
  wtm add main feat/T-123 -v ticket=T-123  # With custom variable`,
	Args: cobra.RangeArgs(1, 3),
	RunE: runAdd,
}

var (
	addNoHooks bool
	addVars    []string
)

func init() {
	addCmd.Flags().BoolVar(&addNoHooks, "no-hooks", false, "Skip running hooks")
	addCmd.Flags().StringArrayVarP(&addVars, "var", "v", nil, "Set custom variable (key=value), can be repeated")
}

// normalizeRemoteBranch extracts the local branch name from a remote branch reference
// e.g., "origin/develop" -> ("develop", "origin/develop")
// Returns (localName, startPoint) where startPoint is the full reference if it's a remote branch
func normalizeRemoteBranch(branchRef string) (string, string) {
	// Check for origin/ prefix
	if strings.HasPrefix(branchRef, "origin/") {
		localName := strings.TrimPrefix(branchRef, "origin/")
		return localName, branchRef
	}

	// Check for other remote prefixes (e.g., upstream/main)
	// For now, only handle origin/ explicitly
	// Other cases fall through to default behavior

	// Not a remote branch reference, return as-is
	return branchRef, ""
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
	// Handle remote branch references (e.g., origin/develop)
	startPoint := ""
	if newBranch == "" && !isPR {
		localName, remoteRef := normalizeRemoteBranch(baseBranch)
		newBranch = localName
		startPoint = remoteRef
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

	// Parse custom variables
	vars, err := parseVars(addVars)
	if err != nil {
		return err
	}

	// Run pre-create hook
	if !addNoHooks {
		if err := hook.RunHookByName(rootDir, "pre-create", newBranch, worktreePath, vars); err != nil {
			return fmt.Errorf("pre-create hook failed: %w", err)
		}
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
		// Check if branch exists locally (not just remote)
		localBranchExists, err := git.LocalBranchExists(bareDir, newBranch)
		if err != nil {
			return fmt.Errorf("failed to check if branch exists: %w", err)
		}

		if localBranchExists {
			// Local branch exists - check out directly
			ui.Info("Creating worktree for existing branch: %s", newBranch)
			if err := git.AddWorktree(bareDir, newBranch, worktreePath, ""); err != nil {
				return fmt.Errorf("failed to create worktree: %w", err)
			}
		} else {
			// Local branch doesn't exist - create it from startPoint or baseBranch
			// Use startPoint if it was a remote branch reference, otherwise use baseBranch
			createFrom := baseBranch
			if startPoint != "" {
				createFrom = startPoint
				ui.Info("Creating new local branch '%s' from remote '%s'", newBranch, startPoint)
			} else {
				ui.Info("Creating new branch '%s' from '%s'", newBranch, baseBranch)
			}
			if err := git.AddWorktree(bareDir, newBranch, worktreePath, createFrom); err != nil {
				return fmt.Errorf("failed to create worktree: %w", err)
			}
		}
	}

	// Save state if there are custom variables
	if len(vars) > 0 {
		s := &state.State{Vars: vars}
		if err := state.Save(rootDir, worktreePath, s); err != nil {
			ui.Warning("Failed to save state: %v", err)
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
			Vars:          vars,
		}
		if err := template.ProcessTemplates(filesDir, worktreePath, data); err != nil {
			ui.Warning("Failed to process files: %v", err)
		}
	}

	// Run post-create hook
	if !addNoHooks {
		if err := hook.RunHookByName(rootDir, "post-create", newBranch, worktreePath, vars); err != nil {
			return fmt.Errorf("post-create hook failed: %w", err)
		}
	}

	ui.Success("✓ Worktree created successfully")
	ui.Info("  Branch: %s", newBranch)
	ui.Info("  Directory: %s", worktreePath)

	return nil
}

// parseVars parses custom variable flags in "key=value" format.
func parseVars(raw []string) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	vars := make(map[string]string, len(raw))
	for _, v := range raw {
		key, value, ok := strings.Cut(v, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid variable format %q, expected key=value", v)
		}
		vars[key] = value
	}
	return vars, nil
}
