package template

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	"github.com/hairyhenderson/gomplate/v4"
)

// TemplateData contains variables available in templates
type TemplateData struct {
	Branch        string // Branch name (e.g., "feature/user-auth")
	Directory     string // Absolute path to worktree directory
	RootDirectory string // Absolute path to repository root
}

// ProcessTemplates processes all files in .worktree/files/
// Files ending with .tmpl are processed as templates and saved without the .tmpl extension
// Other files are copied as-is
func ProcessTemplates(filesDir, worktreeDir string, data TemplateData) error {
	// Check if files directory exists
	if _, err := os.Stat(filesDir); os.IsNotExist(err) {
		// No files directory, nothing to do
		return nil
	}

	// Walk through all files in files directory
	return filepath.WalkDir(filesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the files directory itself
		if path == filesDir {
			return nil
		}

		// Get relative path from files directory
		relPath, err := filepath.Rel(filesDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Determine output path
		outputPath := filepath.Join(worktreeDir, relPath)

		if d.IsDir() {
			// Create directory
			if err := os.MkdirAll(outputPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", outputPath, err)
			}
			return nil
		}

		// Check if file is a template (ends with .tmpl)
		if filepath.Ext(path) == ".tmpl" {
			// Process as template and remove .tmpl extension from output
			outputPath = outputPath[:len(outputPath)-5] // Remove ".tmpl" extension
			if err := processTemplateFile(path, outputPath, data); err != nil {
				return fmt.Errorf("failed to process template %s: %w", relPath, err)
			}
		} else {
			// Copy file as-is
			if err := copyFile(path, outputPath); err != nil {
				return fmt.Errorf("failed to copy file %s: %w", relPath, err)
			}
		}

		return nil
	})
}

// copyFile copies a file from src to dst, preserving permissions
func copyFile(src, dst string) error {
	// Read source file
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Get source file permissions
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Write destination file with same permissions
	if err := os.WriteFile(dst, content, info.Mode()); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// processTemplateFile reads a template file, processes it with gomplate functions, and writes the output
func processTemplateFile(templatePath, outputPath string, templateData TemplateData) error {
	// Read template content
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	// Create a context for gomplate functions
	ctx := context.Background()

	// Get gomplate's function map
	funcMap := gomplate.CreateFuncs(ctx)

	// Create template with gomplate functions
	tmpl, err := template.New(filepath.Base(templatePath)).Funcs(funcMap).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create output buffer
	var outputBuffer bytes.Buffer

	// Execute template with data
	if err := tmpl.Execute(&outputBuffer, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Preserve file permissions from template
	info, err := os.Stat(templatePath)
	if err != nil {
		return fmt.Errorf("failed to stat template file: %w", err)
	}

	// Write output file with the same permissions as the template
	if err := os.WriteFile(outputPath, outputBuffer.Bytes(), info.Mode()); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}
