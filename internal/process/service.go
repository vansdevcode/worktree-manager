package process

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/thejerf/suture/v4"
)

// Service wraps an OS process as a suture.Service.
type Service struct {
	Config   ProcessConfig
	LogFile  string // path to the log file for stdout/stderr
	OnUpdate func(state State)
	// WaitReady is called before starting the process. It blocks until
	// all dependencies listed in Config.After are running.
	WaitReady func(ctx context.Context) error
}

// State represents the current state of a process.
type State struct {
	Name     string   `json:"name"`
	Cmd      string   `json:"cmd"`
	Port     int      `json:"port"`
	Socket   string   `json:"socket,omitempty"`
	Domains  []string `json:"domains"`
	Status   string   `json:"status"` // "waiting", "running", "stopped", "crashed", "completed"
	PID      int      `json:"pid"`
	Restarts int      `json:"restarts"`
	LogFile  string   `json:"log_file,omitempty"`
}

// Serve implements suture.Service. It runs the process and blocks until
// the context is cancelled or the process exits.
func (s *Service) Serve(ctx context.Context) error {
	// Wait for dependencies before starting.
	if s.WaitReady != nil {
		s.notify("waiting", 0)
		if err := s.WaitReady(ctx); err != nil {
			return err // context cancelled
		}
	}

	shell, flag := shellCommand()
	cmd := exec.CommandContext(ctx, shell, flag, s.Config.Cmd)
	cmd.Dir = s.Config.Dir
	cmd.Env = buildEnv(s.Config.Env)

	var logFile *os.File
	if s.LogFile != "" {
		if err := os.MkdirAll(filepath.Dir(s.LogFile), 0o755); err != nil {
			s.notify("crashed", 0)
			return fmt.Errorf("creating log dir for %q: %w", s.Config.Name, err)
		}
		f, err := os.OpenFile(s.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			s.notify("crashed", 0)
			return fmt.Errorf("opening log file for %q: %w", s.Config.Name, err)
		}
		logFile = f
		cmd.Stdout = f
		cmd.Stderr = f
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if logFile != nil {
		defer func() { _ = logFile.Close() }()
	}

	// Create a process group so we can kill all child processes.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// When context is cancelled, kill the process group instead of just the shell.
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		}
		return nil
	}

	if err := cmd.Start(); err != nil {
		s.notify("crashed", 0)
		return fmt.Errorf("starting %q: %w", s.Config.Name, err)
	}

	// Probe the endpoint before marking as running so that dependents
	// wait until the process is actually accepting connections.
	if addr, network := s.endpoint(); addr != "" {
		if err := waitForEndpoint(ctx, network, addr); err != nil {
			// Process may have died while we were probing.
			if ctx.Err() != nil {
				s.notify("stopped", 0)
				return nil
			}
			s.notify("crashed", cmd.Process.Pid)
			return fmt.Errorf("health check for %q on %s/%s: %w", s.Config.Name, network, addr, err)
		}
	}

	s.notify("running", cmd.Process.Pid)

	err := cmd.Wait()

	// If context was cancelled, this is a clean shutdown.
	if ctx.Err() != nil {
		s.notify("stopped", 0)
		return nil
	}

	exitCode := cmd.ProcessState.ExitCode()
	restart := s.Config.Restart
	if restart == "" {
		restart = "always"
	}

	switch restart {
	case "never":
		if exitCode == 0 {
			s.notify("completed", 0)
		} else {
			s.notify("crashed", 0)
		}
		return suture.ErrDoNotRestart
	case "on_failure":
		if exitCode == 0 {
			s.notify("completed", 0)
			return suture.ErrDoNotRestart
		}
		s.notify("crashed", 0)
		return fmt.Errorf("process %q exited with code %d", s.Config.Name, exitCode)
	default: // "always"
		s.notify("crashed", 0)
		return fmt.Errorf("process %q exited: %w", s.Config.Name, err)
	}
}

func (s *Service) String() string {
	return fmt.Sprintf("process:%s", s.Config.Name)
}

func (s *Service) notify(status string, pid int) {
	if s.OnUpdate != nil {
		s.OnUpdate(State{
			Name:    s.Config.Name,
			Cmd:     s.Config.Cmd,
			Port:    s.Config.Port.Int(),
			Socket:  s.Config.Socket,
			Domains: s.Config.Domains,
			Status:  status,
			PID:     pid,
			LogFile: s.LogFile,
		})
	}
}

func shellCommand() (string, string) {
	if runtime.GOOS == "windows" {
		return "cmd", "/c"
	}
	return "sh", "-c"
}

// endpoint returns the network and address to probe, or empty strings if
// the process has no port or socket configured.
func (s *Service) endpoint() (addr, network string) {
	switch {
	case s.Config.Socket != "":
		return s.Config.Socket, "unix"
	case s.Config.Port.Int() != 0:
		return fmt.Sprintf("localhost:%d", s.Config.Port.Int()), "tcp"
	default:
		return "", ""
	}
}

// waitForEndpoint polls until a connection to network/addr succeeds or the
// context is cancelled. It tries every 250ms for up to 30 seconds.
func waitForEndpoint(ctx context.Context, network, addr string) error {
	timeout := 30 * time.Second
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			conn, err := net.DialTimeout(network, addr, time.Second)
			if err == nil {
				_ = conn.Close()
				return nil
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timed out after %s waiting for %s", timeout, addr)
			}
		}
	}
}

func buildEnv(extra map[string]string) []string {
	if len(extra) == 0 {
		return nil // inherit parent env
	}
	env := os.Environ()
	for k, v := range extra {
		env = append(env, k+"="+os.ExpandEnv(v))
	}
	return env
}
