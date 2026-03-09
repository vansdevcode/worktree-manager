# devtree POC — Local Development Routing CLI

## Goal

Build `devtree` as a standalone CLI within this repo (`cmd/devtree/`) that provides local development routing: DNS resolution, TLS-terminating reverse proxy, routing table, and a status dashboard. The existing `wtm` CLI remains untouched. Once `devtree` works end-to-end, we merge worktree management into it.

## Architecture

```
cmd/
  wtm/           # existing — no changes
  devtree/       # new entry point
    main.go
    root.go
    start.go      # devtree start (daemon)
    stop.go       # devtree stop
    bootstrap.go  # devtree bootstrap (one-time OS setup)
    register.go   # devtree register <domain> --port <port>
    unregister.go # devtree unregister <domain>
    list.go       # devtree list
internal/
  routing/        # routing table (YAML state file)
  dns/            # embedded DNS server (miekg/dns)
  proxy/          # embedded Caddy reverse proxy
  dashboard/      # embedded HTML dashboard + JSON API
```

## Phases

### Phase 1: Routing Table (`internal/routing`)

The foundation. A YAML state file that maps domains to upstreams.

**State file** (`~/.config/devtree/routes.yaml`):
```yaml
sites:
  myapp.test:
    upstream: localhost:8080
    meta:
      stack: "go:1.23"
  frontend.test:
    upstream: localhost:3000
    meta:
      stack: "node:22"
      framework: "vite"
```

**Package API**:
- `Load(path string) (*Table, error)` — read YAML state
- `Save(path string, table *Table) error` — write YAML state (atomic write via temp file + rename)
- `(*Table) Register(domain string, upstream string, meta map[string]string) error`
- `(*Table) Unregister(domain string) error`
- `(*Table) Get(domain string) (*Site, bool)`
- `(*Table) List() []Site`

**CLI commands** (wired in this phase):
- `devtree register <domain> --port <port> [--meta key=value ...]`
- `devtree unregister <domain>`
- `devtree list`

**Tests**: unit tests for Table CRUD, YAML round-trip, atomic save.

**Done when**: `devtree register foo.test --port 8080 && devtree list` works.

---

### Phase 2: DNS Server (`internal/dns`)

Embedded DNS that resolves all `*.test` queries to `127.0.0.1`.

**Package API**:
- `NewServer(addr string) *Server`
- `(*Server) Start() error` — listen on UDP+TCP port 53
- `(*Server) Stop() error`

Uses `github.com/miekg/dns`. Responds to A queries matching `*.test` with `127.0.0.1`. All other queries get NXDOMAIN.

**No CLI commands in this phase** — the DNS server is started as part of the daemon (Phase 4).

**Tests**: unit tests using miekg/dns client against the server on a high port.

**Done when**: DNS server resolves `anything.test` to `127.0.0.1` in tests.

---

### Phase 3: Reverse Proxy (`internal/proxy`)

Embedded Caddy that terminates TLS and reverse-proxies to upstreams from the routing table.

**Package API**:
- `NewProxy() *Proxy`
- `(*Proxy) Start() error` — start Caddy on ports 80/443
- `(*Proxy) Stop() error`
- `(*Proxy) Reload(table *routing.Table) error` — rebuild Caddy config from routing table and hot-reload via `caddy.Load()`

Caddy config is built programmatically (no Caddyfile). Each registered site gets:
- `tls internal` for automatic local certificates
- `reverse_proxy` to its upstream

**Dependencies**: `github.com/caddyserver/caddy/v2` and required modules.

**Tests**: integration test — start proxy on high ports, register a route, verify HTTPS request reaches a test HTTP server.

**Done when**: `curl -k https://myapp.test` (with DNS pointing to 127.0.0.1) proxies to the upstream.

---

### Phase 4: Daemon & Dashboard (`cmd/devtree/`, `internal/dashboard`)

Wire DNS + Proxy into a long-running daemon with PID file management, and serve a dashboard.

**Daemon (`start`/`stop`)**:
- `devtree start` — starts DNS server + Caddy proxy in foreground (or background with PID file)
- `devtree stop` — reads PID file, sends SIGTERM
- PID file at `~/.config/devtree/devtree.pid`
- On `register`/`unregister`, signal the daemon to reload (e.g., SIGHUP or write to a Unix socket)

**Hot-reload strategy**:
- `register`/`unregister` update the YAML file, then signal the daemon
- Daemon watches the YAML file (or listens on a Unix socket) and calls `proxy.Reload()`

**Dashboard (`internal/dashboard`)**:
- Single HTML page embedded via `embed` package
- Served by Caddy at `https://dashboard.devtree.test`
- JSON API at `https://dashboard.devtree.test/api/status` returns routing table + health
- Dashboard JS fetches API, renders site list with clickable links, upstream ports, meta labels, health dots
- Auto-refreshes every 5 seconds

**Tests**: daemon lifecycle test (start, register, stop). Dashboard API returns correct JSON.

**Done when**: `devtree start`, `devtree register myapp.test --port 8080`, open `https://dashboard.devtree.test` in browser, see the site listed.

---

### Phase 5: Bootstrap (`cmd/devtree/bootstrap.go`)

One-time system setup command.

**macOS**:
- Create `/etc/resolver/test` with `nameserver 127.0.0.1` (requires sudo)
- Run `caddy trust` to install root CA in system trust store

**Linux**:
- Configure systemd-resolved or `/etc/resolv.conf` for `.test` TLD
- Run `caddy trust`

**CLI**: `devtree bootstrap` — detects OS, runs the appropriate setup, reports what it did.

**Done when**: after `devtree bootstrap && devtree start`, `ping anything.test` resolves to `127.0.0.1` and `curl https://myapp.test` gets a valid TLS certificate.

---

## Post-POC: Merge wtm

After all phases are complete and `devtree` works standalone:

1. Move worktree commands (`init`, `add`, `rm`, `ls`, `pr`, `hook`) into `cmd/devtree/`
2. Rename `add` to `new` per the plan
3. Wire worktree lifecycle to routing: `devtree new` auto-registers a route if `.devtree.yaml` exists in the project
4. `devtree rm` auto-unregisters
5. Retire `cmd/wtm/`

## Dependencies to Add

| Package | Purpose |
|---|---|
| `github.com/miekg/dns` | DNS server |
| `github.com/caddyserver/caddy/v2` | Reverse proxy + TLS |

## State & Config Paths

| File | Purpose |
|---|---|
| `~/.config/devtree/routes.yaml` | Routing table |
| `~/.config/devtree/devtree.pid` | Daemon PID file |
| `/etc/resolver/test` (macOS) | DNS resolver config |

## Design Constraints

- No Caddyfile on disk — config is programmatic via `caddy.Load()`
- Routing table is the single source of truth — proxy config is derived from it
- DNS server is dumb — resolves everything under `.test` to `127.0.0.1`, no per-site logic
- Dashboard is a single embedded HTML file, no build tools, no JS framework
- Cross-platform: macOS, Linux, WSL2. Only `bootstrap` is OS-specific
