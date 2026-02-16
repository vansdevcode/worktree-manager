package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// configureTestGitUser configures git user.name and user.email locally in the test bare repo
// This prevents tests from failing on systems without global git identity configured
func configureTestGitUser(t *testing.T, bareDir string) {
	t.Helper()

	// Configure user.name locally in the bare repo
	cmd := exec.Command("git", "--git-dir="+bareDir, "config", "user.name", "Test User")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user.name: %v", err)
	}

	// Configure user.email locally in the bare repo
	cmd = exec.Command("git", "--git-dir="+bareDir, "config", "user.email", "test@example.com")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user.email: %v", err)
	}
}

func TestConvertGitHubFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "GitHub owner/repo format",
			input:    "owner/repo",
			expected: "git@github.com:owner/repo.git",
		},
		{
			name:     "Already git@ format",
			input:    "git@github.com:owner/repo.git",
			expected: "git@github.com:owner/repo.git",
		},
		{
			name:     "HTTPS URL",
			input:    "https://github.com/owner/repo",
			expected: "https://github.com/owner/repo",
		},
		{
			name:     "HTTPS URL with .git",
			input:    "https://github.com/owner/repo.git",
			expected: "https://github.com/owner/repo.git",
		},
		{
			name:     "GitHub owner/repo with .git suffix",
			input:    "owner/repo.git",
			expected: "git@github.com:owner/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertGitHubFormat(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertGitHubFormat(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCreateInitialBranch(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir: %v", err)
		}
	}()

	bareDir := filepath.Join(tmpDir, "test.bare")

	// Initialize bare repo
	if err := InitBare(bareDir); err != nil {
		t.Fatalf("InitBare failed: %v", err)
	}

	// Configure git identity locally in the test repo
	configureTestGitUser(t, bareDir)

	// Create initial branch
	branchName := "main"
	if err := CreateInitialBranch(bareDir, branchName); err != nil {
		t.Fatalf("CreateInitialBranch failed: %v", err)
	}

	// Verify branch exists
	cmd := exec.Command("git", "--git-dir="+bareDir, "branch", "--list", branchName)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}

	if !strings.Contains(string(output), branchName) {
		t.Errorf("Branch %q not found in output: %s", branchName, string(output))
	}

	// Verify HEAD points to the branch
	cmd = exec.Command("git", "--git-dir="+bareDir, "symbolic-ref", "HEAD")
	output, err = cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}

	expectedRef := "refs/heads/" + branchName
	actualRef := strings.TrimSpace(string(output))
	if actualRef != expectedRef {
		t.Errorf("HEAD = %q, want %q", actualRef, expectedRef)
	}

	// Verify the branch has a commit
	cmd = exec.Command("git", "--git-dir="+bareDir, "rev-parse", branchName)
	output, err = cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get commit: %v", err)
	}

	commitHash := strings.TrimSpace(string(output))
	if len(commitHash) != 40 {
		t.Errorf("Expected 40-char commit hash, got: %s", commitHash)
	}

	// Verify GetDefaultBranch works now
	defaultBranch, err := GetDefaultBranch(bareDir)
	if err != nil {
		t.Fatalf("GetDefaultBranch failed: %v", err)
	}

	if defaultBranch != branchName {
		t.Errorf("GetDefaultBranch() = %q, want %q", defaultBranch, branchName)
	}
}

func TestGetWorktreeBranch(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "git-worktree-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir: %v", err)
		}
	}()

	bareDir := filepath.Join(tmpDir, "test.bare")

	// Initialize bare repo
	if err := InitBare(bareDir); err != nil {
		t.Fatalf("InitBare failed: %v", err)
	}

	// Configure git identity locally in the test repo
	configureTestGitUser(t, bareDir)

	// Create initial branch
	if err := CreateInitialBranch(bareDir, "main"); err != nil {
		t.Fatalf("CreateInitialBranch failed: %v", err)
	}

	tests := []struct {
		name          string
		setupBranch   string
		expectError   bool
		errorContains string
	}{
		{
			name:        "normal branch",
			setupBranch: "main",
			expectError: false,
		},
		{
			name:        "branch with slashes",
			setupBranch: "feature/test-branch",
			expectError: false,
		},
		{
			name:        "branch with underscores",
			setupBranch: "fix_bug_123",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Sanitize the directory name (not the full path)
			dirName := "worktree-" + tt.setupBranch
			dirName = strings.ReplaceAll(dirName, "/", "-")
			dirName = strings.ReplaceAll(dirName, "_", "-")
			worktreePath := filepath.Join(tmpDir, dirName)

			// Create branch if not main
			if tt.setupBranch != "main" {
				if err := AddWorktree(bareDir, tt.setupBranch, worktreePath, "main"); err != nil {
					t.Fatalf("Failed to create worktree: %v", err)
				}
			} else {
				if err := AddWorktree(bareDir, tt.setupBranch, worktreePath, ""); err != nil {
					t.Fatalf("Failed to create worktree: %v", err)
				}
			}
			defer func() {
				if err := os.RemoveAll(worktreePath); err != nil {
					t.Errorf("Failed to remove worktree: %v", err)
				}
			}()

			// Get branch name
			branch, err := GetWorktreeBranch(worktreePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorContains)
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if branch != tt.setupBranch {
					t.Errorf("GetWorktreeBranch() = %q, want %q", branch, tt.setupBranch)
				}
			}
		})
	}
}

func TestGetWorktreeBranch_DetachedHead(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "git-worktree-detached-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir: %v", err)
		}
	}()

	bareDir := filepath.Join(tmpDir, "test.bare")

	// Initialize bare repo
	if err := InitBare(bareDir); err != nil {
		t.Fatalf("InitBare failed: %v", err)
	}

	// Configure git identity locally in the test repo
	configureTestGitUser(t, bareDir)

	// Create initial branch
	if err := CreateInitialBranch(bareDir, "main"); err != nil {
		t.Fatalf("CreateInitialBranch failed: %v", err)
	}

	// Create worktree
	worktreePath := filepath.Join(tmpDir, "worktree-detached")
	if err := AddWorktree(bareDir, "main", worktreePath, ""); err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(worktreePath); err != nil {
			t.Errorf("Failed to remove worktree: %v", err)
		}
	}()

	// Get the commit hash to checkout
	cmd := exec.Command("git", "-C", worktreePath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get commit hash: %v", err)
	}
	commitHash := strings.TrimSpace(string(output))

	// Checkout commit (detached HEAD)
	cmd = exec.Command("git", "-C", worktreePath, "checkout", commitHash)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to checkout commit: %v", err)
	}

	// Try to get branch name - should fail
	_, err = GetWorktreeBranch(worktreePath)
	if err == nil {
		t.Error("Expected error for detached HEAD, got nil")
	}
	if !strings.Contains(err.Error(), "detached HEAD") {
		t.Errorf("Expected error containing 'detached HEAD', got %q", err.Error())
	}
}

func TestGetWorktreeBranch_NonExistent(t *testing.T) {
	// Try to get branch from non-existent directory
	_, err := GetWorktreeBranch("/non/existent/path")
	if err == nil {
		t.Error("Expected error for non-existent path, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get branch name") {
		t.Errorf("Expected error containing 'failed to get branch name', got %q", err.Error())
	}
}

func TestLocalBranchExists(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "git-local-branch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir: %v", err)
		}
	}()

	bareDir := filepath.Join(tmpDir, "test.bare")

	// Initialize bare repo
	if err := InitBare(bareDir); err != nil {
		t.Fatalf("InitBare failed: %v", err)
	}

	// Configure git identity locally in the test repo
	configureTestGitUser(t, bareDir)

	// Create initial branch
	if err := CreateInitialBranch(bareDir, "main"); err != nil {
		t.Fatalf("CreateInitialBranch failed: %v", err)
	}

	// Create a remote branch reference (simulating a fetched remote branch)
	// This mimics what happens when you fetch a remote branch
	cmd := exec.Command("git", "--git-dir="+bareDir, "update-ref", "refs/remotes/origin/develop", "refs/heads/main")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create remote ref: %v", err)
	}

	tests := []struct {
		name   string
		branch string
		want   bool
	}{
		{
			name:   "local branch exists",
			branch: "main",
			want:   true,
		},
		{
			name:   "local branch does not exist",
			branch: "feature-x",
			want:   false,
		},
		{
			name:   "remote-only branch (develop exists as origin/develop, not locally)",
			branch: "develop",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LocalBranchExists(bareDir, tt.branch)
			if err != nil {
				t.Errorf("LocalBranchExists() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("LocalBranchExists(%q) = %v, want %v", tt.branch, got, tt.want)
			}
		})
	}

	// Verify that BranchExists returns true for remote-only branch (old behavior)
	t.Run("BranchExists returns true for remote-only branch", func(t *testing.T) {
		exists, err := BranchExists(bareDir, "develop")
		if err != nil {
			t.Errorf("BranchExists() error = %v", err)
			return
		}
		if !exists {
			t.Error("BranchExists() should return true for remote-only branch (origin/develop)")
		}
	})

	// Verify LocalBranchExists returns false for same remote-only branch
	t.Run("LocalBranchExists returns false for remote-only branch", func(t *testing.T) {
		exists, err := LocalBranchExists(bareDir, "develop")
		if err != nil {
			t.Errorf("LocalBranchExists() error = %v", err)
			return
		}
		if exists {
			t.Error("LocalBranchExists() should return false for remote-only branch (origin/develop)")
		}
	})
}
