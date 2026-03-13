package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/vansdevcode/worktree-manager/internal/daemon"
	"github.com/vansdevcode/worktree-manager/internal/process"
)

var processConfigFlag string

var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Manage supervised processes",
	Long:  "Start, stop, and list processes defined in devtree.toml.",
}

var processUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Start all processes from devtree.toml",
	Long: `Start all processes defined in devtree.toml under supervision.
Processes with a domain configured will have routes auto-registered.

Examples:
  devtree process up
  devtree process up -f /path/to/devtree.toml`,
	Args: cobra.NoArgs,
	RunE: runProcessUp,
}

var processDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop all processes from devtree.toml",
	Long: `Stop all processes that were started from a devtree.toml config
and unregister their routes.

Examples:
  devtree process down
  devtree process down -f /path/to/devtree.toml`,
	Args: cobra.NoArgs,
	RunE: runProcessDown,
}

var processLsJSON bool

var processLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List managed processes and their status",
	Long: `Show the status of all processes being managed by the daemon.

Examples:
  devtree process ls
  devtree process ls --json`,
	Args: cobra.NoArgs,
	RunE: runProcessLs,
}

var processLogsFollow bool
var processLogsLines int

var processLogsCmd = &cobra.Command{
	Use:   "logs <process>",
	Short: "Show logs for a managed process",
	Long: `Display the log output for a supervised process.

Examples:
  devtree process logs web
  devtree process logs web -f
  devtree process logs web -n 50`,
	Args: cobra.ExactArgs(1),
	RunE: runProcessLogs,
}

func init() {
	processCmd.AddCommand(processUpCmd)
	processCmd.AddCommand(processDownCmd)
	processCmd.AddCommand(processLsCmd)
	processCmd.AddCommand(processLogsCmd)

	processCmd.PersistentFlags().StringVarP(&processConfigFlag, "file", "f", "", "Path to devtree.toml (default: auto-discover)")
	processLsCmd.Flags().BoolVar(&processLsJSON, "json", false, "Output as JSON")
	processLogsCmd.Flags().BoolVar(&processLogsFollow, "follow", false, "Follow log output (like tail -f)")
	processLogsCmd.Flags().IntVarP(&processLogsLines, "lines", "n", 100, "Number of lines to show from the end")
}

func resolveConfigPath() (string, error) {
	if processConfigFlag != "" {
		return filepath.Abs(processConfigFlag)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	return process.FindConfig(cwd)
}

func newClient() *daemon.Client {
	return &daemon.Client{SocketPath: DefaultSocketPath()}
}

func runProcessUp(_ *cobra.Command, _ []string) error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Validate config before sending to daemon.
	cfg, err := process.LoadConfig(configPath, cwd)
	if err != nil {
		return err
	}

	client := newClient()
	resp, err := client.ProcessUp(configPath, cwd)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}

	fmt.Printf("Started %d processes for project %q\n", len(cfg.Processes), cfg.Project.Name)
	for _, p := range cfg.Processes {
		route := ""
		for _, d := range p.Domains {
			route += fmt.Sprintf(" -> https://%s", d)
		}
		fmt.Printf("  %s: %s%s\n", p.Name, p.Cmd, route)
	}
	return nil
}

func runProcessDown(_ *cobra.Command, _ []string) error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return err
	}

	client := newClient()
	resp, err := client.ProcessDown(configPath)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}

	fmt.Println("Stopped all processes")
	return nil
}

func runProcessLs(_ *cobra.Command, _ []string) error {
	client := newClient()
	resp, err := client.ProcessList()
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}

	var groups []process.GroupStatus
	if err := json.Unmarshal(resp.Data, &groups); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if processLsJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(groups)
	}

	if len(groups) == 0 {
		fmt.Println("No managed processes")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PROJECT\tPROCESS\tSTATUS\tUPSTREAM\tDOMAINS\tRESTARTS")
	for _, g := range groups {
		for _, p := range g.Processes {
			upstream := "-"
			switch {
			case p.Socket != "":
				upstream = "unix:" + p.Socket
			case p.Port != 0:
				upstream = fmt.Sprintf(":%d", p.Port)
			}
			domains := "-"
			if len(p.Domains) > 0 {
				domains = strings.Join(p.Domains, ", ")
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\n",
				g.Project, p.Name, p.Status, upstream, domains, p.Restarts)
		}
	}
	return w.Flush()
}

func runProcessLogs(_ *cobra.Command, args []string) error {
	name := args[0]

	client := newClient()
	resp, err := client.ProcessLogs(name)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}

	var logPath string
	if err := json.Unmarshal(resp.Data, &logPath); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	f, err := os.Open(logPath)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Seek to show the last N lines.
	if err := seekToLastLines(f, processLogsLines); err != nil {
		return err
	}

	if _, err := io.Copy(os.Stdout, f); err != nil {
		return err
	}

	if !processLogsFollow {
		return nil
	}

	// Follow mode: poll for new data until interrupted.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sig:
			return nil
		case <-ticker.C:
			if _, err := io.Copy(os.Stdout, f); err != nil {
				return err
			}
		}
	}
}

// seekToLastLines positions the file reader to show approximately the last n lines.
func seekToLastLines(f *os.File, n int) error {
	info, err := f.Stat()
	if err != nil {
		return err
	}
	size := info.Size()
	if size == 0 {
		return nil
	}

	// Read up to 64KB from the end to find line boundaries.
	bufSize := int64(64 * 1024)
	if bufSize > size {
		bufSize = size
	}

	buf := make([]byte, bufSize)
	if _, err := f.ReadAt(buf, size-bufSize); err != nil && err != io.EOF {
		return err
	}

	// Count newlines from the end.
	lines := 0
	pos := len(buf) - 1
	for pos >= 0 {
		if buf[pos] == '\n' {
			lines++
			if lines > n {
				pos++ // move past this newline
				break
			}
		}
		pos--
	}
	if pos < 0 {
		pos = 0
	}

	offset := size - bufSize + int64(pos)
	_, err = f.Seek(offset, io.SeekStart)
	return err
}
