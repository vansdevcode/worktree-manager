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
			tmpDir, err := os.MkdirTemp("", "hook-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()

			branchDir := filepath.Join(tmpDir, tt.branch)
			if err := os.MkdirAll(branchDir, 0755); err != nil {
				t.Fatalf("Failed to create branch dir: %v", err)
			}

			hookPath := filepath.Join(tmpDir, "test-hook")
			if err := os.WriteFile(hookPath, []byte(tt.scriptContent), 0755); err != nil {
				t.Fatalf("Failed to write hook: %v", err)
			}

			err = RunHook(hookPath, tt.branch, branchDir, tmpDir, nil)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestRunHook_WithVars(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	branchDir := filepath.Join(tmpDir, "feat-branch")
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		t.Fatalf("Failed to create branch dir: %v", err)
	}

	hookPath := filepath.Join(tmpDir, "test-hook")
	script := "#!/bin/bash\necho \"{{ index .Vars \"ticket\" }}\"\n"
	if err := os.WriteFile(hookPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to write hook: %v", err)
	}

	vars := map[string]string{"ticket": "PROJ-456"}
	err = RunHook(hookPath, "feat-branch", branchDir, tmpDir, vars)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRunHook_NonExistent(t *testing.T) {
	err := RunHook("/nonexistent/path/hook", "branch", "/tmp", "/tmp", nil)
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

	err = RunHook(hookPath, "branch", tmpDir, tmpDir, nil)
	if err != nil {
		t.Errorf("Expected nil for non-executable hook, got: %v", err)
	}
}

func TestRunHookInline(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	branchDir := filepath.Join(tmpDir, "test-branch")
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		t.Fatalf("Failed to create branch dir: %v", err)
	}

	// Inline content without shebang — should auto-prepend #!/bin/bash
	err = RunHookInline("echo \"{{ .Branch }}\"", "post-create", "test-branch", branchDir, tmpDir, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRunHookInline_WithShebang(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	branchDir := filepath.Join(tmpDir, "test-branch")
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		t.Fatalf("Failed to create branch dir: %v", err)
	}

	content := "#!/bin/bash\necho \"{{ .Branch }}\"\n"
	err = RunHookInline(content, "post-create", "test-branch", branchDir, tmpDir, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRunHookInline_Empty(t *testing.T) {
	err := RunHookInline("", "post-create", "branch", "/tmp", "/tmp", nil)
	if err != nil {
		t.Errorf("Expected nil for empty content, got: %v", err)
	}
}

func TestRunHook_NonZeroExitReturnsError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	hookPath := filepath.Join(tmpDir, "failing-hook")
	if err := os.WriteFile(hookPath, []byte("#!/bin/bash\nexit 1\n"), 0755); err != nil {
		t.Fatalf("Failed to write hook: %v", err)
	}

	err = RunHook(hookPath, "branch", tmpDir, tmpDir, nil)
	if err == nil {
		t.Error("Expected error for non-zero exit code, got nil")
	}
}

func TestRunHook_BranchDirNotExist(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	hookPath := filepath.Join(tmpDir, "pre-create")
	if err := os.WriteFile(hookPath, []byte("#!/bin/bash\necho pre-create\n"), 0755); err != nil {
		t.Fatalf("Failed to write hook: %v", err)
	}

	nonExistentDir := filepath.Join(tmpDir, "does-not-exist")
	err = RunHook(hookPath, "new-branch", nonExistentDir, tmpDir, nil)
	if err != nil {
		t.Errorf("Expected nil when branchDir doesn't exist, got: %v", err)
	}
}
