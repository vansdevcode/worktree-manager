package template

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/hairyhenderson/gomplate/v4"
)

// TemplateData contains variables available in templates
type TemplateData struct {
	Branch        string            // Branch name (e.g., "feature/user-auth")
	Directory     string            // Absolute path to worktree directory
	RootDirectory string            // Absolute path to repository root
	Vars          map[string]string // User-defined custom variables
}

// FileSpec describes a file to be created in the worktree.
type FileSpec struct {
	Template string // path relative to the worktree root (source of .wtm.toml)
	Copy     string // path relative to the worktree root
}

// ProcessFiles processes file specs from .wtm.toml config.
// For each entry, the source path (template or copy) is resolved relative to sourceBaseDir
// (typically the worktree where .wtm.toml lives), and the output is placed in worktreeDir.
func ProcessFiles(files map[string]FileSpec, sourceBaseDir, worktreeDir string, data TemplateData) error {
	for outputRelPath, spec := range files {
		outputPath := filepath.Join(worktreeDir, outputRelPath)

		// Ensure output directory exists
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", outputRelPath, err)
		}

		if spec.Template != "" {
			sourcePath := filepath.Join(sourceBaseDir, spec.Template)
			if err := processTemplateFile(sourcePath, outputPath, data); err != nil {
				return fmt.Errorf("failed to process template for %s: %w", outputRelPath, err)
			}
		} else if spec.Copy != "" {
			sourcePath := filepath.Join(sourceBaseDir, spec.Copy)
			if err := copyFile(sourcePath, outputPath); err != nil {
				return fmt.Errorf("failed to copy file for %s: %w", outputRelPath, err)
			}
		}
	}
	return nil
}

// copyFile copies a file from src to dst, preserving permissions
func copyFile(src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if err := os.WriteFile(dst, content, info.Mode()); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// processTemplateFile reads a template file, processes it with gomplate functions, and writes the output
func processTemplateFile(templatePath, outputPath string, templateData TemplateData) error {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	ctx := context.Background()
	funcMap := gomplate.CreateFuncs(ctx)

	tmpl, err := template.New(filepath.Base(templatePath)).Funcs(funcMap).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var outputBuffer bytes.Buffer
	if err := tmpl.Execute(&outputBuffer, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	info, err := os.Stat(templatePath)
	if err != nil {
		return fmt.Errorf("failed to stat template file: %w", err)
	}

	if err := os.WriteFile(outputPath, outputBuffer.Bytes(), info.Mode()); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}
