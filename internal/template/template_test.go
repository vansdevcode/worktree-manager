package template

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestProcessFiles(t *testing.T) {
	tests := []struct {
		name          string
		sourceFiles   map[string]fileInfo // files to create in sourceBaseDir
		fileSpecs     map[string]FileSpec // config [files] entries
		templateData  TemplateData
		wantFiles     map[string]string // expected output files and content
		wantErr       bool
		skipOnWindows bool
	}{
		{
			name:       "empty file specs",
			fileSpecs:  map[string]FileSpec{},
			wantFiles:  map[string]string{},
			templateData: TemplateData{
				Branch: "feature/test",
			},
		},
		{
			name: "process template file",
			sourceFiles: map[string]fileInfo{
				".wtm/env.tmpl": {
					content: "BRANCH={{.Branch}}\nDIR={{.Directory}}",
					mode:    0644,
				},
			},
			fileSpecs: map[string]FileSpec{
				".env": {Template: ".wtm/env.tmpl"},
			},
			templateData: TemplateData{
				Branch:        "feature/auth",
				Directory:     "/workspace/feature-auth",
				RootDirectory: "/workspace",
			},
			wantFiles: map[string]string{
				".env": "BRANCH=feature/auth\nDIR=/workspace/feature-auth",
			},
		},
		{
			name: "copy file as-is",
			sourceFiles: map[string]fileInfo{
				".wtm/init.sql": {
					content: "CREATE TABLE users (id INT);",
					mode:    0644,
				},
			},
			fileSpecs: map[string]FileSpec{
				"init.sql": {Copy: ".wtm/init.sql"},
			},
			templateData: TemplateData{
				Branch: "main",
			},
			wantFiles: map[string]string{
				"init.sql": "CREATE TABLE users (id INT);",
			},
		},
		{
			name: "nested output path",
			sourceFiles: map[string]fileInfo{
				".wtm/local.yml.tmpl": {
					content: "branch: {{.Branch}}",
					mode:    0644,
				},
			},
			fileSpecs: map[string]FileSpec{
				"config/local.yml": {Template: ".wtm/local.yml.tmpl"},
			},
			templateData: TemplateData{
				Branch: "develop",
			},
			wantFiles: map[string]string{
				"config/local.yml": "branch: develop",
			},
		},
		{
			name: "gomplate functions",
			sourceFiles: map[string]fileInfo{
				".wtm/test.tmpl": {
					content: "{{.Branch | strings.ToUpper}}",
					mode:    0644,
				},
			},
			fileSpecs: map[string]FileSpec{
				"test.txt": {Template: ".wtm/test.tmpl"},
			},
			templateData: TemplateData{
				Branch: "feature/auth",
			},
			wantFiles: map[string]string{
				"test.txt": "FEATURE/AUTH",
			},
		},
		{
			name: "custom variables in template",
			sourceFiles: map[string]fileInfo{
				".wtm/env.tmpl": {
					content: `TICKET={{ index .Vars "ticket" }}
DB=app_{{ index .Vars "env" }}`,
					mode: 0644,
				},
			},
			fileSpecs: map[string]FileSpec{
				"config.env": {Template: ".wtm/env.tmpl"},
			},
			templateData: TemplateData{
				Branch: "feat/PROJ-123",
				Vars:   map[string]string{"ticket": "PROJ-123", "env": "staging"},
			},
			wantFiles: map[string]string{
				"config.env": "TICKET=PROJ-123\nDB=app_staging",
			},
		},
		{
			name: "preserve permissions on template",
			sourceFiles: map[string]fileInfo{
				".wtm/deploy.sh.tmpl": {
					content: "#!/bin/bash\necho '{{.Branch}}'",
					mode:    0755,
				},
			},
			fileSpecs: map[string]FileSpec{
				"deploy.sh": {Template: ".wtm/deploy.sh.tmpl"},
			},
			templateData: TemplateData{
				Branch: "main",
			},
			wantFiles: map[string]string{
				"deploy.sh": "#!/bin/bash\necho 'main'",
			},
			skipOnWindows: true,
		},
		{
			name: "multiple files mixed types",
			sourceFiles: map[string]fileInfo{
				".wtm/env.tmpl": {
					content: "BRANCH={{.Branch}}",
					mode:    0644,
				},
				".wtm/data.sql": {
					content: "INSERT INTO t VALUES (1);",
					mode:    0644,
				},
			},
			fileSpecs: map[string]FileSpec{
				".env":     {Template: ".wtm/env.tmpl"},
				"data.sql": {Copy: ".wtm/data.sql"},
			},
			templateData: TemplateData{
				Branch: "develop",
			},
			wantFiles: map[string]string{
				".env":     "BRANCH=develop",
				"data.sql": "INSERT INTO t VALUES (1);",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnWindows && runtime.GOOS == "windows" {
				t.Skip("Skipping permission test on Windows")
			}

			tempDir := t.TempDir()
			sourceBaseDir := filepath.Join(tempDir, "source")
			worktreeDir := filepath.Join(tempDir, "worktree")

			// Create source files
			for relPath, info := range tt.sourceFiles {
				fullPath := filepath.Join(sourceBaseDir, relPath)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatalf("failed to create directory: %v", err)
				}
				if err := os.WriteFile(fullPath, []byte(info.content), info.mode); err != nil {
					t.Fatalf("failed to write file %s: %v", relPath, err)
				}
			}

			// Create worktree directory
			if err := os.MkdirAll(worktreeDir, 0755); err != nil {
				t.Fatalf("failed to create worktree dir: %v", err)
			}

			err := ProcessFiles(tt.fileSpecs, sourceBaseDir, worktreeDir, tt.templateData)

			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Verify output files
			for expectedPath, expectedContent := range tt.wantFiles {
				fullPath := filepath.Join(worktreeDir, expectedPath)

				if _, err := os.Stat(fullPath); os.IsNotExist(err) {
					t.Errorf("expected file %s does not exist", expectedPath)
					continue
				}

				content, err := os.ReadFile(fullPath)
				if err != nil {
					t.Errorf("failed to read file %s: %v", expectedPath, err)
					continue
				}

				if string(content) != expectedContent {
					t.Errorf("file %s content = %q, want %q", expectedPath, string(content), expectedContent)
				}
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
		},
		{
			name:         "custom vars with index",
			templateText: `{{ index .Vars "myKey" }}`,
			data: TemplateData{
				Branch:        "main",
				Directory:     "/path",
				RootDirectory: "/root",
				Vars:          map[string]string{"myKey": "myValue"},
			},
			wantOutput: "myValue",
		},
		{
			name:         "invalid template syntax",
			templateText: "{{.Branch",
			data: TemplateData{
				Branch:        "main",
				Directory:     "/path",
				RootDirectory: "/root",
			},
			wantErr: true,
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			templatePath := filepath.Join(tempDir, "template.tmpl")
			outputPath := filepath.Join(tempDir, "output.txt")

			if err := os.WriteFile(templatePath, []byte(tt.templateText), 0644); err != nil {
				t.Fatalf("failed to write template file: %v", err)
			}

			err := processTemplateFile(templatePath, outputPath, tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("processTemplateFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read output file: %v", err)
			}

			if string(content) != tt.wantOutput {
				t.Errorf("output = %q, want %q", string(content), tt.wantOutput)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		mode    os.FileMode
	}{
		{"regular file", "test content", 0644},
		{"executable file", "#!/bin/bash\necho test", 0755},
		{"read-only file", "readonly", 0444},
		{"empty file", "", 0644},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			srcPath := filepath.Join(tempDir, "source.txt")
			dstPath := filepath.Join(tempDir, "dest.txt")

			if err := os.WriteFile(srcPath, []byte(tt.content), tt.mode); err != nil {
				t.Fatalf("failed to create source file: %v", err)
			}

			if err := copyFile(srcPath, dstPath); err != nil {
				t.Fatalf("copyFile() error: %v", err)
			}

			dstContent, err := os.ReadFile(dstPath)
			if err != nil {
				t.Fatalf("failed to read destination file: %v", err)
			}

			if string(dstContent) != tt.content {
				t.Errorf("destination content = %q, want %q", string(dstContent), tt.content)
			}

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
		name  string
		setup func(t *testing.T, tempDir string) (src, dst string)
	}{
		{
			name: "source file does not exist",
			setup: func(t *testing.T, tempDir string) (src, dst string) {
				return filepath.Join(tempDir, "nonexistent.txt"), filepath.Join(tempDir, "dest.txt")
			},
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			src, dst := tt.setup(t, tempDir)

			err := copyFile(src, dst)
			if err == nil {
				t.Error("copyFile() expected error, got nil")
			}
		})
	}
}

// fileInfo holds test file setup information
type fileInfo struct {
	content string
	mode    os.FileMode
}
