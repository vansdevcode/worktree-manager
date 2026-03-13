package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "devtree",
	Short: "Local development routing CLI",
	Long:  "devtree provides local development routing with DNS, TLS, and reverse proxying for .test domains.",
	SilenceErrors: true,
	SilenceUsage:  true,
}

// DefaultRoutesPath returns the default path for routes.yaml.
func DefaultRoutesPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot determine home directory: %v\n", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".config", "devtree", "routes.yaml")
}

// DefaultSocketPath returns the default path for the daemon Unix socket.
func DefaultSocketPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot determine home directory: %v\n", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".config", "devtree", "devtree.sock")
}

func init() {
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(unregisterCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(bootstrapCmd)
	rootCmd.AddCommand(processCmd)
	rootCmd.AddCommand(dashboardCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
