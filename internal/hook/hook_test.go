package hook

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractShebang(t *testing.T) {
	tests := []struct {
		name                string
		input               string
		expectedInterpreter string
		expectedRemaining   string
	}{
		{
			name:                "Standard bash shebang",
			input:               "#!/bin/bash\necho hello",
			expectedInterpreter: "/bin/bash",
			expectedRemaining:   "echo hello",
		},
		{
			name:                "Python shebang",
			input:               "#!/usr/bin/env python3\nprint('hello')",
			expectedInterpreter: "/usr/bin/env python3",
			expectedRemaining:   "print('hello')",
		},
		{
			name:                "No shebang",
			input:               "echo hello",
			expectedInterpreter: "",
			expectedRemaining:   "echo hello",
		},
		{
			name:                "Shebang without newline",
			input:               "#!/bin/bash",
			expectedInterpreter: "/bin/bash",
			expectedRemaining:   "",
		},
		{
			name:                "Empty content",
			input:               "",
			expectedInterpreter: "",
			expectedRemaining:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interpreter, remaining := ExtractShebang(tt.input)

			if interpreter != tt.expectedInterpreter {
				t.Errorf("ExtractShebang() interpreter = %q, want %q", interpreter, tt.expectedInterpreter)
			}

			if remaining != tt.expectedRemaining {
				t.Errorf("ExtractShebang() remaining = %q, want %q", remaining, tt.expectedRemaining)
			}
		})
	}
}

func TestRunHook(t *testing.T) {
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
			name: "No shebang in script",
			scriptContent: `echo "{{ .Branch }}"
`,
			branch:      "test",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for the hook script
			tmpDir, err := os.MkdirTemp("", "hook-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()

			// Create branch directory
			branchDir := filepath.Join(tmpDir, tt.branch)
			if err := os.MkdirAll(branchDir, 0755); err != nil {
				t.Fatalf("Failed to create branch dir: %v", err)
			}

			// Write hook script
			hookPath := filepath.Join(tmpDir, "test-hook")
			if err := os.WriteFile(hookPath, []byte(tt.scriptContent), 0755); err != nil {
				t.Fatalf("Failed to write hook: %v", err)
			}

			err = RunHook(hookPath, tt.branch, branchDir, tmpDir)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestRunHook_NonExistent(t *testing.T) {
	err := RunHook("/nonexistent/path/hook", "branch", "/tmp", "/tmp")
	if err != nil {
		t.Errorf("Expected nil for non-existent hook, got: %v", err)
	}
}

func TestRunHook_NotExecutable(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	hookPath := filepath.Join(tmpDir, "test-hook")
	if err := os.WriteFile(hookPath, []byte("#!/bin/bash\necho hello"), 0644); err != nil {
		t.Fatalf("Failed to write hook: %v", err)
	}

	err = RunHook(hookPath, "branch", tmpDir, tmpDir)
	if err != nil {
		t.Errorf("Expected nil for non-executable hook, got: %v", err)
	}
}

func TestRunHookByName(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create .worktree/hooks/ directory
	hooksDir := filepath.Join(tmpDir, ".worktree", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatalf("Failed to create hooks dir: %v", err)
	}

	// Create a hook script
	hookContent := "#!/bin/bash\necho \"{{ .Branch }}\"\n"
	hookPath := filepath.Join(hooksDir, "post-create")
	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		t.Fatalf("Failed to write hook: %v", err)
	}

	// Create branch directory
	branchDir := filepath.Join(tmpDir, "test-branch")
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		t.Fatalf("Failed to create branch dir: %v", err)
	}

	err = RunHookByName(tmpDir, "post-create", "test-branch", branchDir)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRunHookByName_NonExistent(t *testing.T) {
	err := RunHookByName("/tmp", "nonexistent-hook", "branch", "/tmp")
	if err != nil {
		t.Errorf("Expected nil for non-existent hook, got: %v", err)
	}
}
