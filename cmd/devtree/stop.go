package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vansdevcode/worktree-manager/internal/daemon"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the devtree daemon",
	Long:  "Stop the running devtree daemon by sending SIGTERM.",
	RunE:  runStop,
}

func runStop(_ *cobra.Command, _ []string) error {
	pidPath := daemon.DefaultPIDPath()
	if err := daemon.StopDaemon(pidPath); err != nil {
		return err
	}
	fmt.Println("Daemon stopped")
	return nil
}
