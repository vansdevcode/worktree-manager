package wtmconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds the merged wtm configuration from .wtm.toml and .wtm.local.toml.
type Config struct {
	Hooks map[string]string  `toml:"hooks"`
	Files map[string]FileSpec `toml:"files"`
	Vars  map[string]string  `toml:"vars"`
}

// FileSpec describes a file to be created in the worktree.
// Exactly one of Template or Copy should be set.
type FileSpec struct {
	Template string `toml:"template"`
	Copy     string `toml:"copy"`
}

// Load reads .wtm.toml from worktreeDir (the branch directory) and .wtm.local.toml
// from rootDir (next to .git/), then deep-merges them. Local values win.
// If neither file exists, returns an empty Config (not an error).
func Load(worktreeDir, rootDir string) (*Config, error) {
	base := &Config{}
	local := &Config{}

	// Read .wtm.toml from the worktree (branch) directory
	wtmPath := filepath.Join(worktreeDir, ".wtm.toml")
	if err := decodeFile(wtmPath, base); err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", wtmPath, err)
	}

	// Read .wtm.local.toml from the root directory
	localPath := filepath.Join(rootDir, ".wtm.local.toml")
	if err := decodeFile(localPath, local); err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", localPath, err)
	}

	return merge(base, local), nil
}

// decodeFile decodes a TOML file into cfg. If the file doesn't exist, cfg is left unchanged.
func decodeFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := toml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse error: %w", err)
	}
	return nil
}

// merge deep-merges local into base at the TOML table level. Local values win.
func merge(base, local *Config) *Config {
	result := &Config{
		Hooks: mergeMaps(base.Hooks, local.Hooks),
		Vars:  mergeMaps(base.Vars, local.Vars),
		Files: mergeFileMaps(base.Files, local.Files),
	}
	return result
}

func mergeMaps(base, local map[string]string) map[string]string {
	if len(base) == 0 && len(local) == 0 {
		return nil
	}
	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range local {
		result[k] = v
	}
	return result
}

func mergeFileMaps(base, local map[string]FileSpec) map[string]FileSpec {
	if len(base) == 0 && len(local) == 0 {
		return nil
	}
	result := make(map[string]FileSpec)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range local {
		result[k] = v
	}
	return result
}
