//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func bootstrapPlatform() error {
	// Check if systemd-resolved is available
	if _, err := exec.LookPath("resolvectl"); err == nil {
		return bootstrapSystemdResolved()
	}
	return bootstrapResolvConf()
}

func bootstrapSystemdResolved() error {
	fmt.Println("Detected systemd-resolved, configuring .test DNS...")

	dropInDir := "/etc/systemd/resolved.conf.d"
	dropInFile := dropInDir + "/devtree-test.conf"
	content := "[Resolve]\nDNS=127.0.0.1\nDomains=~test\n"

	if _, err := os.Stat(dropInDir); os.IsNotExist(err) {
		fmt.Printf("Creating %s (requires sudo)...\n", dropInDir)
		cmd := exec.Command("sudo", "mkdir", "-p", dropInDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("creating resolved.conf.d: %w", err)
		}
	}

	fmt.Printf("Writing %s (requires sudo)...\n", dropInFile)
	cmd := exec.Command("sudo", "tee", dropInFile)
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("writing resolved config: %w", err)
	}

	fmt.Println("Restarting systemd-resolved...")
	cmd = exec.Command("sudo", "systemctl", "restart", "systemd-resolved")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("restarting systemd-resolved: %w", err)
	}

	fmt.Println()
	if err := installCA(); err != nil {
		return err
	}

	fmt.Println("\n--- Bootstrap Summary ---")
	fmt.Printf("  [ok] Configured DNS via systemd-resolved (%s)\n", dropInFile)
	fmt.Println("  [ok] Installed local CA in system trust store")
	fmt.Println("")
	fmt.Println("Next steps:")
	fmt.Println("  1. Run: devtree start")
	fmt.Println("  2. Register a service: devtree register myapp.test --port 8080")
	fmt.Println("  3. Open: https://myapp.test")
	return nil
}

func bootstrapResolvConf() error {
	fmt.Println("Configuring /etc/resolv.conf for .test domains...")

	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading /etc/resolv.conf: %w", err)
	}

	if strings.Contains(string(data), "nameserver 127.0.0.1") {
		fmt.Println("nameserver 127.0.0.1 already present in /etc/resolv.conf")
	} else {
		fmt.Println("Adding nameserver 127.0.0.1 to /etc/resolv.conf (requires sudo)...")
		newContent := "nameserver 127.0.0.1\n" + string(data)
		cmd := exec.Command("sudo", "tee", "/etc/resolv.conf")
		cmd.Stdin = strings.NewReader(newContent)
		cmd.Stdout = nil
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("updating /etc/resolv.conf: %w", err)
		}
	}

	fmt.Println()
	if err := installCA(); err != nil {
		return err
	}

	fmt.Println("\n--- Bootstrap Summary ---")
	fmt.Println("  [ok] Configured DNS via /etc/resolv.conf")
	fmt.Println("  [ok] Installed local CA in system trust store")
	fmt.Println("")
	fmt.Println("Next steps:")
	fmt.Println("  1. Run: devtree start")
	fmt.Println("  2. Register a service: devtree register myapp.test --port 8080")
	fmt.Println("  3. Open: https://myapp.test")
	return nil
}
