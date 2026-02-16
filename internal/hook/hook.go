package hook

import (
	"os"
	"os/exec"
)

// HookData contains data passed to hooks via environment variables
type HookData struct {
	RootDirectory string
	Directory     string
	Branch        string
}

// RunHook executes a hook script with the provided environment variables
func RunHook(hookPath string, data HookData) error {
	// Check if hook exists and is executable
	info, err := os.Stat(hookPath)
	if err != nil {
		return err
	}

	// Check if file is executable
	if info.Mode()&0111 == 0 {
		return nil // Not executable, skip silently
	}

	// Prepare environment variables
	env := os.Environ()
	env = append(env,
		"WT_ROOT_DIRECTORY="+data.RootDirectory,
		"WT_DIRECTORY="+data.Directory,
		"WT_BRANCH="+data.Branch,
	)

	// Execute hook
	cmd := exec.Command(hookPath)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = data.Directory

	return cmd.Run()
}

// PrepareEnv creates environment variables for hook execution (deprecated)
// Use HookData and RunHook instead
func PrepareEnv(rootDir, directory, branch string) map[string]string {
	return map[string]string{
		"WT_ROOT_DIRECTORY": rootDir,
		"WT_DIRECTORY":      directory,
		"WT_BRANCH":         branch,
	}
}
