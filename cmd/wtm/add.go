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
	"github.com/vansdevcode/worktree-manager/internal/pr"
	"github.com/vansdevcode/worktree-manager/internal/state"
	"github.com/vansdevcode/worktree-manager/internal/template"
	"github.com/vansdevcode/worktree-manager/internal/worktree"
	"github.com/vansdevcode/worktree-manager/internal/wtmconfig"
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

	// Not a remote branch reference, return as-is
	return branchRef, ""
}

func runAdd(cmd *cobra.Command, args []string) error {
	// Find root directory
	rootDir, err := config.FindRoot()
	if err != nil {
		return fmt.Errorf("not in a worktree-managed repository (no .wtm.toml found)")
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

	// Load config from an existing worktree that has .wtm.toml.
	// We read from the root directory itself since .wtm.toml might be there,
	// or from the default branch worktree. Try rootDir first as that's where
	// FindRoot found .wtm.toml.
	cfg, err := loadConfigFromRoot(rootDir)
	if err != nil {
		ui.Warning("Failed to load config: %v", err)
		cfg = &wtmconfig.Config{}
	}

	// Merge config vars with command-line vars (command-line wins)
	mergedVars := mergeVars(cfg.Vars, vars)

	// Run pre-create hook
	if !addNoHooks {
		if err := runConfigHook(cfg, "pre-create", newBranch, worktreePath, rootDir, mergedVars); err != nil {
			return fmt.Errorf("pre-create hook failed: %w", err)
		}
	}

	// Fetch remote branch if needed
	if startPoint != "" {
		remoteBranch := strings.TrimPrefix(startPoint, "origin/")
		ui.Info("Fetching remote branch '%s'...", remoteBranch)
		refSpec := fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", remoteBranch, remoteBranch)
		if err := git.FetchRef(bareDir, refSpec); err != nil {
			return fmt.Errorf("failed to fetch remote branch '%s': %w", remoteBranch, err)
		}
		// Use full ref path to avoid ambiguous resolution in bare repos
		startPoint = "refs/remotes/origin/" + remoteBranch
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
			ui.Info("Creating worktree for existing branch: %s", newBranch)
			if err := git.AddWorktree(bareDir, newBranch, worktreePath, ""); err != nil {
				return fmt.Errorf("failed to create worktree: %w", err)
			}
		} else {
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

	// Save state if there are command-line variables
	if len(vars) > 0 {
		s := &state.State{Vars: vars}
		if err := state.Save(rootDir, worktreePath, s); err != nil {
			ui.Warning("Failed to save state: %v", err)
		}
	}

	// Process files from config
	if len(cfg.Files) > 0 {
		ui.Info("Processing files...")
		files := make(map[string]template.FileSpec, len(cfg.Files))
		for k, v := range cfg.Files {
			files[k] = template.FileSpec{Template: v.Template, Copy: v.Copy}
		}
		data := template.TemplateData{
			Branch:        newBranch,
			Directory:     worktreePath,
			RootDirectory: rootDir,
			Vars:          mergedVars,
		}
		// Source files are relative to the worktree that contains .wtm.toml
		sourceDir := findWtmTomlDir(rootDir)
		if err := template.ProcessFiles(files, sourceDir, worktreePath, data); err != nil {
			ui.Warning("Failed to process files: %v", err)
		}
	}

	// Run post-create hook
	if !addNoHooks {
		if err := runConfigHook(cfg, "post-create", newBranch, worktreePath, rootDir, mergedVars); err != nil {
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

// mergeVars merges config vars with command-line vars. Command-line vars win.
func mergeVars(configVars, cliVars map[string]string) map[string]string {
	if len(configVars) == 0 && len(cliVars) == 0 {
		return nil
	}
	result := make(map[string]string)
	for k, v := range configVars {
		result[k] = v
	}
	for k, v := range cliVars {
		result[k] = v
	}
	return result
}

// loadConfigFromRoot loads the wtm config by finding .wtm.toml in the root directory
// or any worktree within it.
func loadConfigFromRoot(rootDir string) (*wtmconfig.Config, error) {
	// First check if .wtm.toml is directly in rootDir (e.g., rootDir IS a worktree)
	if _, err := os.Stat(filepath.Join(rootDir, ".wtm.toml")); err == nil {
		return wtmconfig.Load(rootDir, rootDir)
	}

	// Otherwise, scan for worktrees that contain .wtm.toml
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return &wtmconfig.Config{}, nil
	}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		wtmPath := filepath.Join(rootDir, entry.Name(), ".wtm.toml")
		if _, err := os.Stat(wtmPath); err == nil {
			return wtmconfig.Load(filepath.Join(rootDir, entry.Name()), rootDir)
		}
	}

	return &wtmconfig.Config{}, nil
}

// findWtmTomlDir finds the directory containing .wtm.toml within the root.
func findWtmTomlDir(rootDir string) string {
	if _, err := os.Stat(filepath.Join(rootDir, ".wtm.toml")); err == nil {
		return rootDir
	}
	entries, _ := os.ReadDir(rootDir)
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		dir := filepath.Join(rootDir, entry.Name())
		if _, err := os.Stat(filepath.Join(dir, ".wtm.toml")); err == nil {
			return dir
		}
	}
	return rootDir
}
