# Prompt: devtree — Local Development CLI with Worktree Management and Routing

## Context

I have an existing Go CLI called `wtm` (worktree manager) that manages git worktrees with hooks and templates. I'm evolving it into `devtree` — a single binary that combines worktree management with a local development routing layer (DNS, TLS, reverse proxy, dashboard).

The routing layer is **stack-agnostic**. It does not manage runtimes, servers, or dependencies. Each project is responsible for starting its own dev server on a local port (FrankenPHP, Node, Vite, Go, etc., managed by the project's own mise tasks, nix devshell, or scripts). `devtree` simply routes `*.test` domains to the correct upstream.

The worktree management and routing are tightly coupled: creating a worktree can automatically register a route; removing a worktree can unregister it.

## Architecture

The binary embeds three components alongside the existing worktree management logic:

### 1. DNS Server (via `miekg/dns`)

A lightweight DNS responder that resolves all `*.test` queries to `127.0.0.1`. Listens on port 53 (UDP and TCP).

On macOS, a one-time bootstrap creates `/etc/resolver/test` with `nameserver 127.0.0.1` so the OS forwards `.test` lookups to this embedded DNS. On Linux/WSL2, equivalent systemd-resolved or `/etc/resolv.conf` configuration is needed.

### 2. Reverse Proxy (via embedded Caddy)

Caddy is embedded as a Go library (`github.com/caddyserver/caddy/v2`). It handles:

- **TLS termination** with `tls internal` — Caddy generates a local root CA and per-site certificates. One `caddy trust` invocation installs the root CA in the system trust store. All `*.test` sites get valid HTTPS automatically.
- **Reverse proxying** — routes each registered domain to its project's upstream port.
- **Dashboard hosting** — serves a built-in status dashboard (see below).

Caddy listens on ports 80 and 443. Configuration is managed programmatically via `caddy.Load()` — no Caddyfile on disk.

### 3. Routing Table

A YAML state file maintains the mapping of domains to upstreams:

```yaml
sites:
  feature-x.project-a.test:
    upstream: localhost:44085
    meta:
      stack: "php:8.5"
      framework: "laravel:11"
  main.project-b.test:
    upstream: localhost:3000
    meta:
      stack: "node:22"
  api.project-c.test:
    upstream: localhost:8080
    meta:
      stack: "go:1.23"
```

The `meta` field is optional freeform key-value pairs for display on the dashboard.

## CLI Commands

### Daemon

- **`devtree start`** — Starts the long-running daemon (DNS + Caddy). Manages its own PID file for lifecycle control.
- **`devtree stop`** — Stops the daemon.
- **`devtree bootstrap`** — One-time system setup: creates `/etc/resolver/test` on macOS (requires sudo), configures DNS forwarding on Linux/WSL2, and runs `caddy trust` to install the root CA.

### Routing

- **`devtree register <domain> --port <port> [--meta key:value ...]`** — Registers a new route, adds it to the routing table, and hot-reloads Caddy via `caddy.Load()`.
- **`devtree unregister <domain>`** — Removes a route and reloads Caddy.
- **`devtree list`** — Prints the current routing table to stdout.

### Worktree Management

These commands carry over from the existing `wtm` tool:

- **`devtree new <project> <branch>`** — Creates a git worktree at `~/work/<project>/<branch>`, runs project hooks/templates, and optionally auto-registers a route (`<branch>.<project>.test`) if the project is configured for it.
- **`devtree rm <project> <branch>`** — Removes a worktree and unregisters its route if one was registered.
- Other existing `wtm` commands (list worktrees, run hooks, etc.) are preserved under `devtree`.

The worktree commands can integrate with routing automatically based on project configuration (e.g., a `.devtree.yaml` at the project root that declares the port convention and meta labels), or routing can be managed independently via `devtree register`/`devtree unregister`.

## Dashboard

The embedded Caddy serves a dashboard at `https://dashboard.dev.test`. It is a single static HTML page (embedded in the Go binary via `embed`) that:

- Lists all registered sites as clickable `https://` links.
- Shows the upstream port for each site.
- Displays optional `meta` labels (stack, framework, etc.).
- Shows a health indicator (green/red dot) for each upstream, determined by the dashboard JS fetching each upstream on page load.
- Auto-refreshes on a simple interval (e.g., every 5 seconds).

The dashboard fetches its data from a JSON API endpoint (e.g., `https://dashboard.dev.test/api/status`) that returns the current routing table with health status.

## Cross-Platform

The tool must work on macOS, Linux, and WSL2. The DNS bootstrap step (`devtree bootstrap`) is the only OS-specific part. Everything else is identical across platforms.

## Project Integration Pattern

Projects integrate with `devtree` in two ways:

### Automatic (via worktree commands)

When a project has a `.devtree.yaml` at its root, `devtree new` and `devtree rm` handle registration/unregistration automatically as part of the worktree lifecycle.

### Manual (via register/unregister)

Any project — even those not managed as worktrees — can register itself:

1. Start its own dev server on any available port.
2. Call `devtree register <domain> --port <port>`.
3. Call `devtree unregister <domain>` on shutdown.

This keeps `devtree` completely decoupled from any runtime or framework.

## Tech Stack

- **Language**: Go
- **CLI framework**: Cobra
- **Embedded Caddy**: `github.com/caddyserver/caddy/v2`
- **DNS**: `github.com/miekg/dns`
- **Dashboard**: Single HTML page embedded via Go's `embed` package
- **State file**: YAML (`gopkg.in/yaml.v3`)
- **Distribution**: Single static binary, installable via Nix flake or mise

## Codebase Structure

The existing `wtm` worktree logic should be refactored into an internal Go package so that the worktree commands and the routing commands coexist cleanly under a single Cobra root command. The worktree package has no dependency on Caddy or DNS — it's pure git/filesystem logic. The routing package has no dependency on worktree logic. The CLI layer wires them together (e.g., `devtree new` calls worktree creation, then optionally calls route registration).
