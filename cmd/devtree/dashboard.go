package main

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Open the devtree dashboard in a browser",
	Long:  "Open the devtree dashboard (https://dashboard.devtree.test) in the default browser.",
	Args:  cobra.NoArgs,
	RunE:  runDashboard,
}

func runDashboard(_ *cobra.Command, _ []string) error {
	url := "https://dashboard.devtree.test"
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("unsupported platform; open %s manually", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("opening browser: %w", err)
	}
	fmt.Println(url)
	return nil
}
