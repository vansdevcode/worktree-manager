package worktree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRoot(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()

	// Create nested directories with .bare at the top
	bareDir := filepath.Join(tempDir, ".bare")
	if err := os.Mkdir(bareDir, 0755); err != nil {
		t.Fatal(err)
	}

	nestedDir := filepath.Join(tempDir, "level1", "level2")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Save current dir
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	// Change to nested directory
	if err := os.Chdir(nestedDir); err != nil {
		t.Fatal(err)
	}

	// FindRoot should walk up and find tempDir
	root, err := FindRoot()
	if err != nil {
		t.Errorf("FindRoot() error = %v", err)
	}

	// Resolve symlinks for both paths (handles macOS /var -> /private/var)
	rootResolved, _ := filepath.EvalSymlinks(root)
	tempDirResolved, _ := filepath.EvalSymlinks(tempDir)

	if rootResolved != tempDirResolved {
		t.Errorf("FindRoot() = %q, want %q", rootResolved, tempDirResolved)
	}
}

func TestFindRoot_NotFound(t *testing.T) {
	// Create temp dir WITHOUT .bare
	tempDir := t.TempDir()

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	// FindRoot should return error
	_, err = FindRoot()
	if err == nil {
		t.Error("FindRoot() expected error, got nil")
	}
}

func TestGetFilesDir(t *testing.T) {
	root := "/test/root"
	expected := filepath.Join(root, ".worktree", "files")
	result := GetFilesDir(root)

	if result != expected {
		t.Errorf("GetFilesDir(%q) = %q, want %q", root, result, expected)
	}
}

func TestGetHookPath(t *testing.T) {
	root := "/test/root"
	hookName := "post-create"
	expected := filepath.Join(root, ".worktree", hookName)
	result := GetHookPath(root, hookName)

	if result != expected {
		t.Errorf("GetHookPath(%q, %q) = %q, want %q", root, hookName, result, expected)
	}
}

func TestGenerateWorktreeDirectory(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple branch name",
			input:    "feature-branch",
			expected: "feature-branch",
		},
		{
			name:     "Branch with slashes",
			input:    "feature/user-auth",
			expected: "feature-user-auth",
		},
		{
			name:     "Mixed case - preserves case",
			input:    "Feature/User-Auth",
			expected: "Feature-User-Auth",
		},
		{
			name:     "With underscores",
			input:    "fix_bug_123",
			expected: "fix-bug-123",
		},
		{
			name:     "With special characters",
			input:    "feature/user@auth#123",
			expected: "feature-userauth123",
		},
		{
			name:     "Multiple consecutive special chars",
			input:    "feature///user---auth",
			expected: "feature-user-auth",
		},
		{
			name:     "Uppercase branch",
			input:    "JUI-751",
			expected: "JUI-751",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateWorktreeDirectory(tt.input)
			if result != tt.expected {
				t.Errorf("GenerateWorktreeDirectory(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
