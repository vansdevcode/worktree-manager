package proxy

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/vansdevcode/worktree-manager/internal/routing"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("finding free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port
}

func TestProxyRoundTrip(t *testing.T) {
	// Start a test HTTP backend.
	backendPort := freePort(t)
	backend := &http.Server{
		Addr: fmt.Sprintf("127.0.0.1:%d", backendPort),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("hello from backend"))
		}),
	}
	go func() { _ = backend.ListenAndServe() }()
	defer func() { _ = backend.Close() }()

	// Wait for backend to be ready.
	for i := 0; i < 50; i++ {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", backendPort), 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Set up routing table.
	table := routing.NewTable()
	table.Register("myapp.test", fmt.Sprintf("127.0.0.1:%d", backendPort), nil)

	// Start proxy on high ports.
	httpPort := freePort(t)
	httpsPort := freePort(t)
	srv := New(httpPort, httpsPort)

	if err := srv.Start(table); err != nil {
		t.Fatalf("starting proxy: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	// Give Caddy a moment to provision TLS certificates.
	time.Sleep(2 * time.Second)

	// Make HTTPS request with TLS verification skipped (internal CA not in system trust store during tests).
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // test only
				ServerName:         "myapp.test",
			},
		},
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%d/", httpsPort), nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Host = "myapp.test"

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("making HTTPS request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if string(body) != "hello from backend" {
		t.Errorf("expected body %q, got %q", "hello from backend", string(body))
	}
}

func TestProxyReload(t *testing.T) {
	backendPort := freePort(t)
	backend := &http.Server{
		Addr: fmt.Sprintf("127.0.0.1:%d", backendPort),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("reloaded"))
		}),
	}
	go func() { _ = backend.ListenAndServe() }()
	defer func() { _ = backend.Close() }()

	for i := 0; i < 50; i++ {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", backendPort), 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Start with empty table.
	httpPort := freePort(t)
	httpsPort := freePort(t)
	srv := New(httpPort, httpsPort)

	emptyTable := routing.NewTable()
	if err := srv.Start(emptyTable); err != nil {
		t.Fatalf("starting proxy: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	// Reload with new route.
	table := routing.NewTable()
	table.Register("reloaded.test", fmt.Sprintf("127.0.0.1:%d", backendPort), nil)

	if err := srv.Reload(table); err != nil {
		t.Fatalf("reloading proxy: %v", err)
	}

	time.Sleep(2 * time.Second)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // test only
				ServerName:         "reloaded.test",
			},
		},
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%d/", httpsPort), nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Host = "reloaded.test"

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("making HTTPS request after reload: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}

	if string(body) != "reloaded" {
		t.Errorf("expected body %q, got %q", "reloaded", string(body))
	}
}
