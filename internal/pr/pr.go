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

	// Get PR info using gh CLI
	cmd := exec.Command("gh", "pr", "view", fmt.Sprintf("%d", prNumber), "--json", "headRefName,headRepository")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh pr view failed: %w", err)
	}

	var prInfo struct {
		HeadRefName string `json:"headRefName"`
	}
	if err := json.Unmarshal(output, &prInfo); err != nil {
		return "", fmt.Errorf("failed to parse PR info: %w", err)
	}

	// Fetch the PR
	refSpec := fmt.Sprintf("pull/%d/head:%s", prNumber, branchName)
	if err := git.FetchRef(bareDir, refSpec); err != nil {
		return "", err
	}

	return branchName, nil
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
func GetPRInfo(prNumber int) (title string, description string, err error) {
	// Check if gh CLI is available
	if _, err := exec.LookPath("gh"); err != nil {
		return "", "", fmt.Errorf("gh CLI not available")
	}

	cmd := exec.Command("gh", "pr", "view", fmt.Sprintf("%d", prNumber), "--json", "title,body")
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
