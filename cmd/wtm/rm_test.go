package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vansdevcode/worktree-manager/internal/config"
	"github.com/vansdevcode/worktree-manager/internal/git"
)

// setupTestRepo creates a temporary git repository with bare setup
func setupTestRepo(t *testing.T) (rootDir, bareDir string, cleanup func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "wtm-rm-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	rootDir = tmpDir
	bareDir = config.GetBareDir(rootDir)

	// Initialize bare repo
	if err := git.InitBare(bareDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("InitBare failed: %v", err)
	}

	// Create initial branch
	if err := git.CreateInitialBranch(bareDir, "main"); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("CreateInitialBranch failed: %v", err)
	}

	cleanup = func() {
		_ = os.RemoveAll(tmpDir)
	}

	return rootDir, bareDir, cleanup
}

// TestRmCommand_NormalBranch tests removing a worktree with a normal branch
func TestRmCommand_NormalBranch(t *testing.T) {
	rootDir, bareDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create worktree
	worktreePath := filepath.Join(rootDir, "test-worktree")
	if err := git.AddWorktree(bareDir, "main", worktreePath, ""); err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}

	// Change to root directory (not in worktree)
	oldDir, _ := os.Getwd()
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("Failed to change to root directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	// Run rm command
	err := runRm(rmCmd, []string{"test-worktree"})
	if err != nil {
		t.Errorf("runRm failed: %v", err)
	}

	// Verify worktree is removed
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Errorf("Worktree still exists after removal")
	}
}

// TestRmCommand_MismatchedDirectory tests removing a worktree where directory name != branch name
func TestRmCommand_MismatchedDirectory(t *testing.T) {
	rootDir, bareDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create worktree with custom directory (branch=feature/test, dir=my-custom-dir)
	customDir := filepath.Join(rootDir, "my-custom-dir")
	if err := git.AddWorktree(bareDir, "feature/test", customDir, "main"); err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}

	// Verify branch name is correct
	branch, err := git.GetWorktreeBranch(customDir)
	if err != nil {
		t.Fatalf("Failed to get branch: %v", err)
	}
	if branch != "feature/test" {
		t.Fatalf("Expected branch 'feature/test', got '%s'", branch)
	}

	// Change to root directory
	oldDir, _ := os.Getwd()
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("Failed to change to root directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	// Run rm command with --delete-branch
	rmDeleteBranch = true
	defer func() { rmDeleteBranch = false }()

	err = runRm(rmCmd, []string{"my-custom-dir"})
	if err != nil {
		t.Errorf("runRm failed: %v", err)
	}

	// Verify worktree is removed
	if _, err := os.Stat(customDir); !os.IsNotExist(err) {
		t.Errorf("Worktree still exists after removal")
	}

	// Verify correct branch was deleted (feature/test, not my-custom-dir)
	exists, err := git.BranchExists(bareDir, "feature/test")
	if err != nil {
		t.Fatalf("Failed to check if branch exists: %v", err)
	}
	if exists {
		t.Errorf("Branch 'feature/test' still exists, should have been deleted")
	}
}

// TestRmCommand_PRDirectory tests removing a PR worktree (e.g., pr-123 dir, but actual branch is different)
func TestRmCommand_PRDirectory(t *testing.T) {
	rootDir, bareDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Simulate PR workflow: create worktree with PR branch
	prDir := filepath.Join(rootDir, "pr-123")
	prBranch := "feature/add-new-feature"
	if err := git.AddWorktree(bareDir, prBranch, prDir, "main"); err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}

	// Verify branch name is correct (not "pr-123")
	branch, err := git.GetWorktreeBranch(prDir)
	if err != nil {
		t.Fatalf("Failed to get branch: %v", err)
	}
	if branch != prBranch {
		t.Fatalf("Expected branch '%s', got '%s'", prBranch, branch)
	}

	// Change to root directory
	oldDir, _ := os.Getwd()
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("Failed to change to root directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	// Run rm command with --delete-branch
	rmDeleteBranch = true
	defer func() { rmDeleteBranch = false }()

	err = runRm(rmCmd, []string{"pr-123"})
	if err != nil {
		t.Errorf("runRm failed: %v", err)
	}

	// Verify worktree is removed
	if _, err := os.Stat(prDir); !os.IsNotExist(err) {
		t.Errorf("Worktree still exists after removal")
	}

	// Verify correct branch was deleted (feature/add-new-feature, not pr-123)
	exists, err := git.BranchExists(bareDir, prBranch)
	if err != nil {
		t.Fatalf("Failed to check if branch exists: %v", err)
	}
	if exists {
		t.Errorf("Branch '%s' still exists, should have been deleted", prBranch)
	}
}

// TestRmCommand_DetachedHeadWithoutForce tests that removing detached HEAD worktree fails without --force
func TestRmCommand_DetachedHeadWithoutForce(t *testing.T) {
	rootDir, bareDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create worktree
	worktreePath := filepath.Join(rootDir, "test-worktree")
	if err := git.AddWorktree(bareDir, "main", worktreePath, ""); err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}

	// Get the commit SHA
	cmd := exec.Command("git", "-C", worktreePath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get commit SHA: %v", err)
	}
	commitSHA := strings.TrimSpace(string(output))

	// Checkout detached HEAD
	cmd = exec.Command("git", "-C", worktreePath, "checkout", commitSHA)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to checkout detached HEAD: %v", err)
	}

	// Verify we're in detached HEAD
	cmd = exec.Command("git", "-C", worktreePath, "symbolic-ref", "-q", "HEAD")
	if err := cmd.Run(); err == nil {
		t.Fatal("Expected detached HEAD, but symbolic-ref succeeded")
	}

	// Change to root directory
	oldDir, _ := os.Getwd()
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("Failed to change to root directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	// Run rm command without --force, should fail
	rmForce = false
	err = runRm(rmCmd, []string{"test-worktree"})
	if err == nil {
		t.Errorf("Expected error when removing detached HEAD worktree without --force, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "detached HEAD") {
		t.Errorf("Expected 'detached HEAD' error, got: %v", err)
	}

	// Verify worktree still exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Errorf("Worktree was removed despite error")
	}
}

// TestRmCommand_DetachedHeadWithForce tests that removing detached HEAD worktree succeeds with --force
func TestRmCommand_DetachedHeadWithForce(t *testing.T) {
	rootDir, bareDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create worktree
	worktreePath := filepath.Join(rootDir, "test-worktree")
	if err := git.AddWorktree(bareDir, "main", worktreePath, ""); err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}

	// Get the commit SHA
	cmd := exec.Command("git", "-C", worktreePath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get commit SHA: %v", err)
	}
	commitSHA := strings.TrimSpace(string(output))

	// Checkout detached HEAD
	cmd = exec.Command("git", "-C", worktreePath, "checkout", commitSHA)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to checkout detached HEAD: %v", err)
	}

	// Change to root directory
	oldDir, _ := os.Getwd()
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("Failed to change to root directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	// Run rm command with --force, should succeed
	rmForce = true
	defer func() { rmForce = false }()

	err = runRm(rmCmd, []string{"test-worktree"})
	if err != nil {
		t.Errorf("runRm with --force failed: %v", err)
	}

	// Verify worktree is removed
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Errorf("Worktree still exists after removal with --force")
	}
}

// TestRmCommand_UncommittedChanges tests that removing worktree with uncommitted changes fails without --force
func TestRmCommand_UncommittedChanges(t *testing.T) {
	rootDir, bareDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create worktree
	worktreePath := filepath.Join(rootDir, "test-worktree")
	if err := git.AddWorktree(bareDir, "main", worktreePath, ""); err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}

	// Create uncommitted changes
	testFile := filepath.Join(worktreePath, "test.txt")
	if err := os.WriteFile(testFile, []byte("uncommitted"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	cmd := exec.Command("git", "-C", worktreePath, "add", "test.txt")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to stage file: %v", err)
	}

	// Change to root directory
	oldDir, _ := os.Getwd()
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("Failed to change to root directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	// Run rm command without --force, should fail
	rmForce = false
	err := runRm(rmCmd, []string{"test-worktree"})
	if err == nil {
		t.Errorf("Expected error when removing worktree with uncommitted changes, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "uncommitted changes") {
		t.Errorf("Expected 'uncommitted changes' error, got: %v", err)
	}

	// Verify worktree still exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Errorf("Worktree was removed despite error")
	}
}

// TestRmCommand_ForceWithUncommittedChanges tests that removing worktree with uncommitted changes succeeds with --force
func TestRmCommand_ForceWithUncommittedChanges(t *testing.T) {
	rootDir, bareDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create worktree
	worktreePath := filepath.Join(rootDir, "test-worktree")
	if err := git.AddWorktree(bareDir, "main", worktreePath, ""); err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}

	// Create uncommitted changes
	testFile := filepath.Join(worktreePath, "test.txt")
	if err := os.WriteFile(testFile, []byte("uncommitted"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	cmd := exec.Command("git", "-C", worktreePath, "add", "test.txt")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to stage file: %v", err)
	}

	// Change to root directory
	oldDir, _ := os.Getwd()
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("Failed to change to root directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	// Run rm command with --force, should succeed
	rmForce = true
	defer func() { rmForce = false }()

	err := runRm(rmCmd, []string{"test-worktree"})
	if err != nil {
		t.Errorf("runRm with --force failed: %v", err)
	}

	// Verify worktree is removed
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Errorf("Worktree still exists after removal with --force")
	}
}
