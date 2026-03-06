package daemon

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
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
	d := New("/dev/null", pidPath, "127.0.0.1:0", 0, 0)
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
