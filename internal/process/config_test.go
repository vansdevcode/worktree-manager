package process

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devtree.toml")

	content := `
[project]
name = "myapp"

[[process]]
name = "web"
cmd = "go run ."
port = 8080
domain = "myapp.test"
dir = "backend"

[[process]]
name = "worker"
cmd = "go run ./worker"
`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Project.Name != "myapp" {
		t.Errorf("project name = %q, want %q", cfg.Project.Name, "myapp")
	}
	if cfg.Project.Domain != "myapp.test" {
		t.Errorf("project domain = %q, want %q", cfg.Project.Domain, "myapp.test")
	}

	if len(cfg.Processes) != 2 {
		t.Fatalf("len(processes) = %d, want 2", len(cfg.Processes))
	}

	web := cfg.Processes[0]
	if web.Name != "web" {
		t.Errorf("process[0].name = %q, want %q", web.Name, "web")
	}
	if web.Dir != filepath.Join(dir, "backend") {
		t.Errorf("process[0].dir = %q, want %q", web.Dir, filepath.Join(dir, "backend"))
	}
	if len(web.Domains) != 1 || web.Domains[0] != "myapp.test" {
		t.Errorf("process[0].domains = %v, want [myapp.test]", web.Domains)
	}

	worker := cfg.Processes[1]
	if worker.Dir != dir {
		t.Errorf("process[1].dir = %q, want %q", worker.Dir, dir)
	}
	if len(worker.Domains) != 0 {
		t.Errorf("process[1].domains = %v, want empty", worker.Domains)
	}
}

func TestLoadConfig_DefaultDomain(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devtree.toml")

	content := `
[project]
name = "api"

[[process]]
name = "server"
cmd = "npm start"
`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Project.Domain != "api.test" {
		t.Errorf("default domain = %q, want %q", cfg.Project.Domain, "api.test")
	}
}

func TestLoadConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "missing project name",
			content: `[[process]]` + "\n" + `name = "web"` + "\n" + `cmd = "go run ."`,
			wantErr: "project.name is required",
		},
		{
			name:    "no processes",
			content: `[project]` + "\n" + `name = "app"`,
			wantErr: "at least one [[process]] is required",
		},
		{
			name: "missing process name",
			content: `[project]
name = "app"
[[process]]
cmd = "go run ."`,
			wantErr: "process[0]: name is required",
		},
		{
			name: "missing cmd",
			content: `[project]
name = "app"
[[process]]
name = "web"`,
			wantErr: `process "web": cmd is required`,
		},
		{
			name: "domain without port or socket",
			content: `[project]
name = "app"
[[process]]
name = "web"
cmd = "go run ."
domain = "app.test"`,
			wantErr: `process "web": port or socket is required when domain is set`,
		},
		{
			name: "port and socket both set",
			content: `[project]
name = "app"
[[process]]
name = "web"
cmd = "go run ."
port = 8080
socket = "/tmp/web.sock"`,
			wantErr: `process "web": port and socket are mutually exclusive`,
		},
		{
			name: "duplicate name",
			content: `[project]
name = "app"
[[process]]
name = "web"
cmd = "go run ."
[[process]]
name = "web"
cmd = "go run ./other"`,
			wantErr: `duplicate name "web"`,
		},
		{
			name: "invalid restart policy",
			content: `[project]
name = "app"
[[process]]
name = "web"
cmd = "go run ."
restart = "sometimes"`,
			wantErr: `invalid restart policy "sometimes"`,
		},
		{
			name: "after references unknown process",
			content: `[project]
name = "app"
[[process]]
name = "web"
cmd = "go run ."
after = ["db"]`,
			wantErr: `after references unknown process "db"`,
		},
		{
			name: "after references self",
			content: `[project]
name = "app"
[[process]]
name = "web"
cmd = "go run ."
after = ["web"]`,
			wantErr: `cannot depend on itself`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "devtree.toml")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := LoadConfig(path)
			if err == nil {
				t.Fatal("expected error")
			}
			if got := err.Error(); !contains(got, tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", got, tt.wantErr)
			}
		})
	}
}

func TestLoadConfig_RestartAndAfter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devtree.toml")

	content := `
[project]
name = "app"

[[process]]
name = "db"
cmd = "mysqld"

[[process]]
name = "seed"
cmd = "mysql-seed"
restart = "never"
after = ["db"]

[[process]]
name = "web"
cmd = "npm start"
restart = "on_failure"
after = ["db"]
`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	seed := cfg.Processes[1]
	if seed.Restart != "never" {
		t.Errorf("seed.restart = %q, want %q", seed.Restart, "never")
	}
	if len(seed.After) != 1 || seed.After[0] != "db" {
		t.Errorf("seed.after = %v, want [db]", seed.After)
	}

	web := cfg.Processes[2]
	if web.Restart != "on_failure" {
		t.Errorf("web.restart = %q, want %q", web.Restart, "on_failure")
	}
}

func TestLoadConfig_JSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devtree.json")

	content := `{
  "project": { "name": "myapp" },
  "process": [
    { "name": "web", "cmd": "go run .", "port": 8080, "domain": "myapp.test" },
    { "name": "worker", "cmd": "go run ./worker" }
  ]
}`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig JSON: %v", err)
	}

	if cfg.Project.Name != "myapp" {
		t.Errorf("project name = %q, want %q", cfg.Project.Name, "myapp")
	}
	if cfg.Project.Domain != "myapp.test" {
		t.Errorf("project domain = %q, want %q", cfg.Project.Domain, "myapp.test")
	}
	if len(cfg.Processes) != 2 {
		t.Fatalf("len(processes) = %d, want 2", len(cfg.Processes))
	}
	if cfg.Processes[0].Port.Int() != 8080 {
		t.Errorf("process[0].port = %d, want 8080", cfg.Processes[0].Port.Int())
	}
	if len(cfg.Processes[1].Domains) != 0 {
		t.Errorf("process[1].domains = %v, want empty", cfg.Processes[1].Domains)
	}
}

func TestLoadConfig_PortEnvVar(t *testing.T) {
	t.Setenv("DEVTREE_TEST_PORT", "3000")

	dir := t.TempDir()
	path := filepath.Join(dir, "devtree.toml")

	content := `
[project]
name = "myapp"

[[process]]
name = "web"
cmd = "npm run dev"
port = "$DEVTREE_TEST_PORT"
domain = "myapp.test"
`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Processes[0].Port.Int() != 3000 {
		t.Errorf("port = %d, want 3000", cfg.Processes[0].Port.Int())
	}
}

func TestLoadConfig_PortEnvVar_JSON(t *testing.T) {
	t.Setenv("DEVTREE_TEST_PORT", "4000")

	dir := t.TempDir()
	path := filepath.Join(dir, "devtree.json")

	content := `{
  "project": { "name": "myapp" },
  "process": [
    { "name": "web", "cmd": "npm run dev", "port": "$DEVTREE_TEST_PORT", "domain": "myapp.test" },
    { "name": "api", "cmd": "go run .", "port": 8080, "domain": "api.test" }
  ]
}`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Processes[0].Port.Int() != 4000 {
		t.Errorf("port = %d, want 4000", cfg.Processes[0].Port.Int())
	}
	if cfg.Processes[1].Port.Int() != 8080 {
		t.Errorf("port = %d, want 8080", cfg.Processes[1].Port.Int())
	}
}

func TestLoadConfig_PortEnvVar_Invalid(t *testing.T) {
	t.Setenv("DEVTREE_TEST_PORT", "not_a_number")

	dir := t.TempDir()
	path := filepath.Join(dir, "devtree.toml")

	content := `
[project]
name = "myapp"

[[process]]
name = "web"
cmd = "npm run dev"
port = "$DEVTREE_TEST_PORT"
`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for non-numeric port env var")
	}
	if !contains(err.Error(), "not a valid integer") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "not a valid integer")
	}
}

func TestLoadConfig_Socket(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devtree.toml")

	content := `
[project]
name = "myapp"

[[process]]
name = "web"
cmd = "go run ."
socket = "/tmp/myapp.sock"
domain = "myapp.test"

[[process]]
name = "worker"
cmd = "go run ./worker"
`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	web := cfg.Processes[0]
	if web.Socket != "/tmp/myapp.sock" {
		t.Errorf("process[0].socket = %q, want %q", web.Socket, "/tmp/myapp.sock")
	}
	if web.Port.Int() != 0 {
		t.Errorf("process[0].port = %d, want 0", web.Port.Int())
	}
	if len(web.Domains) != 1 || web.Domains[0] != "myapp.test" {
		t.Errorf("process[0].domains = %v, want [myapp.test]", web.Domains)
	}
}

func TestLoadConfig_MultipleDomains(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devtree.toml")

	content := `
[project]
name = "myapp"

[[process]]
name = "web"
cmd = "go run ."
port = 8080
domain = ["myapp.test", "admin.myapp.test", "api.myapp.test"]
`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	web := cfg.Processes[0]
	want := []string{"myapp.test", "admin.myapp.test", "api.myapp.test"}
	if len(web.Domains) != len(want) {
		t.Fatalf("domains = %v, want %v", web.Domains, want)
	}
	for i, d := range want {
		if web.Domains[i] != d {
			t.Errorf("domains[%d] = %q, want %q", i, web.Domains[i], d)
		}
	}
}

func TestLoadConfig_MultipleDomains_JSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devtree.json")

	content := `{
  "project": { "name": "myapp" },
  "process": [
    { "name": "web", "cmd": "go run .", "port": 8080, "domain": ["myapp.test", "admin.myapp.test"] }
  ]
}`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	web := cfg.Processes[0]
	if len(web.Domains) != 2 {
		t.Fatalf("domains = %v, want 2 entries", web.Domains)
	}
	if web.Domains[0] != "myapp.test" || web.Domains[1] != "admin.myapp.test" {
		t.Errorf("domains = %v, want [myapp.test admin.myapp.test]", web.Domains)
	}
}

func TestFindConfig_PrefersToml(t *testing.T) {
	root := t.TempDir()
	tomlPath := filepath.Join(root, "devtree.toml")
	jsonPath := filepath.Join(root, "devtree.json")
	_ = os.WriteFile(tomlPath, []byte("[project]\nname=\"x\"\n"), 0o644)
	_ = os.WriteFile(jsonPath, []byte(`{"project":{"name":"x"}}`), 0o644)

	found, err := FindConfig(root)
	if err != nil {
		t.Fatalf("FindConfig: %v", err)
	}
	if found != tomlPath {
		t.Errorf("found = %q, want %q (should prefer .toml)", found, tomlPath)
	}
}

func TestFindConfig_FallsBackToJSON(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a")
	_ = os.MkdirAll(sub, 0o755)
	jsonPath := filepath.Join(root, "devtree.json")
	_ = os.WriteFile(jsonPath, []byte(`{"project":{"name":"x"}}`), 0o644)

	found, err := FindConfig(sub)
	if err != nil {
		t.Fatalf("FindConfig: %v", err)
	}
	if found != jsonPath {
		t.Errorf("found = %q, want %q", found, jsonPath)
	}
}

func TestFindConfig(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "devtree.toml")
	if err := os.WriteFile(configPath, []byte("[project]\nname=\"x\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := FindConfig(sub)
	if err != nil {
		t.Fatalf("FindConfig: %v", err)
	}
	if found != configPath {
		t.Errorf("found = %q, want %q", found, configPath)
	}
}

func TestFindConfig_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := FindConfig(dir)
	if err == nil {
		t.Fatal("expected error")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
