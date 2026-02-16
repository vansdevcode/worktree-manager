package pr

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/vansdevcode/worktree-manager/internal/git"
)

// FetchPR fetches a pull request using the three-tier fallback strategy
// Returns the branch name that was created
func FetchPR(bareDir string, prNumber int, branchName string) (string, error) {
	// Tier 1: Try gh CLI first (provides best metadata)
	branch, err := fetchPRWithGH(bareDir, prNumber, branchName)
	if err == nil {
		return branch, nil
	}

	// Tier 2: Try GitHub API (requires no dependencies but needs network)
	// For now, skip API implementation as it requires token management
	// TODO: Implement GitHub API fallback

	// Tier 3: Use pull/$ID/head refspec (always works)
	return fetchPRWithRefspec(bareDir, prNumber, branchName)
}

// fetchPRWithGH uses gh CLI to fetch PR information and check it out
func fetchPRWithGH(bareDir string, prNumber int, branchName string) (string, error) {
	// Check if gh CLI is available
	if _, err := exec.LookPath("gh"); err != nil {
		return "", fmt.Errorf("gh CLI not found")
	}

	// Get the repository slug from the remote URL
	repoSlug, err := getRepoSlug(bareDir)
	if err != nil {
		return "", fmt.Errorf("failed to determine repository: %w", err)
	}

	// Get PR info using gh CLI with explicit --repo flag
	cmd := exec.Command("gh", "pr", "view", fmt.Sprintf("%d", prNumber), "--repo", repoSlug, "--json", "headRefName")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh pr view failed: %w; output: %s", err, strings.TrimSpace(string(output)))
	}

	var prInfo struct {
		HeadRefName string `json:"headRefName"`
	}
	if err := json.Unmarshal(output, &prInfo); err != nil {
		return "", fmt.Errorf("failed to parse PR info: %w", err)
	}

	// Use the actual PR branch name from GitHub metadata
	actualBranch := prInfo.HeadRefName
	if actualBranch == "" {
		return "", fmt.Errorf("PR metadata missing headRefName")
	}

	// Fetch the PR using the actual branch name
	refSpec := fmt.Sprintf("pull/%d/head:%s", prNumber, actualBranch)
	if err := git.FetchRef(bareDir, refSpec); err != nil {
		return "", err
	}

	return actualBranch, nil
}

// getRepoSlug extracts the repository slug (owner/repo) from the remote URL
func getRepoSlug(bareDir string) (string, error) {
	cmd := exec.Command("git", "--git-dir="+bareDir, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}

	remoteURL := strings.TrimSpace(string(output))

	// Parse different URL formats:
	// - https://github.com/owner/repo.git
	// - git@github.com:owner/repo.git
	// - https://github.com/owner/repo

	// Remove .git suffix
	remoteURL = strings.TrimSuffix(remoteURL, ".git")

	// Handle SSH format: git@github.com:owner/repo
	if strings.HasPrefix(remoteURL, "git@github.com:") {
		return strings.TrimPrefix(remoteURL, "git@github.com:"), nil
	}

	// Handle HTTPS format: https://github.com/owner/repo
	if strings.HasPrefix(remoteURL, "https://github.com/") {
		return strings.TrimPrefix(remoteURL, "https://github.com/"), nil
	}

	// Handle HTTP format: http://github.com/owner/repo
	if strings.HasPrefix(remoteURL, "http://github.com/") {
		return strings.TrimPrefix(remoteURL, "http://github.com/"), nil
	}

	return "", fmt.Errorf("unsupported remote URL format: %s", remoteURL)
}

// fetchPRWithRefspec uses the pull/$ID/head refspec to fetch the PR
func fetchPRWithRefspec(bareDir string, prNumber int, branchName string) (string, error) {
	// Fetch using pull/$ID/head refspec
	refSpec := fmt.Sprintf("pull/%d/head:%s", prNumber, branchName)

	cmd := exec.Command("git", "--git-dir="+bareDir, "fetch", "origin", refSpec)
	err := cmd.Run()
	if err != nil {
		// Try alternative format
		refSpec = fmt.Sprintf("+refs/pull/%d/head:refs/heads/%s", prNumber, branchName)
		cmd = exec.Command("git", "--git-dir="+bareDir, "fetch", "origin", refSpec)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to fetch PR: %s", string(output))
		}
	}

	return branchName, nil
}

// GetPRInfo attempts to get PR information (best effort)
// bareDir is required to determine the repository context
func GetPRInfo(bareDir string, prNumber int) (title string, description string, err error) {
	// Check if gh CLI is available
	if _, err := exec.LookPath("gh"); err != nil {
		return "", "", fmt.Errorf("gh CLI not available")
	}

	// Get the repository slug from the remote URL
	repoSlug, err := getRepoSlug(bareDir)
	if err != nil {
		return "", "", fmt.Errorf("failed to determine repository: %w", err)
	}

	cmd := exec.Command("gh", "pr", "view", fmt.Sprintf("%d", prNumber), "--repo", repoSlug, "--json", "title,body")
	output, err := cmd.Output()
	if err != nil {
		return "", "", err
	}

	var prInfo struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := json.Unmarshal(output, &prInfo); err != nil {
		return "", "", err
	}

	return strings.TrimSpace(prInfo.Title), strings.TrimSpace(prInfo.Body), nil
}
