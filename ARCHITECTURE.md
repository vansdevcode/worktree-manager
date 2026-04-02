# Architecture & Implementation Plan

This document captures architecture decisions and implementation plans for **wtm** (worktree manager) and **devtree** (dev environment orchestrator). Use it as context when working on either tool.

## Repository Structure

This repo contains two CLI tools:

- **wtm** (`cmd/wtm`) — Git worktree manager. Branch: `main`.
- **devtree** (`cmd/devtree`) — Dev environment orchestrator with daemon, DNS, reverse proxy, Nix-based services. Branch: `feat/devtree`.

Both are written in Go. They are **separate tools** that integrate via hooks — not merged into a single binary.

---

## wtm — Worktree Manager

### Current State

wtm manages git worktrees with a bare repository pattern:

```
myrepo/
├── .git/               # Bare repository (git clone --bare)
├── .worktree/          # Hooks and template files (being replaced by .wtm.toml)
├── main/               # Default branch worktree
├── feature-123/        # Feature branch worktree
└── fix-bug-456/        # Bug fix worktree
```

- `FindRoot()` in `internal/config/config.go` identifies a wtm repo by looking for the `.worktree/` directory.
- Bare repo is cloned directly into `.git/` (no `.git` file indirection).
- Commands: `wtm init`, `wtm add`, `wtm ls`, `wtm rm`.

### Next: Replace `.worktree/` Directory with `.wtm.toml`

**Goal:** Replace the `.worktree/` directory convention with a single config file, similar to `mise.toml`.

#### Config Resolution

1. **`.wtm.toml`** — committed in the repo, version-controlled (team defaults)
2. **`.wtm.local.toml`** — at root next to `.git/`, developer overrides (gitignored)

Merge strategy: **deep merge at the TOML table level, local values win.** Missing keys in local fall back to committed defaults.

`.wtm.local.toml` should be automatically added to `.git/info/exclude` on `wtm init` so it's ignored without touching `.gitignore`.

#### Reading Config

- Read `.wtm.toml` from the **worktree being operated on** (not via `git show`). The developer sees what file is active for their branch.
- Read `.wtm.local.toml` from the root directory (next to `.git/`).
- Deep merge, local wins.

#### Example `.wtm.toml`

```toml
[hooks]
post-create = """
npm install
cp .env.example .env
"""
pre-delete = "echo 'cleaning up'"

[files]
".env" = { template = ".wtm/env.tmpl" }
"config/local.yml" = { template = ".wtm/local.yml.tmpl" }
"init.sql" = { copy = ".wtm/init.sql" }

[vars]
db_host = "localhost"
db_port = "5432"
```

#### Example `.wtm.local.toml`

```toml
[vars]
db_host = "my-remote-db.internal"

[hooks]
post-create = "npm install && npm run setup:local"
```

#### Implementation Notes

- `FindRoot()` currently looks for `.worktree/` directory. After migration, it should look for `.wtm.toml` in the repo (read from bare git) OR `.wtm.local.toml` at root. Consider a transitional period supporting both.
- Template files referenced in `[files]` live in a `.wtm/` directory inside the repo (committed, tracked).
- Hook scripts can be inline (in TOML) or file references in `.wtm/hooks/`.
- The `.worktree/` directory and its handling code can be removed once migration is complete.

---

## devtree — Dev Environment Orchestrator

### Architecture Overview

devtree is an environment orchestrator with pluggable backends. The daemon runs on the host and manages environments.

```
devtree daemon (host)
├── DNS resolver (built-in, port 10053)
├── Reverse proxy (embedded Caddy)
├── Dashboard (localhost, browser-accessible)
├── MCP server (for AI tool integration)
└── Environments
    ├── nix backend (local, default — build this first)
    ├── docker backend (future)
    └── remote backend (future)
```

### Current State (branch: feat/devtree)

- Daemon with suture-based process supervision (health checks + dependencies)
- Simple DNS resolver on port 10053, auto-creates `/etc/resolver/test`
- Embedded Caddy for reverse proxy
- Dashboard on localhost showing registered domains and service statuses
- Reads from `devtree.yaml`
- Nix-based environment management

### DNS

- Uses `.test` TLD — IANA-reserved for testing, no conflicts.
- **Do NOT use `.local`** — reserved for mDNS/Bonjour, causes 5-second DNS delays on macOS.
- **Do NOT use `.localhost`** — wildcard subdomains behave inconsistently.
- macOS: `/etc/resolver/test` (auto-created by devtree).
- Linux: will need systemd-resolved or NetworkManager dnsmasq configuration (future).
- WSL2: Windows-side DNS configuration (future).

### Certificates (to implement)

Use **mkcert** for trusted local HTTPS:

1. `mkcert -install` — installs local CA into system trust stores (one-time setup)
2. `mkcert "*.project-name.test"` — generates wildcard cert trusted by browsers
3. Configure Caddy to use mkcert-generated certs instead of its own self-signed ones

Flow:
- On project register: generate `*.project-name.test` wildcard cert via mkcert, configure Caddy
- On project remove: clean up certs
- Wildcard certs cover all branch subdomains: `my-branch.project-name.test`

mkcert is a single binary — can be vendored or installed via Nix. Works on macOS, Linux, Windows.

### Domain Patterns

For worktree-managed projects:
- `my-branch.project-name.test` — branch-specific domain
- Wildcard `*.project-name.test` covers all branches
- For multi-service projects: `my-branch.project-name-api.test` (flat) preferred over `my-branch.api.project-name.test` (nested) to keep wildcard certs simple

### Process Supervision

**Decision: Keep suture. Do not adopt process-compose.**

Rationale:
- Suture already works with health checks and dependencies
- devtree supervises a known, small set of services (PHP-FPM, MySQL, Redis, queue workers)
- process-compose would add a dependency, translation layer, and dashboard integration headaches
- The daemon manages all project services — when devtree stops, everything stops cleanly
- For docker backend (future), the container handles its own process supervision

### MCP Server

Add an MCP server to the devtree daemon so AI tools (Claude Code, etc.) can:
- List environments and their status
- Start/stop environments
- Register/unregister domains
- View service logs
- Spin up ad-hoc environments with required tools (PHP, MySQL, Redis, etc.)

This is a key differentiator — AI-managed dev environments.

### Future Backends

#### Docker Backend
- One container per environment (nixos image or minimal image with nix)
- All environment processes run inside the container
- devtree daemon still runs on host for DNS + proxy to container services
- Developer doesn't need nix installed on host for this backend

#### Remote Backend
- SSH/cloud-based environments
- Same config, proxy over tunnel
- Like docker context but for devtree

**Important:** These are future work. Build nix backend first, ship it, then add backends.

### Extensibility (Future, v2+)

Considering gh-cli style extensions: custom scripts/binaries in `~/.config/share/devtree/extensions`. Possibly nix-based extensions where a `.nix` file outputs scripts, binaries, languages, and services (inspired by devenv).

**Do not build this now.** Ship the Laravel/PHP experience first. Once 5-10 people use it daily, the extension API will design itself from real usage patterns.

---

## Integration: wtm + devtree

The tools integrate via **hooks**, not shared code:

1. Developer runs `wtm add main feature-x`
2. wtm's `post-create` hook calls devtree to register `feature-x.project-name.test`
3. devtree provisions the environment (services, DNS, proxy, certs)
4. Developer navigates to `feature-x.project-name.test` — full HTTPS, no warnings

On `wtm rm feature-x`:
1. wtm's `pre-delete` hook calls devtree to deregister the domain
2. devtree tears down the environment

Example `.wtm.toml` hook:
```toml
[hooks]
post-create = "devtree env register --branch {{ .Branch }} --project {{ .Vars.project_name }}"
pre-delete = "devtree env deregister --branch {{ .Branch }} --project {{ .Vars.project_name }}"
```

---

## Priority / Build Order

### Phase 1: Ship for Laravel at Kirschbaum
1. **wtm**: Implement `.wtm.toml` / `.wtm.local.toml` config (replace `.worktree/` directory)
2. **devtree**: mkcert integration for trusted HTTPS (quick win over Herd)
3. **devtree**: wtm hook integration (automatic domain registration on worktree create/remove)
4. **devtree**: MCP server (AI demo differentiator)
5. Common Laravel services: PHP-FPM, MySQL, Redis, Mailpit, queue workers

### Phase 2: Polish & Extend
6. Dashboard improvements
7. Docker backend
8. Linux / WSL2 DNS support
9. Additional services (OpenSearch, RustFS, etc.)

### Phase 3: Community
10. Extension/plugin system
11. Remote environments
12. Multi-language support beyond PHP/Laravel

---

## Naming

- **wtm** — worktree manager. Name is fine, short, clear.
- **devtree** — temporary name. Considered alternatives: `devrouter`, `hive`, `forge`, `denv`, `loom`. Pick something short, `brew install`-able, and not taken on GitHub before public launch.

---

## Technical Reference

### Key Files (wtm)
- `internal/config/config.go` — `FindRoot()`, `GetBareDir()`, config path helpers
- `cmd/wtm/init.go` — repo initialization (bare clone into `.git/`)
- `cmd/wtm/add.go` — worktree creation, PR checkout
- `cmd/wtm/rm.go` — worktree removal with safety checks
- `cmd/wtm/ls.go` — list worktrees
- `internal/git/git.go` — git operations (all take `bareDir` parameter, use `--git-dir`)
- `internal/hook/` — hook execution
- `internal/template/` — template file processing

### Key Files (devtree, branch feat/devtree)
- `cmd/devtree/` — CLI entry point
- `internal/daemon/` — daemon with suture supervision
- `internal/dns/` — DNS resolver
- `internal/proxy/` — embedded Caddy reverse proxy
- `internal/dashboard/` — web dashboard
- `internal/routing/` — domain routing logic
