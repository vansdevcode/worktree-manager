package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vansdevcode/worktree-manager/internal/routing"
)

var unregisterCmd = &cobra.Command{
	Use:   "unregister <domain>",
	Short: "Remove a registered domain",
	Long: `Remove a domain-to-port mapping so that it stops being routed.

Examples:
  devtree unregister myapp.test`,
	Args: cobra.ExactArgs(1),
	RunE: runUnregister,
}

func runUnregister(_ *cobra.Command, args []string) error {
	domain := args[0]

	routesPath := DefaultRoutesPath()
	table, err := routing.Load(routesPath)
	if err != nil {
		return fmt.Errorf("loading routes: %w", err)
	}

	if err := table.Unregister(domain); err != nil {
		return err
	}

	if err := routing.Save(routesPath, table); err != nil {
		return fmt.Errorf("saving routes: %w", err)
	}

	fmt.Printf("Unregistered %s\n", domain)

	sendReload()

	return nil
}
