## Codebase Patterns
- Use `gopkg.in/yaml.v3` for YAML marshaling/unmarshaling
- Pre-commit hook runs `golangci-lint` — must use `_ =` for intentionally ignored error returns (errcheck)
- Atomic file writes: use `os.CreateTemp` + `os.Rename` pattern
- Module path: `github.com/vansdevcode/worktree-manager`
- Existing CLI is `cmd/wtm/`, new devtree CLI goes in `cmd/devtree/`
- Config dir convention: `~/.config/devtree/` for routing state files
- DNS package uses `mdns` import alias for `github.com/miekg/dns` to avoid collision with package name
- miekg/dns `Server.NotifyStartedFunc` callback signals when server is ready to accept connections
- Caddy programmatic config: build JSON as `map[string]any`, marshal, pass to `caddy.Load(data, true)`
- Import `_ "github.com/caddyserver/caddy/v2/modules/standard"` to register all standard Caddy modules
- TLS test clients must set `ServerName` in `tls.Config` to match the domain (SNI), not the IP address
- Daemon pattern: hidden `daemon` subcommand runs foreground; `start` launches it detached via `exec.Command` + `Setsid`
- PID file lifecycle: write on start, check with signal 0, remove on stop/SIGTERM cleanup
- Platform-specific code uses build tags in separate files (e.g., `daemon_unix.go` with `//go:build !windows`)
- Unix socket paths limited to ~104 chars on macOS; use `/tmp` for test sockets, not `t.TempDir()`
- Dashboard pattern: separate HTTP server on random port, reverse-proxied by Caddy via `SetDashboardPort()`

## 2026-03-05 - US-001
- Implemented `internal/routing` package with routing table YAML state file
- Types: `Table` (holds `Sites` map), `Site` (Upstream string, Meta map)
- Functions: `Load(path)`, `Save(path, table)` with atomic write (temp + rename)
- Methods: `Register`, `Unregister`, `Get`, `List` (sorted alphabetically)
- Files changed: `internal/routing/routing.go`, `internal/routing/routing_test.go`
- **Learnings for future iterations:**
  - golangci-lint errcheck requires `_ =` prefix for intentionally ignored error returns in cleanup paths
  - `gopkg.in/yaml.v3` is already in go.sum, no need to `go get` it
  - YAML unmarshal into a struct with nil map needs a nil check after unmarshal
---

## 2026-03-05 - US-002
- Implemented `devtree register <domain> --port <port>` CLI command
- Created `cmd/devtree/` package with `main.go`, `root.go`, `register.go`
- Register loads routes.yaml, calls `table.Register()`, saves atomically, prints confirmation
- Supports `--meta key=value` flags (repeatable) for arbitrary metadata
- Attempts Unix socket reload (best-effort, silently ignored if daemon not running)
- Files changed: `cmd/devtree/main.go`, `cmd/devtree/root.go`, `cmd/devtree/register.go`
- **Learnings for future iterations:**
  - `defer conn.Close()` must be `defer func() { _ = conn.Close() }()` for errcheck
  - devtree CLI uses cobra, same as wtm CLI but in separate `cmd/devtree/` package
  - `DefaultRoutesPath()` and `DefaultSocketPath()` are in `root.go` for reuse by other subcommands
---

## 2026-03-05 - US-003
- Implemented `devtree unregister <domain>` CLI command
- Created `cmd/devtree/unregister.go` with `runUnregister` function
- Loads routes.yaml, calls `table.Unregister()` (returns error if domain not found), saves atomically, prints confirmation
- Reuses `sendReload()` from register.go for Unix socket reload
- Added `unregisterCmd` to root command in `root.go`
- Files changed: `cmd/devtree/unregister.go` (new), `cmd/devtree/root.go` (modified)
- **Learnings for future iterations:**
  - `table.Unregister()` already returns an error for non-existent domains, so cobra's `RunE` propagates it as a non-zero exit
  - `sendReload()` is already defined in `register.go` and available package-wide — no need to duplicate
---

## 2026-03-05 - US-004
- Implemented `devtree list` CLI command
- Created `cmd/devtree/list.go` with tabwriter-based table output
- Prints "No routes registered" when table is empty
- Output sorted alphabetically by domain (uses `table.List()` which already sorts)
- Meta displayed as comma-separated key=value pairs
- Files changed: `cmd/devtree/list.go` (new), `cmd/devtree/root.go` (modified)
- **Learnings for future iterations:**
  - `text/tabwriter` works well for CLI table output with tab-separated columns
  - `table.List()` already returns sites sorted alphabetically — no extra sorting needed in the command
---

## 2026-03-05 - US-005
- Implemented `internal/dns` package with embedded DNS server
- `Server` struct wraps UDP and TCP `miekg/dns` servers
- `handler.ServeDNS` resolves `*.test` A queries to 127.0.0.1, all others get NXDOMAIN
- Functions: `New(addr)`, `Start()` (blocks until ready), `Stop()`
- Files changed: `internal/dns/dns.go` (new), `internal/dns/dns_test.go` (new), `go.mod`, `go.sum`
- 6 tests: UDP/TCP resolution, subdomains, NXDOMAIN for non-.test, NXDOMAIN for non-A queries, case insensitivity
- **Learnings for future iterations:**
  - Use `mdns` as import alias for `github.com/miekg/dns` to avoid package name collision
  - miekg/dns doesn't support port 0; tests must find a free port first via `net.ListenUDP`, then close and pass to `New()`
  - `Server.NotifyStartedFunc` is the reliable way to wait for server readiness before sending queries
  - DNS names are FQDN with trailing dot — match with `strings.HasSuffix(name, ".test.")`
---

## 2026-03-05 - US-006
- Implemented `internal/proxy` package with TLS-terminating reverse proxy using embedded Caddy
- `Server` struct wraps Caddy with `New(httpPort, httpsPort)`, `Start(table)`, `Reload(table)`, `Stop()`
- `buildConfig()` creates Caddy JSON config programmatically with `tls internal` issuers and `reverse_proxy` handlers
- Config loaded via `caddy.Load(data, true)` — no Caddyfile on disk
- Files changed: `internal/proxy/proxy.go` (new), `internal/proxy/proxy_test.go` (new), `go.mod`, `go.sum`
- 2 tests: round-trip HTTPS through proxy to backend, reload with new route
- **Learnings for future iterations:**
  - Caddy's `caddy.Load()` takes JSON bytes and a boolean (whether to retain config in memory)
  - Import `_ "github.com/caddyserver/caddy/v2/modules/standard"` to register HTTP, TLS, reverse_proxy modules
  - When testing TLS with `http.Client`, must set `tls.Config.ServerName` to match the domain — the URL hostname sets SNI, not the `Host` header
  - Caddy stores its internal CA at `~/Library/Application Support/Caddy/pki/authorities/local/` on macOS; expired CAs cause "tls: internal error"
  - Caddy global instance: only one Caddy runs per process; `caddy.Stop()` stops it; tests share the process
---

## 2026-03-05 - US-007
- Implemented `internal/daemon` package with `Daemon` struct managing DNS + proxy lifecycle
- Functions: `New()`, `Run()` (blocks on SIGTERM/SIGINT), `IsRunning()`, `StopDaemon()`, `DefaultPIDPath()`
- `devtree start` launches daemon in background via `exec.Command` with `Setsid: true` for process detachment
- `devtree stop` reads PID file, sends SIGTERM, removes PID file; errors if not running
- Hidden `devtree daemon` command runs the daemon in foreground (called by `start`)
- PID file written to `~/.config/devtree/devtree.pid`; daemon cleans up PID file on SIGTERM via deferred `os.Remove`
- Files changed: `internal/daemon/daemon.go` (new), `internal/daemon/daemon_test.go` (new), `cmd/devtree/start.go` (new), `cmd/devtree/stop.go` (new), `cmd/devtree/daemon.go` (new), `cmd/devtree/daemon_unix.go` (new), `cmd/devtree/root.go` (modified)
- 5 unit tests: PID file write, IsRunning with no file/stale PID/current process, StopDaemon when not running
- **Learnings for future iterations:**
  - Use `Setsid: true` in `SysProcAttr` to detach daemon from parent process session
  - Platform-specific `SysProcAttr` needs build tags (`//go:build !windows`) in separate file
  - Signal 0 (`proc.Signal(syscall.Signal(0))`) checks if a process exists without sending a real signal
  - Hidden cobra commands (`Hidden: true`) are useful for internal subcommands like `daemon`
---

## 2026-03-05 - US-008
- Implemented Unix socket listener in `internal/daemon/daemon.go` for hot-reload
- Daemon listens on `~/.config/devtree/devtree.sock` and accepts "reload" messages
- On reload: re-reads `routes.yaml` and calls `proxy.Reload()` to hot-reload Caddy config
- Socket cleaned up on daemon shutdown (deferred `listener.Close()` + `os.Remove`)
- `register` and `unregister` commands already send reload via `sendReload()` in `register.go`
- If daemon is not running, register/unregister still update YAML without error (existing behavior)
- Added `socketPath` field to `Daemon` struct; updated `New()` signature
- Files changed: `internal/daemon/daemon.go` (modified), `internal/daemon/daemon_test.go` (modified), `cmd/devtree/daemon.go` (modified)
- 3 new tests: socket creation, stale socket removal, reload reads routes
- **Learnings for future iterations:**
  - Unix socket paths are limited to ~104 chars on macOS; use `/tmp` for test socket paths, not `t.TempDir()`
  - `proxyServer` may be nil when socket receives reload in tests; add nil check before calling `Reload()`
  - `bufio.Scanner` with line-based protocol ("reload\n") works well for simple Unix socket commands
---

## 2026-03-05 - US-009
- Implemented `internal/dashboard` package with embedded HTML dashboard and JSON API
- Single HTML page embedded via `go:embed` — no build tools or JS framework
- `Server` struct with `New(routesPath)`, `Start()` (returns port), `Stop()`
- JSON API at `/api/status` returns sites with domain, upstream, meta, and health status
- Health check: TCP connect with 500ms timeout to each upstream; green/red dot
- HTML dashboard auto-refreshes every 5s via `setInterval` + `fetch`
- Integrated into daemon: dashboard server starts on random port, Caddy reverse-proxies `dashboard.devtree.test` to it
- Added `SetDashboardPort(port)` to proxy `Server` to configure the dashboard route in Caddy config
- Files changed: `internal/dashboard/dashboard.go` (new), `internal/dashboard/index.html` (new), `internal/dashboard/dashboard_test.go` (new), `internal/daemon/daemon.go` (modified), `internal/proxy/proxy.go` (modified)
- 4 tests: HTML serving, empty API response, API with routes + health checks, checkHealth unit test
- **Learnings for future iterations:**
  - `go:embed` with `embed.FS` works well for serving single-file HTML dashboards via `http.FileServer`
  - Dashboard as a separate HTTP server reverse-proxied by Caddy is simpler than writing a custom Caddy module
  - `net.DialTimeout("tcp", addr, timeout)` is the simplest health check for TCP services
  - Dashboard route must be added before user routes in Caddy config to avoid domain conflicts
---
