package worktree

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// GenerateWorktreeDirectory converts a branch name to a filesystem-safe directory name
// while preserving the original case.
// Examples:
//
//	feature/user-auth -> feature-user-auth
//	Feature/User-Auth -> Feature-User-Auth
//	fix_bug_123 -> fix-bug-123
func GenerateWorktreeDirectory(branchName string) string {
	// Replace / and _ with -
	name := strings.ReplaceAll(branchName, "/", "-")
	name = strings.ReplaceAll(name, "_", "-")

	// Remove special characters, keep only alphanumeric, hyphens, and preserve case
	reg := regexp.MustCompile(`[^a-zA-Z0-9-]+`)
	name = reg.ReplaceAllString(name, "")

	// Remove consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	name = reg.ReplaceAllString(name, "-")

	// Trim hyphens from start/end
	name = strings.Trim(name, "-")

	return name
}

// GetWorktreePath returns the path for a worktree
func GetWorktreePath(rootDir, directoryName string) string {
	return filepath.Join(rootDir, directoryName)
}

// GetWorktreeConfigPath returns the path to .worktree directory
func GetWorktreeConfigPath(rootDir string) string {
	return filepath.Join(rootDir, ".worktree")
}

// GetFilesPath returns the path to files directory
func GetFilesPath(rootDir string) string {
	return filepath.Join(GetWorktreeConfigPath(rootDir), "files")
}

// GeneratePRDirectoryName generates directory name for PR worktrees
// pr/123 -> pr-123
// pr/123/custom -> custom
func GeneratePRDirectoryName(prNumber, customName string) string {
	if customName != "" {
		return customName
	}
	return fmt.Sprintf("pr-%s", prNumber)
}

// FindRoot walks up the directory tree to find the repository root
// The root is identified by the presence of a .bare directory
func FindRoot() (string, error) {
	currentDir := "."
	absPath, err := filepath.Abs(currentDir)
	if err != nil {
		return "", err
	}

	for {
		bareDir := filepath.Join(absPath, ".bare")
		if _, err := filepath.EvalSymlinks(bareDir); err == nil {
			return absPath, nil
		}

		parent := filepath.Dir(absPath)
		if parent == absPath {
			// Reached root of filesystem
			return "", fmt.Errorf("not in a git worktree repository (no .bare directory found)")
		}
		absPath = parent
	}
}

// GetFilesDir returns the path to the files directory
func GetFilesDir(root string) string {
	return GetFilesPath(root)
}

// GetHookPath returns the path to a specific hook
func GetHookPath(root, hookName string) string {
	return filepath.Join(GetWorktreeConfigPath(root), hookName)
}
