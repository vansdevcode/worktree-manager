package main

import (
	"github.com/spf13/cobra"
	"github.com/vansdevcode/worktree-manager/internal/daemon"
)

var daemonCmd = &cobra.Command{
	Use:    "daemon",
	Short:  "Run the daemon in the foreground (internal use)",
	Hidden: true,
	RunE:   runDaemon,
}

func runDaemon(_ *cobra.Command, _ []string) error {
	d := daemon.New(
		DefaultRoutesPath(),
		daemon.DefaultPIDPath(),
		DefaultSocketPath(),
		"127.0.0.1:10053",
		80,
		443,
	)
	return d.Run()
}
