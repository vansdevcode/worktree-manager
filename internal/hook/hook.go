package hook

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/hairyhenderson/gomplate/v4"
	"github.com/vansdevcode/worktree-manager/pkg/ui"
)

// TemplateData holds the data available to hook templates
type TemplateData struct {
	Branch        string
	Directory     string
	RootDirectory string
	Vars          map[string]string
}

// RunHook processes a hook script as a Go template and executes it.
// hookPath: path to the hook script file
// branchName: the branch name (for template data)
// branchDirectory: absolute path to the worktree directory
// rootDirectory: absolute path to the repository root
// vars: user-defined custom variables (may be nil)
func RunHook(hookPath, branchName, branchDirectory, rootDirectory string, vars map[string]string) error {
	// Check if hook exists and is executable
	info, err := os.Stat(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Hook doesn't exist, skip silently
		}
		return err
	}

	// Check if file is executable
	if info.Mode()&0111 == 0 {
		return nil // Not executable, skip silently
	}

	// Read script content
	content, err := os.ReadFile(hookPath)
	if err != nil {
		return fmt.Errorf("failed to read hook script: %w", err)
	}

	return runHookContent(string(content), filepath.Base(hookPath), branchName, branchDirectory, rootDirectory, vars)
}

// RunHookInline processes inline hook content (from .wtm.toml) as a Go template and executes it.
// If the content does not start with a shebang, "#!/bin/bash" is prepended.
func RunHookInline(content, hookName, branchName, branchDirectory, rootDirectory string, vars map[string]string) error {
	if content == "" {
		return nil
	}

	// If no shebang, prepend #!/bin/bash
	if !strings.HasPrefix(content, "#!") {
		content = "#!/bin/bash\n" + content
	}

	return runHookContent(content, hookName, branchName, branchDirectory, rootDirectory, vars)
}

// runHookContent is the shared implementation for running hook content.
func runHookContent(content, name, branchName, branchDirectory, rootDirectory string, vars map[string]string) error {
	templateData := TemplateData{
		Branch:        branchName,
		Directory:     branchDirectory,
		RootDirectory: rootDirectory,
		Vars:          vars,
	}

	ctx := context.Background()
	funcMap := gomplate.CreateFuncs(ctx)

	tmpl, err := template.New(name).Funcs(funcMap).Parse(content)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var outputBuffer bytes.Buffer
	if err := tmpl.Execute(&outputBuffer, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Extract shebang from processed content to determine interpreter
	processedContent := outputBuffer.String()
	interpreter, scriptContent := ExtractShebang(processedContent)

	if interpreter == "" {
		return fmt.Errorf("no shebang found in hook script %s", name)
	}

	// Create temp file with processed script
	tmpFile, err := os.CreateTemp("", "wtm-hook-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// Write processed script content (without shebang since we're using interpreter directly)
	if _, err := tmpFile.WriteString(scriptContent); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	_ = tmpFile.Close()

	// Parse interpreter — handle "/usr/bin/env python3" style shebangs
	interpreterParts := strings.Fields(interpreter)

	cmd := exec.Command(interpreterParts[0], append(interpreterParts[1:], tmpFile.Name())...)
	// Use branchDirectory as working dir if it exists, otherwise fall back to rootDirectory.
	// This allows pre-create hooks to run before the worktree directory is created.
	if _, statErr := os.Stat(branchDirectory); statErr == nil {
		cmd.Dir = branchDirectory
	} else {
		cmd.Dir = rootDirectory
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		ui.Error(fmt.Sprintf("Hook script failed: %v", err))
		return err
	}

	return nil
}

// ExtractShebang extracts the shebang line and returns the interpreter and remaining content
func ExtractShebang(content string) (interpreter string, remaining string) {
	if len(content) < 2 || content[:2] != "#!" {
		return "", content
	}

	// Find end of shebang line
	end := 0
	for i := 2; i < len(content); i++ {
		if content[i] == '\n' {
			end = i
			break
		}
	}

	if end == 0 {
		// No newline found, entire content is shebang
		return content[2:], ""
	}

	interpreter = content[2:end]
	remaining = content[end+1:] // Skip the newline

	return interpreter, remaining
}
