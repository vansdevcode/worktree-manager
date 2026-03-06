package main

import (
	"fmt"
	"strings"
	"text/tabwriter"
	"os"

	"github.com/spf13/cobra"
	"github.com/vansdevcode/worktree-manager/internal/routing"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered routes",
	Long:  "Display a table of all registered domains, their upstreams, and metadata.",
	Args:  cobra.NoArgs,
	RunE:  runList,
}

func runList(_ *cobra.Command, _ []string) error {
	routesPath := DefaultRoutesPath()
	table, err := routing.Load(routesPath)
	if err != nil {
		return fmt.Errorf("loading routes: %w", err)
	}

	sites := table.List()
	if len(sites) == 0 {
		fmt.Println("No routes registered")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "Domain\tUpstream\tMeta")
	for _, site := range sites {
		meta := formatMeta(site.Meta)
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", site.Domain, site.Upstream, meta)
	}
	return w.Flush()
}

func formatMeta(meta map[string]string) string {
	if len(meta) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(meta))
	for k, v := range meta {
		pairs = append(pairs, k+"="+v)
	}
	return strings.Join(pairs, ", ")
}
