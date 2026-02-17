package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHookCommand(t *testing.T) {
	tests := []struct {
		name          string
		scriptContent string
		branch        string
		expectError   bool
	}{
		{
			name: "Simple template variables",
			scriptContent: `#!/bin/bash
echo "{{ .Branch }}"
`,
			branch:      "feature-test",
			expectError: false,
		},
		{
			name: "Slug function",
			scriptContent: `#!/bin/bash
echo "{{ .Branch | strings.Slug }}"
`,
			branch:      "feature-user-auth",
			expectError: false,
		},
		{
			name: "No shebang in processed content",
			scriptContent: `echo "{{ .Branch }}"
`,
			branch:      "test",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory structure
			// Root dir with .worktree/hooks, and branch directory as a child
			tmpDir, err := os.MkdirTemp("", "hook-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()

			// Create .worktree/hooks directory
			hooksDir := filepath.Join(tmpDir, ".worktree", "hooks")
			if err := os.MkdirAll(hooksDir, 0755); err != nil {
				t.Fatalf("Failed to create .worktree/hooks dir: %v", err)
			}

			// Create branch directory (direct child of root)
			branchDir := filepath.Join(tmpDir, tt.branch)
			if err := os.MkdirAll(branchDir, 0755); err != nil {
				t.Fatalf("Failed to create branch dir: %v", err)
			}

			// Write hook script in .worktree/hooks/
			hookPath := filepath.Join(hooksDir, "test-hook")
			if err := os.WriteFile(hookPath, []byte(tt.scriptContent), 0755); err != nil {
				t.Fatalf("Failed to write hook: %v", err)
			}

			// Change to branch directory before running hook
			originalWd, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get working directory: %v", err)
			}
			defer func() { _ = os.Chdir(originalWd) }()

			if err := os.Chdir(branchDir); err != nil {
				t.Fatalf("Failed to change to branch directory: %v", err)
			}

			// Run hook command with hook name (not path)
			err = runHook(nil, []string{"test-hook"})

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
