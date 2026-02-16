package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// ConvertGitHubFormat converts GitHub shorthand to git URL
func ConvertGitHubFormat(repo string) string {
	// If already a full URL, return as-is
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") || strings.HasPrefix(repo, "git@") {
		return repo
	}

	// Convert owner/repo to git@github.com:owner/repo.git
	if strings.Contains(repo, "/") && !strings.Contains(repo, ":") {
		return fmt.Sprintf("git@github.com:%s.git", repo)
	}

	return repo
}

// CloneBare clones a repository as a bare repository
func CloneBare(url, bareDir string) error {
	cmd := exec.Command("git", "clone", "--bare", url, bareDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %s", string(output))
	}
	return nil
}

// InitBare initializes a new bare repository
func InitBare(bareDir string) error {
	cmd := exec.Command("git", "init", "--bare", bareDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git init failed: %s", string(output))
	}
	return nil
}

// GetDefaultBranch returns the default branch of the repository
func GetDefaultBranch(bareDir string) (string, error) {
	// Try to get symbolic ref
	cmd := exec.Command("git", "--git-dir="+bareDir, "symbolic-ref", "refs/remotes/origin/HEAD")
	output, err := cmd.Output()
	if err == nil {
		// Parse refs/remotes/origin/main -> main
		branch := strings.TrimSpace(string(output))
		branch = strings.TrimPrefix(branch, "refs/remotes/origin/")
		return branch, nil
	}

	// Fallback: try common branch names
	for _, branch := range []string{"main", "master", "develop"} {
		cmd := exec.Command("git", "--git-dir="+bareDir, "show-ref", "--verify", "refs/heads/"+branch)
		if err := cmd.Run(); err == nil {
			return branch, nil
		}
	}

	// Last resort: get first branch
	cmd = exec.Command("git", "--git-dir="+bareDir, "branch", "--format=%(refname:short)")
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("no branches found")
	}

	branches := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(branches) > 0 && branches[0] != "" {
		return branches[0], nil
	}

	return "", fmt.Errorf("no branches found")
}

// AddWorktree adds a new worktree
// If startPoint is empty, branch must exist
// If startPoint is provided, creates new branch from startPoint
func AddWorktree(bareDir, branch, path, startPoint string) error {
	args := []string{"--git-dir=" + bareDir, "worktree", "add"}

	if startPoint != "" {
		// Create new branch from startPoint
		args = append(args, "-b", branch, path, startPoint)
	} else {
		// Use existing branch
		args = append(args, path, branch)
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree add failed: %s", string(output))
	}

	// Initialize submodules if any
	cmd = exec.Command("git", "-C", path, "submodule", "update", "--init", "--recursive")
	_ = cmd.Run() // Ignore errors as submodules may not exist

	return nil
}

// BranchExists checks if a branch exists in the repository
func BranchExists(bareDir, branch string) (bool, error) {
	cmd := exec.Command("git", "--git-dir="+bareDir, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	err := cmd.Run()
	if err == nil {
		return true, nil
	}

	// Check if it's a remote branch
	cmd = exec.Command("git", "--git-dir="+bareDir, "show-ref", "--verify", "--quiet", "refs/remotes/origin/"+branch)
	err = cmd.Run()
	return err == nil, nil
}

// ListWorktrees lists all worktrees
func ListWorktrees(bareDir string) (string, error) {
	cmd := exec.Command("git", "--git-dir="+bareDir, "worktree", "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git worktree list failed: %s", string(output))
	}
	return string(output), nil
}

// RemoveWorktree removes a worktree
func RemoveWorktree(bareDir, path string) error {
	cmd := exec.Command("git", "--git-dir="+bareDir, "worktree", "remove", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree remove failed: %s", string(output))
	}
	return nil
}

// RemoveWorktreeForce removes a worktree forcefully
func RemoveWorktreeForce(bareDir, path string) error {
	cmd := exec.Command("git", "--git-dir="+bareDir, "worktree", "remove", "--force", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree remove failed: %s", string(output))
	}
	return nil
}

// DeleteBranch deletes a branch
func DeleteBranch(bareDir, branch string) error {
	cmd := exec.Command("git", "--git-dir="+bareDir, "branch", "-D", branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git branch delete failed: %s", string(output))
	}
	return nil
}

// HasUncommittedChanges checks if the worktree has uncommitted changes
func HasUncommittedChanges(path string) (bool, error) {
	cmd := exec.Command("git", "-C", path, "diff-index", "--quiet", "HEAD", "--")
	err := cmd.Run()
	if err != nil {
		// Exit code 1 means there are changes
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

// HasUntrackedFiles checks if the worktree has untracked files
func HasUntrackedFiles(path string) (bool, error) {
	cmd := exec.Command("git", "-C", path, "ls-files", "--others", "--exclude-standard")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(output))) > 0, nil
}

// FetchRef fetches a specific ref from origin
func FetchRef(bareDir, ref string) error {
	cmd := exec.Command("git", "--git-dir="+bareDir, "fetch", "origin", ref)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch failed: %s", string(output))
	}
	return nil
}
