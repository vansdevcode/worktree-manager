package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/vansdevcode/worktree-manager/internal/certs"
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

func installCA() error {
	fmt.Println("Installing local CA certificate into system trust store...")
	if err := certs.EnsureCA(); err != nil {
		return fmt.Errorf("installing CA: %w", err)
	}
	return nil
}
