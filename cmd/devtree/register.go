package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vansdevcode/worktree-manager/internal/routing"
)

var registerPort int
var registerSocket string
var registerMeta []string

var registerCmd = &cobra.Command{
	Use:   "register <domain>",
	Short: "Register a domain-to-upstream mapping",
	Long: `Register a local service by mapping a domain to a port or Unix socket.

Examples:
  devtree register myapp.test --port 8080
  devtree register myapp.test --socket /tmp/myapp.sock
  devtree register myapp.test --port 8080 --meta stack=go:1.23 --meta env=dev`,
	Args: cobra.ExactArgs(1),
	RunE: runRegister,
}

func init() {
	registerCmd.Flags().IntVar(&registerPort, "port", 0, "Port number for the upstream service")
	registerCmd.Flags().StringVar(&registerSocket, "socket", "", "Unix socket path for the upstream service")
	registerCmd.MarkFlagsMutuallyExclusive("port", "socket")
	registerCmd.Flags().StringArrayVar(&registerMeta, "meta", nil, "Metadata key=value pairs (can be repeated)")
}

func runRegister(_ *cobra.Command, args []string) error {
	domain := args[0]

	var upstream string
	switch {
	case registerSocket != "":
		upstream = "unix/" + registerSocket
	case registerPort != 0:
		upstream = fmt.Sprintf("localhost:%d", registerPort)
	default:
		return fmt.Errorf("either --port or --socket is required")
	}

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
	client := newClient()
	_ = client.SendReload()
}
