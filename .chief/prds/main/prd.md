# PRD: devtree — Local Development Routing CLI

## Introduction

`devtree` is a standalone CLI that provides local development routing for developers working on multiple projects simultaneously. It bundles a DNS server, TLS-terminating reverse proxy, routing table, and a status dashboard into a single tool. Developers register local services (e.g., `myapp.test` on port 8080), and `devtree` handles DNS resolution, HTTPS certificates, and reverse proxying automatically. This eliminates the need to remember port numbers, configure `/etc/hosts` manually, or set up nginx/Caddy per project.

## Goals

- Provide zero-config HTTPS for local development services via `.test` TLD
- Replace manual `/etc/hosts` editing and per-project reverse proxy setup with a single CLI
- Enable developers to access local services by memorable domain names (e.g., `https://myapp.test`)
- Offer a visual dashboard showing all registered services and their health status
- Support macOS, Linux, and WSL2 with a one-time bootstrap command
- Keep the routing table as the single source of truth — all other state is derived from it

## User Stories

### US-001: Routing Table YAML State File
**Priority:** 1
**Description:** As a developer, I need a persistent routing table so that registered domains and their upstreams survive restarts.

**Acceptance Criteria:**
- [ ] Routing table stored at `~/.config/devtree/routes.yaml`
- [ ] YAML schema supports `sites` map with `upstream` (string) and `meta` (map of string to string) per domain
- [ ] `Load(path)` reads and parses the YAML file; returns empty table if file doesn't exist
- [ ] `Save(path, table)` writes atomically (temp file + rename) to prevent corruption
- [ ] Unit tests cover CRUD operations, YAML round-trip, and atomic save

### US-002: Register a Route via CLI
**Priority:** 1
**Description:** As a developer, I want to register a domain-to-port mapping from the command line so that I can expose a local service.

**Acceptance Criteria:**
- [ ] `devtree register <domain> --port <port>` adds an entry to `routes.yaml`
- [ ] Optional `--meta key=value` flags store arbitrary metadata (e.g., `--meta stack=go:1.23`)
- [ ] If the domain already exists, the entry is updated (not duplicated)
- [ ] Upstream is stored as `localhost:<port>`
- [ ] Command prints confirmation: `Registered myapp.test -> localhost:8080`
- [ ] If the daemon is running, sends a reload command via Unix socket

### US-003: Unregister a Route via CLI
**Priority:** 1
**Description:** As a developer, I want to remove a registered domain so that it stops being routed.

**Acceptance Criteria:**
- [ ] `devtree unregister <domain>` removes the entry from `routes.yaml`
- [ ] If the domain doesn't exist, prints an error message and exits with non-zero code
- [ ] Command prints confirmation: `Unregistered myapp.test`
- [ ] If the daemon is running, sends a reload command via Unix socket

### US-004: List Registered Routes
**Priority:** 1
**Description:** As a developer, I want to see all registered routes so that I know what's currently configured.

**Acceptance Criteria:**
- [ ] `devtree list` prints a table with columns: Domain, Upstream, Meta
- [ ] If no routes are registered, prints `No routes registered`
- [ ] Output is sorted alphabetically by domain

### US-005: Embedded DNS Server
**Priority:** 2
**Description:** As a developer, I need `.test` domains to resolve to `127.0.0.1` automatically so that browsers and CLI tools can reach my local services by name.

**Acceptance Criteria:**
- [ ] DNS server listens on UDP and TCP port 53 (on `127.0.0.1`)
- [ ] All A queries for `*.test` respond with `127.0.0.1`
- [ ] All other queries respond with NXDOMAIN
- [ ] Uses `github.com/miekg/dns` library
- [ ] Unit tests verify resolution using miekg/dns client against a high port

### US-006: TLS-Terminating Reverse Proxy
**Priority:** 2
**Description:** As a developer, I want HTTPS to work automatically for my `.test` domains so that I get production-like local development without certificate hassle.

**Acceptance Criteria:**
- [ ] Reverse proxy listens on ports 80 and 443
- [ ] Each registered site gets `tls internal` (Caddy's built-in local CA) and `reverse_proxy` to its upstream
- [ ] Caddy config is built programmatically via `caddy.Load()` — no Caddyfile on disk
- [ ] `Reload(table)` rebuilds and hot-reloads config from the current routing table
- [ ] Integration test: start proxy on high ports, register a route, verify HTTPS request reaches a test HTTP server

### US-007: Daemon Start and Stop
**Priority:** 3
**Description:** As a developer, I want to start and stop the devtree daemon so that DNS and proxying run in the background.

**Acceptance Criteria:**
- [ ] `devtree start` launches the daemon in the background
- [ ] Daemon starts the DNS server and Caddy reverse proxy
- [ ] PID file written to `~/.config/devtree/devtree.pid`
- [ ] If daemon is already running (PID file exists and process alive), print error and exit
- [ ] `devtree stop` reads PID file, sends SIGTERM, removes PID file
- [ ] If daemon is not running, `devtree stop` prints error and exits with non-zero code
- [ ] Daemon handles SIGTERM gracefully — stops DNS, stops proxy, removes PID file

### US-008: Unix Socket for Hot-Reload
**Priority:** 3
**Description:** As a developer, I want route changes to take effect immediately without restarting the daemon.

**Acceptance Criteria:**
- [ ] Daemon listens on a Unix socket at `~/.config/devtree/devtree.sock`
- [ ] `register` and `unregister` commands send a reload message to the socket after updating YAML
- [ ] Daemon receives reload message, re-reads `routes.yaml`, and calls `proxy.Reload()`
- [ ] If daemon is not running, `register`/`unregister` still update YAML (no error)

### US-009: Status Dashboard
**Priority:** 4
**Description:** As a developer, I want a web dashboard to see all my registered services and their status at a glance.

**Acceptance Criteria:**
- [ ] Dashboard served at `https://dashboard.devtree.test`
- [ ] Single HTML page embedded via Go `embed` package — no build tools, no JS framework
- [ ] JSON API at `https://dashboard.devtree.test/api/status` returns routing table and health info
- [ ] Dashboard renders: site list with clickable domain links, upstream ports, meta labels, health dots
- [ ] Auto-refreshes every 5 seconds via JS fetch
- [ ] Health check: basic TCP connect to each upstream; green dot if reachable, red if not

### US-010: Bootstrap System Setup
**Priority:** 5
**Description:** As a developer, I want a one-time setup command so that my OS is configured to route `.test` domains through devtree.

**Acceptance Criteria:**
- [ ] `devtree bootstrap` detects the current OS
- [ ] macOS: creates `/etc/resolver/test` with `nameserver 127.0.0.1` (prompts for sudo)
- [ ] macOS: runs `caddy trust` to install root CA in system trust store
- [ ] Linux: configures systemd-resolved or `/etc/resolv.conf` for `.test` TLD
- [ ] Linux: runs `caddy trust`
- [ ] Prints a summary of what was configured
- [ ] After bootstrap + start, `ping anything.test` resolves to `127.0.0.1`

## Functional Requirements

- FR-1: The system must persist routes in a YAML file at `~/.config/devtree/routes.yaml`
- FR-2: The system must write the YAML file atomically (temp file + rename) to prevent corruption
- FR-3: `devtree register <domain> --port <port>` must add or update a route entry
- FR-4: `devtree unregister <domain>` must remove a route entry
- FR-5: `devtree list` must display all registered routes in a table format
- FR-6: The embedded DNS server must resolve all `*.test` A queries to `127.0.0.1`
- FR-7: The embedded DNS server must return NXDOMAIN for non-`.test` queries
- FR-8: The reverse proxy must terminate TLS using Caddy's internal CA for each registered domain
- FR-9: The reverse proxy must forward requests to the upstream specified in the routing table
- FR-10: The reverse proxy config must be built programmatically — no Caddyfile on disk
- FR-11: `devtree start` must launch DNS + proxy in the background and write a PID file
- FR-12: `devtree stop` must send SIGTERM to the daemon and clean up the PID file
- FR-13: The daemon must listen on a Unix socket for reload commands
- FR-14: `register`/`unregister` must signal the daemon via Unix socket after updating YAML
- FR-15: The daemon must re-read the routing table and hot-reload Caddy on receiving a reload signal
- FR-16: The dashboard must be served at `https://dashboard.devtree.test` as an embedded HTML page
- FR-17: The dashboard API at `/api/status` must return the routing table and health status as JSON
- FR-18: The dashboard must perform basic TCP health checks on each upstream
- FR-19: `devtree bootstrap` must configure OS-level DNS resolution for `.test` TLD
- FR-20: `devtree bootstrap` must install Caddy's root CA into the system trust store

## Non-Goals

- No merging of `wtm` worktree commands into `devtree` (post-POC scope)
- No automatic service discovery — developers must explicitly register routes
- No support for non-`.test` TLDs
- No remote/team routing — this is strictly local (`127.0.0.1`)
- No Windows native support (WSL2 only)
- No active health checks with retry/circuit-breaker logic — health is display-only
- No authentication on the dashboard
- No custom TLS certificates — only Caddy's internal CA
- No Caddyfile support — config is always programmatic
- No log aggregation or request tracing from proxied services

## Technical Considerations

- **Project structure:** New CLI at `cmd/devtree/` alongside existing `cmd/wtm/` (untouched)
- **Key dependencies:** `github.com/miekg/dns` (DNS), `github.com/caddyserver/caddy/v2` (proxy + TLS)
- **Privileged ports:** DNS (53) and HTTPS (443) require elevated permissions; `devtree start` may need sudo or `setcap` on Linux
- **State paths:** `~/.config/devtree/routes.yaml` (routing table), `~/.config/devtree/devtree.pid` (PID), `~/.config/devtree/devtree.sock` (Unix socket)
- **Caddy embedding:** Caddy is used as a Go library, not as an external binary (except for `caddy trust` in bootstrap)
- **Cross-platform:** Core logic is OS-agnostic; only `bootstrap` has OS-specific branches (macOS, Linux)
- **Dashboard:** Single HTML file using `embed` — vanilla JS, no build step, no framework

## Success Metrics

- Developer can go from zero to `https://myapp.test` proxying to `localhost:8080` in under 3 commands (`bootstrap`, `start`, `register`)
- Route registration takes effect within 1 second (hot-reload via Unix socket)
- Dashboard loads in under 500ms and shows accurate health status
- All unit and integration tests pass across macOS and Linux
- DNS resolution for `*.test` works in browsers, `curl`, and `ping` after bootstrap

## Open Questions

- Should `devtree start` require sudo explicitly, or should it attempt to bind privileged ports and fail with a helpful message?
- Should the Unix socket protocol be a simple text protocol (e.g., `RELOAD\n`) or structured (e.g., JSON)?
- Should `devtree list` also show health status (requires daemon running) or just the static routing table?
- Should `devtree register` validate that the upstream port is actually listening before registering?
