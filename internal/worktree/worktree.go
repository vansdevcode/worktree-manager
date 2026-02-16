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

// GeneratePRDirectoryName generates directory name for PR worktrees
// pr/123 -> pr-123
// pr/123/custom -> custom
func GeneratePRDirectoryName(prNumber, customName string) string {
	if customName != "" {
		return customName
	}
	return fmt.Sprintf("pr-%s", prNumber)
}
