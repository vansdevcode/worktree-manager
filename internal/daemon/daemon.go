package daemon

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/vansdevcode/worktree-manager/internal/dns"
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

	d.proxyServer = proxy.New(d.httpPort, d.httpsPort)
	if err := d.proxyServer.Start(table); err != nil {
		_ = d.dnsServer.Stop()
		return fmt.Errorf("starting proxy: %w", err)
	}
	defer func() { _ = d.proxyServer.Stop() }()

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

	return nil
}

// startSocket creates a Unix socket listener and starts accepting connections
// in a background goroutine. Each connection can send "reload" to trigger a
// re-read of routes.yaml and a proxy reload.
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
		if line == "reload" {
			d.reload()
		}
	}
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
