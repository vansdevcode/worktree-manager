package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vansdevcode/worktree-manager/internal/routing"
)

var registerPort int
var registerMeta []string

var registerCmd = &cobra.Command{
	Use:   "register <domain>",
	Short: "Register a domain-to-port mapping",
	Long: `Register a local service by mapping a domain to a port.

Examples:
  devtree register myapp.test --port 8080
  devtree register myapp.test --port 8080 --meta stack=go:1.23 --meta env=dev`,
	Args: cobra.ExactArgs(1),
	RunE: runRegister,
}

func init() {
	registerCmd.Flags().IntVar(&registerPort, "port", 0, "Port number for the upstream service (required)")
	_ = registerCmd.MarkFlagRequired("port")
	registerCmd.Flags().StringArrayVar(&registerMeta, "meta", nil, "Metadata key=value pairs (can be repeated)")
}

func runRegister(_ *cobra.Command, args []string) error {
	domain := args[0]
	upstream := fmt.Sprintf("localhost:%d", registerPort)

	meta := parseMeta(registerMeta)

	routesPath := DefaultRoutesPath()
	table, err := routing.Load(routesPath)
	if err != nil {
		return fmt.Errorf("loading routes: %w", err)
	}

	table.Register(domain, upstream, meta)

	if err := routing.Save(routesPath, table); err != nil {
		return fmt.Errorf("saving routes: %w", err)
	}

	fmt.Printf("Registered %s -> %s\n", domain, upstream)

	sendReload()

	return nil
}

func parseMeta(pairs []string) map[string]string {
	if len(pairs) == 0 {
		return nil
	}
	meta := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		k, v, _ := strings.Cut(pair, "=")
		meta[k] = v
	}
	return meta
}

func sendReload() {
	sockPath := DefaultSocketPath()
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()
	_, _ = conn.Write([]byte("reload\n"))
}
