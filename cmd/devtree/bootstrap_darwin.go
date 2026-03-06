//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func bootstrapPlatform() error {
	// Create /etc/resolver/test with nameserver 127.0.0.1
	resolverDir := "/etc/resolver"
	resolverFile := resolverDir + "/test"
	resolverContent := "nameserver 127.0.0.1\n"

	if _, err := os.Stat(resolverDir); os.IsNotExist(err) {
		fmt.Printf("Creating %s (requires sudo)...\n", resolverDir)
		cmd := exec.Command("sudo", "mkdir", "-p", resolverDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("creating resolver directory: %w", err)
		}
	}

	fmt.Printf("Writing %s (requires sudo)...\n", resolverFile)
	cmd := exec.Command("sudo", "tee", resolverFile)
	cmd.Stdin = strings.NewReader(resolverContent)
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("writing resolver file: %w", err)
	}

	fmt.Println("\nInstalling Caddy local CA certificate...")
	if err := runCaddyTrust(); err != nil {
		return err
	}

	fmt.Println("\n--- Bootstrap Summary ---")
	fmt.Println("  [ok] Created /etc/resolver/test (nameserver 127.0.0.1)")
	fmt.Println("  [ok] Installed Caddy local CA in system trust store")
	fmt.Println("")
	fmt.Println("Next steps:")
	fmt.Println("  1. Run: devtree start")
	fmt.Println("  2. Register a service: devtree register myapp.test --port 8080")
	fmt.Println("  3. Open: https://myapp.test")
	return nil
}
