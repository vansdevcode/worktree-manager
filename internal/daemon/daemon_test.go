package daemon

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestIsRunning_NoPIDFile(t *testing.T) {
	_, running := IsRunning("/tmp/nonexistent-devtree-test.pid")
	if running {
		t.Error("expected not running when PID file doesn't exist")
	}
}

func TestIsRunning_StalePID(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "test.pid")
	// Write a PID that almost certainly doesn't exist.
	_ = os.WriteFile(pidFile, []byte("999999999"), 0o644)
	_, running := IsRunning(pidFile)
	if running {
		t.Error("expected not running for stale PID")
	}
}

func TestIsRunning_CurrentProcess(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "test.pid")
	pid := os.Getpid()
	_ = os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0o644)
	gotPID, running := IsRunning(pidFile)
	if !running {
		t.Error("expected running for current process PID")
	}
	if gotPID != pid {
		t.Errorf("expected PID %d, got %d", pid, gotPID)
	}
}

func TestStopDaemon_NotRunning(t *testing.T) {
	err := StopDaemon("/tmp/nonexistent-devtree-test.pid")
	if err == nil {
		t.Error("expected error when daemon is not running")
	}
}

func TestWritePID(t *testing.T) {
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "sub", "test.pid")
	d := New("/dev/null", pidPath, "", "127.0.0.1:0", 0, 0)
	if err := d.writePID(); err != nil {
		t.Fatalf("writePID: %v", err)
	}
	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("reading PID file: %v", err)
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		t.Fatalf("parsing PID: %v", err)
	}
	if pid != os.Getpid() {
		t.Errorf("expected PID %d, got %d", os.Getpid(), pid)
	}
}

func TestStartSocket(t *testing.T) {
	sockPath := tempSockPath(t)
	d := New("/dev/null", "", sockPath, "127.0.0.1:0", 0, 0)

	listener, err := d.startSocket()
	if err != nil {
		t.Fatalf("startSocket: %v", err)
	}
	defer func() { _ = listener.Close() }()

	// Verify socket file exists.
	if _, err := os.Stat(sockPath); err != nil {
		t.Fatalf("socket file not created: %v", err)
	}

	// Connect and send a message (best-effort — no proxy to reload, but should not panic).
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("connecting to socket: %v", err)
	}
	_, err = conn.Write([]byte("reload\n"))
	if err != nil {
		t.Fatalf("writing to socket: %v", err)
	}
	_ = conn.Close()

	// Give the goroutine a moment to process.
	time.Sleep(50 * time.Millisecond)
}

func tempSockPath(t *testing.T) string {
	t.Helper()
	// Unix socket paths are limited to ~104 chars on macOS; use /tmp directly.
	f, err := os.CreateTemp("/tmp", "devtree-test-*.sock")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	_ = f.Close()
	_ = os.Remove(path)
	t.Cleanup(func() { _ = os.Remove(path) })
	return path
}

func TestStartSocket_RemovesStaleSocket(t *testing.T) {
	sockPath := tempSockPath(t)

	// Create a stale socket file.
	_ = os.WriteFile(sockPath, []byte("stale"), 0o644)

	d := New("/dev/null", "", sockPath, "127.0.0.1:0", 0, 0)
	listener, err := d.startSocket()
	if err != nil {
		t.Fatalf("startSocket should remove stale socket: %v", err)
	}
	_ = listener.Close()
}

func TestStartSocket_ReloadReadsRoutes(t *testing.T) {
	sockPath := tempSockPath(t)
	dir := t.TempDir()
	routesPath := filepath.Join(dir, "routes.yaml")

	// Write a valid but empty routes file.
	_ = os.WriteFile(routesPath, []byte("sites: {}\n"), 0o644)

	d := New(routesPath, "", sockPath, "127.0.0.1:0", 0, 0)

	listener, err := d.startSocket()
	if err != nil {
		t.Fatalf("startSocket: %v", err)
	}
	defer func() { _ = listener.Close() }()

	// Send reload — should not error (no proxy, but reload() logs instead of failing).
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("connecting to socket: %v", err)
	}
	_, _ = conn.Write([]byte("reload\n"))
	_ = conn.Close()

	time.Sleep(50 * time.Millisecond)
}
