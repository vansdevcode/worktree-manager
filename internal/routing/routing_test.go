package routing

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTable(t *testing.T) {
	table := NewTable()
	if table.Sites == nil {
		t.Fatal("Sites map should be initialized")
	}
	if len(table.Sites) != 0 {
		t.Fatalf("expected 0 sites, got %d", len(table.Sites))
	}
}

func TestRegister(t *testing.T) {
	table := NewTable()
	meta := map[string]string{"stack": "go:1.23"}
	table.Register("myapp.test", "localhost:8080", meta)

	site, ok := table.Get("myapp.test")
	if !ok {
		t.Fatal("expected site to exist")
	}
	if site.Upstream != "localhost:8080" {
		t.Fatalf("expected upstream localhost:8080, got %s", site.Upstream)
	}
	if site.Meta["stack"] != "go:1.23" {
		t.Fatalf("expected meta stack go:1.23, got %s", site.Meta["stack"])
	}
}

func TestRegisterUpdate(t *testing.T) {
	table := NewTable()
	table.Register("myapp.test", "localhost:8080", nil)
	table.Register("myapp.test", "localhost:9090", map[string]string{"updated": "true"})

	site, ok := table.Get("myapp.test")
	if !ok {
		t.Fatal("expected site to exist")
	}
	if site.Upstream != "localhost:9090" {
		t.Fatalf("expected updated upstream localhost:9090, got %s", site.Upstream)
	}
	if len(table.Sites) != 1 {
		t.Fatalf("expected 1 site (no duplicates), got %d", len(table.Sites))
	}
}

func TestUnregister(t *testing.T) {
	table := NewTable()
	table.Register("myapp.test", "localhost:8080", nil)

	if err := table.Unregister("myapp.test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := table.Get("myapp.test"); ok {
		t.Fatal("expected site to be removed")
	}
}

func TestUnregisterNonExistent(t *testing.T) {
	table := NewTable()
	if err := table.Unregister("missing.test"); err == nil {
		t.Fatal("expected error for non-existent domain")
	}
}

func TestList(t *testing.T) {
	table := NewTable()
	table.Register("beta.test", "localhost:3000", nil)
	table.Register("alpha.test", "localhost:8080", nil)
	table.Register("gamma.test", "localhost:9090", nil)

	sites := table.List()
	if len(sites) != 3 {
		t.Fatalf("expected 3 sites, got %d", len(sites))
	}
	if sites[0].Domain != "alpha.test" {
		t.Fatalf("expected first site alpha.test, got %s", sites[0].Domain)
	}
	if sites[1].Domain != "beta.test" {
		t.Fatalf("expected second site beta.test, got %s", sites[1].Domain)
	}
	if sites[2].Domain != "gamma.test" {
		t.Fatalf("expected third site gamma.test, got %s", sites[2].Domain)
	}
}

func TestListEmpty(t *testing.T) {
	table := NewTable()
	sites := table.List()
	if len(sites) != 0 {
		t.Fatalf("expected 0 sites, got %d", len(sites))
	}
}

func TestLoadNonExistent(t *testing.T) {
	table, err := Load("/tmp/devtree-test-nonexistent/routes.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(table.Sites) != 0 {
		t.Fatalf("expected empty table, got %d sites", len(table.Sites))
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "routes.yaml")

	table := NewTable()
	table.Register("myapp.test", "localhost:8080", map[string]string{"stack": "go:1.23"})
	table.Register("frontend.test", "localhost:3000", map[string]string{"stack": "node:22", "framework": "vite"})

	if err := Save(path, table); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	if len(loaded.Sites) != 2 {
		t.Fatalf("expected 2 sites, got %d", len(loaded.Sites))
	}

	site, ok := loaded.Get("myapp.test")
	if !ok {
		t.Fatal("expected myapp.test to exist")
	}
	if site.Upstream != "localhost:8080" {
		t.Fatalf("expected upstream localhost:8080, got %s", site.Upstream)
	}
	if site.Meta["stack"] != "go:1.23" {
		t.Fatalf("expected meta stack go:1.23, got %s", site.Meta["stack"])
	}

	site2, ok := loaded.Get("frontend.test")
	if !ok {
		t.Fatal("expected frontend.test to exist")
	}
	if site2.Meta["framework"] != "vite" {
		t.Fatalf("expected meta framework vite, got %s", site2.Meta["framework"])
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "routes.yaml")

	table := NewTable()
	table.Register("myapp.test", "localhost:8080", nil)

	if err := Save(path, table); err != nil {
		t.Fatalf("save error: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("expected file to exist")
	}
}

func TestSaveAtomicNoCorruption(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "routes.yaml")

	// Save initial data
	table := NewTable()
	table.Register("myapp.test", "localhost:8080", nil)
	if err := Save(path, table); err != nil {
		t.Fatalf("save error: %v", err)
	}

	// Save updated data
	table.Register("other.test", "localhost:9090", nil)
	if err := Save(path, table); err != nil {
		t.Fatalf("save error: %v", err)
	}

	// Verify no temp files remain
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir error: %v", err)
	}
	for _, e := range entries {
		if e.Name() != "routes.yaml" {
			t.Fatalf("unexpected file remaining: %s", e.Name())
		}
	}

	// Verify data integrity
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if len(loaded.Sites) != 2 {
		t.Fatalf("expected 2 sites after atomic save, got %d", len(loaded.Sites))
	}
}

func TestLoadEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "routes.yaml")

	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatalf("write error: %v", err)
	}

	table, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(table.Sites) != 0 {
		t.Fatalf("expected empty table, got %d sites", len(table.Sites))
	}
}
