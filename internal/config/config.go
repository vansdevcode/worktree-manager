package config

import (
	"os"
	"path/filepath"
)

// Config holds the configuration for the worktree manager
type Config struct {
	RootDir     string
	BareDir     string
	WorktreeDir string
	NoHooks     bool
}

// FindRoot walks up the directory tree to find the .bare directory
func FindRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		bareDir := filepath.Join(currentDir, ".bare")

		if fi, err := os.Lstat(bareDir); err == nil {
            if fi.Mode()&os.ModeSymlink == 0 && fi.IsDir() {
                return currentDir, nil
            }
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break
		}
		currentDir = parent
	}

	return "", os.ErrNotExist
}

// GetBareDir returns the path to the bare repository
func GetBareDir(rootDir string) string {
	return filepath.Join(rootDir, ".bare")
}

// GetWorktreeDir returns the path to the worktree metadata directory
func GetWorktreeDir(rootDir string) string {
	return filepath.Join(rootDir, ".worktree")
}

// GetFilesDir returns the path to the files directory
func GetFilesDir(rootDir string) string {
	return filepath.Join(rootDir, ".worktree", "files")
}

// GetHookPath returns the path to a hook script
func GetHookPath(rootDir, hookName string) string {
	return filepath.Join(rootDir, ".worktree", "hooks", hookName)
}
