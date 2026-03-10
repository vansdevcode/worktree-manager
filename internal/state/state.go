package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// State holds persisted per-worktree metadata.
type State struct {
	Vars map[string]string `json:"vars,omitempty"`
}

// statePath returns the state file path for a worktree directory.
// State is stored at <rootDir>/.worktree/state/<directoryName>.json
func statePath(rootDir, worktreeDir string) string {
	dirName := filepath.Base(worktreeDir)
	return filepath.Join(rootDir, ".worktree", "state", dirName+".json")
}

// Load reads the state file for a worktree. Returns an empty State if the file doesn't exist.
func Load(rootDir, worktreeDir string) (*State, error) {
	data, err := os.ReadFile(statePath(rootDir, worktreeDir))
	if err != nil {
		if os.IsNotExist(err) {
			return &State{}, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}
	return &s, nil
}

// Save writes the state file for a worktree. Creates the directory if needed.
func Save(rootDir, worktreeDir string, s *State) error {
	p := statePath(rootDir, worktreeDir)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(p, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}
	return nil
}

// Remove deletes the state file for a worktree.
func Remove(rootDir, worktreeDir string) error {
	err := os.Remove(statePath(rootDir, worktreeDir))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove state file: %w", err)
	}
	return nil
}
