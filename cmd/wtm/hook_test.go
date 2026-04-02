package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHookCommand(t *testing.T) {
	tests := []struct {
		name          string
		hookContent   string
		branch        string
		expectError   bool
	}{
		{
			name: "Simple template variables",
			hookContent: `#!/bin/bash
echo "{{ .Branch }}"
`,
			branch:      "feature-test",
			expectError: false,
		},
		{
			name: "Slug function",
			hookContent: `#!/bin/bash
echo "{{ .Branch | strings.Slug }}"
`,
			branch:      "feature-user-auth",
			expectError: false,
		},
		{
			name: "Inline without shebang",
			hookContent: `echo "{{ .Branch }}"
`,
			branch:      "test",
			expectError: false, // RunHookInline auto-prepends #!/bin/bash
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "hook-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()

			// Create .wtm.toml with the hook
			wtmToml := "[hooks]\ntest-hook = " + quoteToml(tt.hookContent) + "\n"
			if err := os.WriteFile(filepath.Join(tmpDir, ".wtm.toml"), []byte(wtmToml), 0644); err != nil {
				t.Fatalf("Failed to create .wtm.toml: %v", err)
			}

			// Create branch directory (direct child of root)
			branchDir := filepath.Join(tmpDir, tt.branch)
			if err := os.MkdirAll(branchDir, 0755); err != nil {
				t.Fatalf("Failed to create branch dir: %v", err)
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

			// Run hook command with hook name
			err = runHookCmd(nil, []string{"test-hook"})

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// quoteToml wraps a multiline string in TOML triple-quoted format.
func quoteToml(s string) string {
	return `"""` + "\n" + s + `"""`
}
