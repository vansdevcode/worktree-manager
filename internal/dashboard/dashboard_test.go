package dashboard

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/vansdevcode/worktree-manager/internal/routing"
)

func TestDashboardServesHTML(t *testing.T) {
	dir := t.TempDir()
	routesPath := filepath.Join(dir, "routes.yaml")

	srv := New(routesPath)
	port, err := srv.Start()
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	resp, err := http.Get(addr(port) + "/index.html")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Fatalf("content-type: got %q, want text/html", ct)
	}
}

func TestAPIStatusEmpty(t *testing.T) {
	dir := t.TempDir()
	routesPath := filepath.Join(dir, "routes.yaml")

	srv := New(routesPath)
	port, err := srv.Start()
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	resp, err := http.Get(addr(port) + "/api/status")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(status.Sites) != 0 {
		t.Fatalf("sites: got %d, want 0", len(status.Sites))
	}
}

func TestAPIStatusWithRoutes(t *testing.T) {
	dir := t.TempDir()
	routesPath := filepath.Join(dir, "routes.yaml")

	// Start a test TCP server so health check passes.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()
	tcpAddr := ln.Addr().String()

	// Register routes.
	table := routing.NewTable()
	table.Register("myapp.test", tcpAddr, map[string]string{"stack": "go"})
	table.Register("dead.test", "localhost:1", nil) // not reachable
	if err := routing.Save(routesPath, table); err != nil {
		t.Fatalf("save: %v", err)
	}

	srv := New(routesPath)
	port, err := srv.Start()
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	resp, err := http.Get(addr(port) + "/api/status")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(status.Sites) != 2 {
		t.Fatalf("sites: got %d, want 2", len(status.Sites))
	}

	// Sites are sorted alphabetically.
	if status.Sites[0].Domain != "dead.test" {
		t.Fatalf("first domain: got %q, want dead.test", status.Sites[0].Domain)
	}
	if status.Sites[0].Healthy {
		t.Fatal("dead.test should be unhealthy")
	}
	if status.Sites[1].Domain != "myapp.test" {
		t.Fatalf("second domain: got %q, want myapp.test", status.Sites[1].Domain)
	}
	if !status.Sites[1].Healthy {
		t.Fatal("myapp.test should be healthy")
	}
	if status.Sites[1].Meta["stack"] != "go" {
		t.Fatalf("meta: got %q, want go", status.Sites[1].Meta["stack"])
	}
}

func TestCheckHealth(t *testing.T) {
	// Healthy: listen on a port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	if !checkHealth(ln.Addr().String()) {
		t.Fatal("expected healthy for open port")
	}
	if checkHealth("localhost:1") {
		t.Fatal("expected unhealthy for closed port")
	}
	if checkHealth("nocolon") {
		t.Fatal("expected unhealthy for bad format")
	}
}

func addr(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}
