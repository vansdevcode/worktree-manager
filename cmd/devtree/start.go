package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/vansdevcode/worktree-manager/internal/daemon"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the devtree daemon",
	Long:  "Start the devtree daemon in the background, launching the DNS server and reverse proxy.",
	RunE:  runStart,
}

func runStart(_ *cobra.Command, _ []string) error {
	pidPath := daemon.DefaultPIDPath()
	if _, running := daemon.IsRunning(pidPath); running {
		return fmt.Errorf("daemon is already running")
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	cmd := exec.Command(exe, "daemon")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	cmd.SysProcAttr = daemonSysProcAttr()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting daemon: %w", err)
	}

	fmt.Printf("Daemon started (PID %d)\n", cmd.Process.Pid)
	return nil
}
