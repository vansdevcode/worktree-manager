## Codebase Patterns
- Use `gopkg.in/yaml.v3` for YAML marshaling/unmarshaling
- Pre-commit hook runs `golangci-lint` — must use `_ =` for intentionally ignored error returns (errcheck)
- Atomic file writes: use `os.CreateTemp` + `os.Rename` pattern
- Module path: `github.com/vansdevcode/worktree-manager`
- Existing CLI is `cmd/wtm/`, new devtree CLI goes in `cmd/devtree/`
- Config dir convention: `~/.config/devtree/` for routing state files
- DNS package uses `mdns` import alias for `github.com/miekg/dns` to avoid collision with package name
- miekg/dns `Server.NotifyStartedFunc` callback signals when server is ready to accept connections

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
