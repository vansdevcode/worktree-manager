package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Configure your OS to route .test domains through devtree",
	Long:  "One-time setup command that configures DNS resolution for .test domains and installs the local CA certificate.",
	RunE:  runBootstrap,
}

func runBootstrap(_ *cobra.Command, _ []string) error {
	fmt.Printf("Detected OS: %s\n\n", runtime.GOOS)
	return bootstrapPlatform()
}

func runCaddyTrust() error {
	caddyPath, err := exec.LookPath("caddy")
	if err != nil {
		fmt.Println("Warning: caddy not found in PATH. Please install Caddy and run 'caddy trust' manually.")
		return nil
	}

	cmd := exec.Command(caddyPath, "trust")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running caddy trust: %w", err)
	}
	return nil
}
