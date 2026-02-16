package template

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestProcessTemplates(t *testing.T) {
	tests := []struct {
		name          string
		setupFiles    map[string]fileInfo // relative path -> content/mode
		templateData  TemplateData
		wantFiles     map[string]string // expected output files and content
		wantErr       bool
		skipOnWindows bool // Skip tests that rely on Unix permissions
	}{
		{
			name:       "no files directory - should succeed",
			setupFiles: map[string]fileInfo{
				// Empty - no files directory
			},
			templateData: TemplateData{
				Branch:        "feature/test",
				Directory:     "/path/to/worktree",
				RootDirectory: "/path/to/root",
			},
			wantFiles: map[string]string{},
			wantErr:   false,
		},
		{
			name: "copy non-template file as-is",
			setupFiles: map[string]fileInfo{
				"README.md": {
					content: "# Test Project\nThis is a test.",
					mode:    0644,
				},
			},
			templateData: TemplateData{
				Branch:        "feature/test",
				Directory:     "/path/to/worktree",
				RootDirectory: "/path/to/root",
			},
			wantFiles: map[string]string{
				"README.md": "# Test Project\nThis is a test.",
			},
			wantErr: false,
		},
		{
			name: "process template file and strip .tmpl extension",
			setupFiles: map[string]fileInfo{
				"config.yaml.tmpl": {
					content: "branch: {{.Branch}}\ndirectory: {{.Directory}}",
					mode:    0644,
				},
			},
			templateData: TemplateData{
				Branch:        "feature/auth",
				Directory:     "/workspace/feature-auth",
				RootDirectory: "/workspace",
			},
			wantFiles: map[string]string{
				"config.yaml": "branch: feature/auth\ndirectory: /workspace/feature-auth",
			},
			wantErr: false,
		},
		{
			name: "nested directories",
			setupFiles: map[string]fileInfo{
				"dir1/dir2/file.txt": {
					content: "nested file",
					mode:    0644,
				},
				"dir1/template.env.tmpl": {
					content: "BRANCH={{.Branch}}",
					mode:    0644,
				},
			},
			templateData: TemplateData{
				Branch:        "feature/nested",
				Directory:     "/path/to/worktree",
				RootDirectory: "/path/to/root",
			},
			wantFiles: map[string]string{
				"dir1/dir2/file.txt": "nested file",
				"dir1/template.env":  "BRANCH=feature/nested",
			},
			wantErr: false,
		},
		{
			name: "preserve file permissions - executable",
			setupFiles: map[string]fileInfo{
				"script.sh": {
					content: "#!/bin/bash\necho 'test'",
					mode:    0755,
				},
			},
			templateData: TemplateData{
				Branch:        "feature/test",
				Directory:     "/path/to/worktree",
				RootDirectory: "/path/to/root",
			},
			wantFiles: map[string]string{
				"script.sh": "#!/bin/bash\necho 'test'",
			},
			wantErr:       false,
			skipOnWindows: true,
		},
		{
			name: "preserve permissions on template files",
			setupFiles: map[string]fileInfo{
				"deploy.sh.tmpl": {
					content: "#!/bin/bash\necho '{{.Branch}}'",
					mode:    0755,
				},
			},
			templateData: TemplateData{
				Branch:        "main",
				Directory:     "/path/to/worktree",
				RootDirectory: "/path/to/root",
			},
			wantFiles: map[string]string{
				"deploy.sh": "#!/bin/bash\necho 'main'",
			},
			wantErr:       false,
			skipOnWindows: true,
		},
		{
			name: "multiple files mixed types",
			setupFiles: map[string]fileInfo{
				"static.txt": {
					content: "static content",
					mode:    0644,
				},
				"dynamic.txt.tmpl": {
					content: "Branch: {{.Branch}}",
					mode:    0644,
				},
				"executable.sh": {
					content: "#!/bin/bash\necho 'static'",
					mode:    0755,
				},
				"executable-template.sh.tmpl": {
					content: "#!/bin/bash\necho '{{.Branch}}'",
					mode:    0755,
				},
			},
			templateData: TemplateData{
				Branch:        "develop",
				Directory:     "/workspace/develop",
				RootDirectory: "/workspace",
			},
			wantFiles: map[string]string{
				"static.txt":             "static content",
				"dynamic.txt":            "Branch: develop",
				"executable.sh":          "#!/bin/bash\necho 'static'",
				"executable-template.sh": "#!/bin/bash\necho 'develop'",
			},
			wantErr:       false,
			skipOnWindows: true,
		},
		{
			name: "gomplate functions support",
			setupFiles: map[string]fileInfo{
				"gomplate-test.txt.tmpl": {
					content: "Upper: {{.Branch | strings.ToUpper}}\nLower: {{.Branch | strings.ToLower}}",
					mode:    0644,
				},
			},
			templateData: TemplateData{
				Branch:        "Feature/Auth",
				Directory:     "/workspace/feature-auth",
				RootDirectory: "/workspace",
			},
			wantFiles: map[string]string{
				"gomplate-test.txt": "Upper: FEATURE/AUTH\nLower: feature/auth",
			},
			wantErr: false,
		},
		{
			name: "all template data fields",
			setupFiles: map[string]fileInfo{
				"full-data.txt.tmpl": {
					content: "Branch: {{.Branch}}\nDirectory: {{.Directory}}\nRoot: {{.RootDirectory}}",
					mode:    0644,
				},
			},
			templateData: TemplateData{
				Branch:        "feature/test",
				Directory:     "/workspace/feature-test",
				RootDirectory: "/workspace",
			},
			wantFiles: map[string]string{
				"full-data.txt": "Branch: feature/test\nDirectory: /workspace/feature-test\nRoot: /workspace",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnWindows && runtime.GOOS == "windows" {
				t.Skip("Skipping permission test on Windows")
			}

			// Create temporary directories
			tempDir := t.TempDir()
			filesDir := filepath.Join(tempDir, "files")
			worktreeDir := filepath.Join(tempDir, "worktree")

			// Setup files directory only if there are files to setup
			if len(tt.setupFiles) > 0 {
				if err := os.MkdirAll(filesDir, 0755); err != nil {
					t.Fatalf("failed to create files dir: %v", err)
				}

				// Create test files
				for relPath, info := range tt.setupFiles {
					fullPath := filepath.Join(filesDir, relPath)
					dir := filepath.Dir(fullPath)
					if err := os.MkdirAll(dir, 0755); err != nil {
						t.Fatalf("failed to create directory %s: %v", dir, err)
					}
					if err := os.WriteFile(fullPath, []byte(info.content), info.mode); err != nil {
						t.Fatalf("failed to write file %s: %v", relPath, err)
					}
				}
			}

			// Create worktree directory
			if err := os.MkdirAll(worktreeDir, 0755); err != nil {
				t.Fatalf("failed to create worktree dir: %v", err)
			}

			// Run ProcessTemplates
			err := ProcessTemplates(filesDir, worktreeDir, tt.templateData)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessTemplates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If we expected an error, we're done
			if tt.wantErr {
				return
			}

			// Verify output files
			for expectedPath, expectedContent := range tt.wantFiles {
				fullPath := filepath.Join(worktreeDir, expectedPath)

				// Check file exists
				if _, err := os.Stat(fullPath); os.IsNotExist(err) {
					t.Errorf("expected file %s does not exist", expectedPath)
					continue
				}

				// Check content
				content, err := os.ReadFile(fullPath)
				if err != nil {
					t.Errorf("failed to read file %s: %v", expectedPath, err)
					continue
				}

				if string(content) != expectedContent {
					t.Errorf("file %s content = %q, want %q", expectedPath, string(content), expectedContent)
				}

				// Check permissions (skip on Windows)
				if !tt.skipOnWindows {
					// Find the original file to compare permissions
					var originalMode os.FileMode
					for setupPath, info := range tt.setupFiles {
						// Check if this is the source file (either direct match or .tmpl version)
						if setupPath == expectedPath || setupPath == expectedPath+".tmpl" {
							originalMode = info.mode
							break
						}
					}

					if originalMode != 0 {
						info, err := os.Stat(fullPath)
						if err != nil {
							t.Errorf("failed to stat file %s: %v", expectedPath, err)
							continue
						}

						if info.Mode().Perm() != originalMode.Perm() {
							t.Errorf("file %s mode = %v, want %v", expectedPath, info.Mode().Perm(), originalMode.Perm())
						}
					}
				}
			}

			// Verify no extra files were created
			err = filepath.WalkDir(worktreeDir, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if path == worktreeDir {
					return nil
				}
				if d.IsDir() {
					return nil
				}

				relPath, err := filepath.Rel(worktreeDir, path)
				if err != nil {
					return err
				}

				if _, expected := tt.wantFiles[relPath]; !expected {
					t.Errorf("unexpected file created: %s", relPath)
				}

				return nil
			})
			if err != nil {
				t.Errorf("failed to walk output directory: %v", err)
			}
		})
	}
}

func TestProcessTemplateFile(t *testing.T) {
	tests := []struct {
		name         string
		templateText string
		data         TemplateData
		wantOutput   string
		wantErr      bool
	}{
		{
			name:         "simple template",
			templateText: "Branch: {{.Branch}}",
			data: TemplateData{
				Branch:        "main",
				Directory:     "/path/to/dir",
				RootDirectory: "/path/to/root",
			},
			wantOutput: "Branch: main",
			wantErr:    false,
		},
		{
			name:         "all fields",
			templateText: "{{.Branch}}|{{.Directory}}|{{.RootDirectory}}",
			data: TemplateData{
				Branch:        "feature/test",
				Directory:     "/work/feature-test",
				RootDirectory: "/work",
			},
			wantOutput: "feature/test|/work/feature-test|/work",
			wantErr:    false,
		},
		{
			name:         "gomplate strings functions",
			templateText: "{{.Branch | strings.ToUpper}}",
			data: TemplateData{
				Branch:        "feature/auth",
				Directory:     "/path",
				RootDirectory: "/root",
			},
			wantOutput: "FEATURE/AUTH",
			wantErr:    false,
		},
		{
			name:         "invalid template syntax",
			templateText: "{{.Branch",
			data: TemplateData{
				Branch:        "main",
				Directory:     "/path",
				RootDirectory: "/root",
			},
			wantOutput: "",
			wantErr:    true,
		},
		{
			name:         "empty template",
			templateText: "",
			data: TemplateData{
				Branch:        "main",
				Directory:     "/path",
				RootDirectory: "/root",
			},
			wantOutput: "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary files
			tempDir := t.TempDir()
			templatePath := filepath.Join(tempDir, "template.tmpl")
			outputPath := filepath.Join(tempDir, "output.txt")

			// Write template file
			if err := os.WriteFile(templatePath, []byte(tt.templateText), 0644); err != nil {
				t.Fatalf("failed to write template file: %v", err)
			}

			// Process template
			err := processTemplateFile(templatePath, outputPath, tt.data)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("processTemplateFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Read output
			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read output file: %v", err)
			}

			if string(content) != tt.wantOutput {
				t.Errorf("output = %q, want %q", string(content), tt.wantOutput)
			}

			// Verify permissions preserved
			templateInfo, err := os.Stat(templatePath)
			if err != nil {
				t.Fatalf("failed to stat template: %v", err)
			}

			outputInfo, err := os.Stat(outputPath)
			if err != nil {
				t.Fatalf("failed to stat output: %v", err)
			}

			if outputInfo.Mode() != templateInfo.Mode() {
				t.Errorf("output mode = %v, want %v", outputInfo.Mode(), templateInfo.Mode())
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		mode    os.FileMode
		wantErr bool
	}{
		{
			name:    "regular file",
			content: "test content",
			mode:    0644,
			wantErr: false,
		},
		{
			name:    "executable file",
			content: "#!/bin/bash\necho test",
			mode:    0755,
			wantErr: false,
		},
		{
			name:    "read-only file",
			content: "readonly",
			mode:    0444,
			wantErr: false,
		},
		{
			name:    "empty file",
			content: "",
			mode:    0644,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir := t.TempDir()
			srcPath := filepath.Join(tempDir, "source.txt")
			dstPath := filepath.Join(tempDir, "dest.txt")

			// Create source file
			if err := os.WriteFile(srcPath, []byte(tt.content), tt.mode); err != nil {
				t.Fatalf("failed to create source file: %v", err)
			}

			// Copy file
			err := copyFile(srcPath, dstPath)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("copyFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Verify content
			dstContent, err := os.ReadFile(dstPath)
			if err != nil {
				t.Fatalf("failed to read destination file: %v", err)
			}

			if string(dstContent) != tt.content {
				t.Errorf("destination content = %q, want %q", string(dstContent), tt.content)
			}

			// Verify permissions
			dstInfo, err := os.Stat(dstPath)
			if err != nil {
				t.Fatalf("failed to stat destination file: %v", err)
			}

			if dstInfo.Mode() != tt.mode {
				t.Errorf("destination mode = %v, want %v", dstInfo.Mode(), tt.mode)
			}
		})
	}
}

func TestCopyFile_Errors(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, tempDir string) (src, dst string)
		wantErr bool
	}{
		{
			name: "source file does not exist",
			setup: func(t *testing.T, tempDir string) (src, dst string) {
				return filepath.Join(tempDir, "nonexistent.txt"), filepath.Join(tempDir, "dest.txt")
			},
			wantErr: true,
		},
		{
			name: "destination directory does not exist",
			setup: func(t *testing.T, tempDir string) (src, dst string) {
				srcPath := filepath.Join(tempDir, "source.txt")
				if err := os.WriteFile(srcPath, []byte("test"), 0644); err != nil {
					t.Fatalf("failed to create source file: %v", err)
				}
				return srcPath, filepath.Join(tempDir, "nonexistent", "dest.txt")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			src, dst := tt.setup(t, tempDir)

			err := copyFile(src, dst)

			if (err != nil) != tt.wantErr {
				t.Errorf("copyFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// fileInfo holds test file setup information
type fileInfo struct {
	content string
	mode    os.FileMode
}
