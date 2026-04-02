package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/thejerf/suture/v4"
	"github.com/vansdevcode/worktree-manager/internal/dashboard"
	"github.com/vansdevcode/worktree-manager/internal/dns"
	"github.com/vansdevcode/worktree-manager/internal/process"
	"github.com/vansdevcode/worktree-manager/internal/proxy"
	"github.com/vansdevcode/worktree-manager/internal/routing"
)

// Daemon manages the DNS server and reverse proxy lifecycle.
type Daemon struct {
	routesPath  string
	pidPath     string
	socketPath  string
	dnsAddr     string
	httpPort    int
	httpsPort   int
	dnsServer   *dns.Server
	proxyServer *proxy.Server
	processSup  *process.Supervisor
}

// New creates a daemon with the given configuration.
func New(routesPath, pidPath, socketPath, dnsAddr string, httpPort, httpsPort int) *Daemon {
	return &Daemon{
		routesPath: routesPath,
		pidPath:    pidPath,
		socketPath: socketPath,
		dnsAddr:    dnsAddr,
		httpPort:   httpPort,
		httpsPort:  httpsPort,
	}
}

// Run starts the DNS server and proxy, writes the PID file, and blocks until
// SIGTERM or SIGINT is received. It cleans up on exit.
func (d *Daemon) Run() error {
	if err := d.writePID(); err != nil {
		return err
	}
	defer func() { _ = os.Remove(d.pidPath) }()

	table, err := routing.Load(d.routesPath)
	if err != nil {
		return fmt.Errorf("loading routes: %w", err)
	}

	d.dnsServer = dns.New(d.dnsAddr)
	if err := d.dnsServer.Start(); err != nil {
		return fmt.Errorf("starting dns: %w", err)
	}
	defer func() { _ = d.dnsServer.Stop() }()

	// Create the top-level suture supervisor for process management.
	root := suture.New("devtree", suture.Spec{
		EventHook: func(e suture.Event) {
			log.Printf("[suture] %s", e)
		},
	})

	logDir := filepath.Join(filepath.Dir(d.pidPath), "logs")
	d.processSup = process.NewSupervisor(root, logDir, d.registerRoute, d.unregisterRoute)

	dashSrv := dashboard.New(d.routesPath)
	dashSrv.SetProcessLister(d.processSup)
	dashPort, err := dashSrv.Start()
	if err != nil {
		_ = d.dnsServer.Stop()
		return fmt.Errorf("starting dashboard: %w", err)
	}
	defer func() { _ = dashSrv.Stop() }()

	d.proxyServer = proxy.New(d.httpPort, d.httpsPort)
	d.proxyServer.SetDashboardPort(dashPort)
	if err := d.proxyServer.Start(table); err != nil {
		_ = dashSrv.Stop()
		_ = d.dnsServer.Stop()
		return fmt.Errorf("starting proxy: %w", err)
	}
	defer func() { _ = d.proxyServer.Stop() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := root.Serve(ctx); err != nil {
			log.Printf("[suture] root supervisor exited: %v", err)
		}
	}()

	listener, err := d.startSocket()
	if err != nil {
		return fmt.Errorf("starting socket: %w", err)
	}
	defer func() {
		_ = listener.Close()
		_ = os.Remove(d.socketPath)
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig

	cancel() // stop all supervised processes

	return nil
}

// registerRoute adds a route and reloads the proxy.
func (d *Daemon) registerRoute(domain, upstream string, meta map[string]string) error {
	table, err := routing.Load(d.routesPath)
	if err != nil {
		return fmt.Errorf("loading routes: %w", err)
	}
	table.Register(domain, upstream, meta)
	if err := routing.Save(d.routesPath, table); err != nil {
		return fmt.Errorf("saving routes: %w", err)
	}
	d.reload()
	return nil
}

// unregisterRoute removes a route and reloads the proxy.
func (d *Daemon) unregisterRoute(domain string) error {
	table, err := routing.Load(d.routesPath)
	if err != nil {
		return fmt.Errorf("loading routes: %w", err)
	}
	if err := table.Unregister(domain); err != nil {
		return err
	}
	if err := routing.Save(d.routesPath, table); err != nil {
		return fmt.Errorf("saving routes: %w", err)
	}
	d.reload()
	return nil
}

// startSocket creates a Unix socket listener and starts accepting connections
// in a background goroutine. Each connection can send "reload" to trigger a
// re-read of routes.yaml and a proxy reload, or JSON commands for process management.
func (d *Daemon) startSocket() (net.Listener, error) {
	// Remove stale socket file if it exists.
	_ = os.Remove(d.socketPath)

	listener, err := net.Listen("unix", d.socketPath)
	if err != nil {
		return nil, fmt.Errorf("listening on %s: %w", d.socketPath, err)
	}

	go d.acceptLoop(listener)

	return listener, nil
}

func (d *Daemon) acceptLoop(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return // listener closed
		}
		go d.handleConn(conn)
	}
}

func (d *Daemon) handleConn(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line[0] == '{' {
			d.handleJSON(conn, []byte(line))
		} else if line == "reload" {
			d.reload()
		}
	}
}

func (d *Daemon) handleJSON(conn net.Conn, data []byte) {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		d.sendResponse(conn, ErrResponse(fmt.Errorf("invalid request: %w", err)))
		return
	}

	var resp Response
	switch req.Cmd {
	case "process-up":
		if err := d.processSup.Up(req.ConfigPath, req.WorkDir); err != nil {
			resp = ErrResponse(err)
		} else {
			resp = OKResponse(nil)
		}
	case "process-down":
		if err := d.processSup.Down(req.ConfigPath); err != nil {
			resp = ErrResponse(err)
		} else {
			resp = OKResponse(nil)
		}
	case "process-ls":
		resp = OKResponse(d.processSup.List())
	case "process-logs":
		logFile := d.processSup.LogFile(req.Name)
		if logFile == "" {
			resp = ErrResponse(fmt.Errorf("process %q not found or has no log file", req.Name))
		} else {
			resp = OKResponse(logFile)
		}
	default:
		resp = ErrResponse(fmt.Errorf("unknown command: %s", req.Cmd))
	}

	d.sendResponse(conn, resp)
}

func (d *Daemon) sendResponse(conn net.Conn, resp Response) {
	data, _ := json.Marshal(resp)
	data = append(data, '\n')
	_, _ = conn.Write(data)
}

func (d *Daemon) reload() {
	table, err := routing.Load(d.routesPath)
	if err != nil {
		log.Printf("reload: loading routes: %v", err)
		return
	}
	if d.proxyServer == nil {
		return
	}
	if err := d.proxyServer.Reload(table); err != nil {
		log.Printf("reload: reloading proxy: %v", err)
	}
}

func (d *Daemon) writePID() error {
	dir := filepath.Dir(d.pidPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	pid := os.Getpid()
	return os.WriteFile(d.pidPath, []byte(strconv.Itoa(pid)), 0o644)
}

// DefaultPIDPath returns the default path for the PID file.
func DefaultPIDPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "devtree", "devtree.pid")
}

// IsRunning checks if a daemon is already running by reading the PID file
// and checking if the process exists.
func IsRunning(pidPath string) (int, bool) {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return 0, false
	}
	// Signal 0 checks if the process exists without sending a signal.
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return 0, false
	}
	return pid, true
}

// StopDaemon reads the PID file, sends SIGTERM, and removes the PID file.
func StopDaemon(pidPath string) error {
	pid, running := IsRunning(pidPath)
	if !running {
		return fmt.Errorf("daemon is not running")
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process %d: %w", pid, err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("sending SIGTERM to %d: %w", pid, err)
	}
	_ = os.Remove(pidPath)
	return nil
}
