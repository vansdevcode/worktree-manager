package config

import (
	"os"
	"path/filepath"
)

// Config holds the configuration for the worktree manager
type Config struct {
	RootDir string
	BareDir string
	NoHooks bool
}

// FindRoot walks up the directory tree to find a directory containing .wtm.toml.
func FindRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		configPath := filepath.Join(currentDir, ".wtm.toml")
		if _, err := os.Stat(configPath); err == nil {
			return currentDir, nil
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
	return filepath.Join(rootDir, ".git")
}

// GetStateDir returns the path to the wtm state directory inside the bare repo
func GetStateDir(rootDir string) string {
	return filepath.Join(rootDir, ".git", "wtm-state")
}
