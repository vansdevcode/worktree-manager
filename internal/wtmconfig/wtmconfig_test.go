package wtmconfig

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad_BothFiles(t *testing.T) {
	rootDir := t.TempDir()
	worktreeDir := filepath.Join(rootDir, "my-branch")
	if err := os.MkdirAll(worktreeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write .wtm.toml in worktree
	wtmToml := `
[hooks]
post-create = "npm install"
pre-delete = "echo cleanup"

[files]
".env" = { template = ".wtm/env.tmpl" }

[vars]
db_host = "localhost"
db_port = "5432"
`
	if err := os.WriteFile(filepath.Join(worktreeDir, ".wtm.toml"), []byte(wtmToml), 0644); err != nil {
		t.Fatal(err)
	}

	// Write .wtm.local.toml in root
	localToml := `
[vars]
db_host = "remote-db.internal"

[hooks]
post-create = "npm install && npm run setup:local"
`
	if err := os.WriteFile(filepath.Join(rootDir, ".wtm.local.toml"), []byte(localToml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(worktreeDir, rootDir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Local overrides base for post-create
	if cfg.Hooks["post-create"] != "npm install && npm run setup:local" {
		t.Errorf("Hooks[post-create] = %q, want local override", cfg.Hooks["post-create"])
	}
	// Base value preserved when not overridden
	if cfg.Hooks["pre-delete"] != "echo cleanup" {
		t.Errorf("Hooks[pre-delete] = %q, want base value", cfg.Hooks["pre-delete"])
	}
	// Local overrides base for db_host
	if cfg.Vars["db_host"] != "remote-db.internal" {
		t.Errorf("Vars[db_host] = %q, want local override", cfg.Vars["db_host"])
	}
	// Base value preserved
	if cfg.Vars["db_port"] != "5432" {
		t.Errorf("Vars[db_port] = %q, want base value", cfg.Vars["db_port"])
	}
	// Files from base preserved
	if cfg.Files[".env"].Template != ".wtm/env.tmpl" {
		t.Errorf("Files[.env] = %+v, want template from base", cfg.Files[".env"])
	}
}

func TestLoad_OnlyBase(t *testing.T) {
	rootDir := t.TempDir()
	worktreeDir := filepath.Join(rootDir, "branch")
	if err := os.MkdirAll(worktreeDir, 0755); err != nil {
		t.Fatal(err)
	}

	wtmToml := `
[vars]
key = "value"
`
	if err := os.WriteFile(filepath.Join(worktreeDir, ".wtm.toml"), []byte(wtmToml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(worktreeDir, rootDir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Vars["key"] != "value" {
		t.Errorf("Vars[key] = %q, want %q", cfg.Vars["key"], "value")
	}
}

func TestLoad_OnlyLocal(t *testing.T) {
	rootDir := t.TempDir()
	worktreeDir := filepath.Join(rootDir, "branch")
	if err := os.MkdirAll(worktreeDir, 0755); err != nil {
		t.Fatal(err)
	}

	localToml := `
[hooks]
post-create = "make setup"
`
	if err := os.WriteFile(filepath.Join(rootDir, ".wtm.local.toml"), []byte(localToml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(worktreeDir, rootDir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Hooks["post-create"] != "make setup" {
		t.Errorf("Hooks[post-create] = %q, want %q", cfg.Hooks["post-create"], "make setup")
	}
}

func TestLoad_NeitherFile(t *testing.T) {
	rootDir := t.TempDir()
	worktreeDir := filepath.Join(rootDir, "branch")
	if err := os.MkdirAll(worktreeDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(worktreeDir, rootDir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Hooks != nil {
		t.Errorf("Hooks = %v, want nil", cfg.Hooks)
	}
	if cfg.Files != nil {
		t.Errorf("Files = %v, want nil", cfg.Files)
	}
	if cfg.Vars != nil {
		t.Errorf("Vars = %v, want nil", cfg.Vars)
	}
}

func TestLoad_InvalidToml(t *testing.T) {
	rootDir := t.TempDir()
	worktreeDir := filepath.Join(rootDir, "branch")
	if err := os.MkdirAll(worktreeDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(worktreeDir, ".wtm.toml"), []byte("invalid [[["), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(worktreeDir, rootDir)
	if err == nil {
		t.Error("Load() expected error for invalid TOML, got nil")
	}
}

func TestMergeMaps(t *testing.T) {
	tests := []struct {
		name  string
		base  map[string]string
		local map[string]string
		want  map[string]string
	}{
		{"both nil", nil, nil, nil},
		{"base only", map[string]string{"a": "1"}, nil, map[string]string{"a": "1"}},
		{"local only", nil, map[string]string{"b": "2"}, map[string]string{"b": "2"}},
		{"local wins", map[string]string{"a": "1"}, map[string]string{"a": "2"}, map[string]string{"a": "2"}},
		{"combined", map[string]string{"a": "1"}, map[string]string{"b": "2"}, map[string]string{"a": "1", "b": "2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeMaps(tt.base, tt.local)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeMaps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoad_FileSpec(t *testing.T) {
	rootDir := t.TempDir()
	worktreeDir := filepath.Join(rootDir, "branch")
	if err := os.MkdirAll(worktreeDir, 0755); err != nil {
		t.Fatal(err)
	}

	wtmToml := `
[files]
".env" = { template = ".wtm/env.tmpl" }
"config/local.yml" = { template = ".wtm/local.yml.tmpl" }
"init.sql" = { copy = ".wtm/init.sql" }
`
	if err := os.WriteFile(filepath.Join(worktreeDir, ".wtm.toml"), []byte(wtmToml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(worktreeDir, rootDir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(cfg.Files) != 3 {
		t.Fatalf("Files count = %d, want 3", len(cfg.Files))
	}
	if cfg.Files[".env"].Template != ".wtm/env.tmpl" {
		t.Errorf("Files[.env] = %+v", cfg.Files[".env"])
	}
	if cfg.Files["init.sql"].Copy != ".wtm/init.sql" {
		t.Errorf("Files[init.sql] = %+v", cfg.Files["init.sql"])
	}
}
