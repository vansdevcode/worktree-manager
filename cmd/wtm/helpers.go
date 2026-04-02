package main

import (
	"os"
	"path/filepath"

	"github.com/vansdevcode/worktree-manager/internal/hook"
	"github.com/vansdevcode/worktree-manager/internal/wtmconfig"
)

// runConfigHook runs a hook from the wtm config. It checks:
// 1. If the hook value is a file path starting with ".wtm/hooks/", run it as a file.
// 2. Otherwise, treat it as inline script content.
func runConfigHook(cfg *wtmconfig.Config, hookName, branchName, branchDirectory, rootDirectory string, vars map[string]string) error {
	if cfg == nil || len(cfg.Hooks) == 0 {
		return nil
	}

	content, ok := cfg.Hooks[hookName]
	if !ok || content == "" {
		return nil
	}

	// Check if it's a file reference
	if isHookFileRef(content) {
		hookPath := filepath.Join(rootDirectory, content)
		return hook.RunHook(hookPath, branchName, branchDirectory, rootDirectory, vars)
	}

	return hook.RunHookInline(content, hookName, branchName, branchDirectory, rootDirectory, vars)
}

// isHookFileRef returns true if the hook value looks like a file path reference
// rather than inline script content.
func isHookFileRef(value string) bool {
	// If the value is a path to a file that exists, treat it as a file reference
	if _, err := os.Stat(value); err == nil {
		return true
	}
	return false
}
