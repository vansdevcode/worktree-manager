package state

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	rootDir := t.TempDir()
	worktreeDir := filepath.Join(rootDir, "feat-ticket-123")

	// Create .git directory (mimics real setup — state lives in .git/wtm-state/)
	if err := os.MkdirAll(filepath.Join(rootDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	s := &State{
		Vars: map[string]string{
			"ticket": "TICKET-123",
			"env":    "staging",
		},
	}

	if err := Save(rootDir, worktreeDir, s); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file was created at expected path
	expectedPath := filepath.Join(rootDir, ".git", "wtm-state", "feat-ticket-123.json")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("state file not created at %s: %v", expectedPath, err)
	}

	loaded, err := Load(rootDir, worktreeDir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if !reflect.DeepEqual(loaded.Vars, s.Vars) {
		t.Errorf("Load() Vars = %v, want %v", loaded.Vars, s.Vars)
	}
}

func TestLoad_NonExistent(t *testing.T) {
	rootDir := t.TempDir()
	worktreeDir := filepath.Join(rootDir, "nonexistent")

	s, err := Load(rootDir, worktreeDir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if s.Vars != nil {
		t.Errorf("Load() Vars = %v, want nil", s.Vars)
	}
}

func TestSave_EmptyVars(t *testing.T) {
	rootDir := t.TempDir()
	worktreeDir := filepath.Join(rootDir, "my-branch")

	s := &State{}
	if err := Save(rootDir, worktreeDir, s); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(rootDir, worktreeDir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.Vars != nil {
		t.Errorf("Load() Vars = %v, want nil", loaded.Vars)
	}
}

func TestRemove(t *testing.T) {
	rootDir := t.TempDir()
	worktreeDir := filepath.Join(rootDir, "my-branch")

	s := &State{Vars: map[string]string{"key": "val"}}
	if err := Save(rootDir, worktreeDir, s); err != nil {
		t.Fatal(err)
	}

	if err := Remove(rootDir, worktreeDir); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}

	// Should return empty state after removal
	loaded, err := Load(rootDir, worktreeDir)
	if err != nil {
		t.Fatalf("Load() after Remove() error: %v", err)
	}
	if loaded.Vars != nil {
		t.Errorf("Load() after Remove() Vars = %v, want nil", loaded.Vars)
	}
}

func TestRemove_NonExistent(t *testing.T) {
	rootDir := t.TempDir()
	worktreeDir := filepath.Join(rootDir, "nonexistent")

	if err := Remove(rootDir, worktreeDir); err != nil {
		t.Fatalf("Remove() error for non-existent: %v", err)
	}
}
