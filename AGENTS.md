# AGENTS.md - Developer Guide for Worktree Manager

This guide is for AI coding agents and developers working on Worktree Manager.

## Project Overview

**Type:** Git Worktree Management CLI (Golang)  
**Purpose:** Simplifies Git worktree workflows with templates, hooks, and PR support  
**Language:** Go 1.22+  
**Repository:** https://github.com/vansdevcode/worktree-manager

## Build, Test & Run Commands

### Using mise (Recommended)

```bash
mise run build      # Build the binary (outputs: gh-wtm)
mise run test       # Run tests with coverage
mise run clean      # Clean build artifacts
mise run install    # Install both wtm and gh wtm
mise run help       # Show all available tasks
```

### Manual Commands

```bash
# Build
go build -o wtm cmd/wtm/*.go

# Test
go test -v ./...
go test -v -race -coverprofile=coverage.txt ./...

# Run
./gh-wtm --help
```

## Project Structure

```
worktree-manager/
├── cmd/wtm/             # CLI entry point and commands
│   ├── main.go            # Application entry point
│   ├── root.go            # Root cobra command
│   ├── init.go            # init command
│   ├── add.go             # add command (with PR support)
│   ├── rm.go              # rm command (with safety checks)
│   ├── ls.go              # ls command
│   └── pr.go              # pr command (shorthand)
├── internal/              # Private application code
│   ├── git/              # Git operations (clone, worktree, branch)
│   ├── worktree/         # Worktree utilities (slug, paths)
│   ├── template/         # Go template processing
│   ├── hook/             # Hook execution system
│   ├── pr/               # PR fetching (3-tier fallback)
│   └── config/           # Configuration management
├── pkg/ui/               # Public packages
│   └── output.go         # Colored terminal output
├── .github/workflows/    # CI/CD pipelines
│   ├── ci.yml           # Test and lint
│   └── release.yml      # Build and release binaries
├── .mise.toml           # mise configuration and tasks
└── install.sh           # Installation script
```

## Code Style Guidelines

### Go Conventions

**Naming:**
- Package names: lowercase, single word (`git`, `hook`, `template`)
- Exported functions: PascalCase (`AddWorktree`, `ProcessTemplates`)
- Private functions: camelCase (`generateSlug`, `runCommand`)
- Constants: PascalCase or SCREAMING_SNAKE_CASE
- Interfaces: noun or adjective (`Reader`, `Executable`)

**Error Handling:**
```go
// Always check errors
result, err := someFunc()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Use ui package for user-facing errors
if err := validate(); err != nil {
    ui.Error(fmt.Sprintf("Validation failed: %v", err))
    os.Exit(1)
}
```

**Cobra Commands:**
```go
var cmd = &cobra.Command{
    Use:     "commandname [flags]",
    Short:   "Brief description",
    Long:    `Detailed description...`,
    RunE:    runCommandName,  // Use RunE for error handling
}

func runCommandName(cmd *cobra.Command, args []string) error {
    // Implementation
    return nil
}
```

### File Organization

**internal/git/git.go:**
- All git operations use `exec.Command`
- Always specify `--git-dir` for bare repo operations
- Return wrapped errors with context

**internal/template/template.go:**
- Use Go's `text/template` package
- Template data: `{Branch, BranchSlug, Directory, RootDirectory}`
- Preserve file permissions when copying

**internal/hook/hook.go:**
- Hooks are executable scripts (any language)
- Set environment variables: `WT_BRANCH`, `WT_BRANCH_SLUG`, `WT_DIRECTORY`, `WT_ROOT_DIRECTORY`
- Execute with working directory = worktree directory

## Testing

### Unit Tests

- Test files: `*_test.go` alongside source files
- Use table-driven tests for multiple scenarios
- Mock external dependencies (git commands, filesystem)

```go
func TestGenerateBranchSlug(t *testing.T) {
    tests := []struct {
        name   string
        branch string
        want   string
    }{
        {"simple", "feature-test", "feature-test"},
        {"with slashes", "feature/test", "feature-test"},
        {"uppercase", "Feature/Test", "feature-test"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := GenerateBranchSlug(tt.branch)
            if got != tt.want {
                t.Errorf("got %q, want %q", got, tt.want)
            }
        })
    }
}
```

### Integration Tests

(To be added) - Will test full workflows with real git repos

## Key Features Implementation

### 1. PR Support (Three-Tier Fallback)

Located in `internal/pr/pr.go`:

1. **gh CLI** (best): `gh pr view <number> --json headRefName`
2. **GitHub API**: (TODO) Direct API calls
3. **Git refspec** (always works): `git fetch origin pull/<ID>/head:<branch>`

```go
func FetchPR(bareDir, prNumber string) (string, error) {
    // Try gh CLI first
    if branch, err := fetchPRWithGH(bareDir, prNumber); err == nil {
        return branch, nil
    }
    
    // Fall back to refspec (no dependencies)
    return fetchPRWithRefspec(bareDir, prNumber)
}
```

### 2. Template System

Located in `internal/template/template.go`:

- Uses Go's `text/template` package
- Files in `.worktree/templates/` are processed
- Variables: `{{.Branch}}`, `{{.BranchSlug}}`, `{{.Directory}}`, `{{.RootDirectory}}`

### 3. Hook System

Located in `internal/hook/hook.go`:

- Hooks: `.worktree/post-create`, `.worktree/post-delete`
- Must be executable (`chmod +x`)
- Any language (Bash, Python, etc.)
- Environment variables set: `WT_BRANCH`, `WT_BRANCH_SLUG`, `WT_DIRECTORY`, `WT_ROOT_DIRECTORY`

### 4. Safety Checks

Located in `cmd/wtm/rm.go`:

- Prevent removing current worktree
- Check for uncommitted changes
- Check for untracked files
- `--force` flag to bypass checks

## Adding New Commands

1. Create new file: `cmd/wtm/commandname.go`
2. Define cobra command with `RunE` function
3. Register in `cmd/wtm/root.go` via `init()`
4. Add tests in `cmd/wtm/commandname_test.go`
5. Update README.md with usage examples

## Dependencies

Managed via `go.mod`:

- `github.com/spf13/cobra` - CLI framework
- Standard library for everything else (no external dependencies for core functionality)

## CI/CD Pipeline

### GitHub Actions Workflows

**.github/workflows/ci.yml:**
- Runs on: push to main, pull requests
- Jobs: test, lint, build
- Go version: 1.22

**.github/workflows/release.yml:**
- Runs on: git tags (v*)
- Builds for: linux/darwin/windows, amd64/arm64
- Creates GitHub release with binaries
- Auto-generates release notes

### Creating a Release

```bash
git tag v0.1.0
git push origin v0.1.0
```

This triggers the release workflow which:
1. Runs tests
2. Builds binaries for all platforms
3. Creates GitHub release
4. Uploads binaries as release assets

## Common Patterns

### Git Operations

```go
// Always use bareDir for git commands
cmd := exec.Command("git", "--git-dir", bareDir, "worktree", "add", path, branch)
output, err := cmd.CombinedOutput()
if err != nil {
    return fmt.Errorf("git worktree add failed: %w\n%s", err, output)
}
```

### User Output

```go
import "github.com/vansdevcode/worktree-manager/pkg/ui"

ui.Info("Creating worktree...")
ui.Success("✓ Worktree created successfully")
ui.Warning("⚠ Uncommitted changes detected")
ui.Error("✗ Operation failed")
```

### Path Handling

```go
// Use filepath.Join for cross-platform paths
configPath := filepath.Join(rootDir, ".worktree")

// Use filepath.Abs for absolute paths
absPath, err := filepath.Abs(relativePath)
```

## Debugging

```bash
# Run with verbose output
go run ./cmd/wtm init myrepo -v

# Run specific test
go test -v ./internal/git -run TestConvertGitHubFormat

# Check race conditions
go test -race ./...
```

## Documentation Requirements

When adding/modifying features:

1. Update command help text in cobra command definition
2. Add usage examples to README.md
3. Update ROADMAP.md if it's a planned feature
4. Document template variables/hook environment variables if changed

## Commit Message Style

Follow conventional commits:

```
feat: add pr command for quick PR checkout
fix: handle spaces in branch names correctly
docs: update README with new template variables
refactor: simplify git operations code
test: add tests for branch slug generation
chore: update dependencies
```

## Safety & Security

**Never:**
- Execute arbitrary user input without validation
- Remove directories outside worktree root
- Skip safety checks without explicit `--force` flag
- Hardcode credentials or tokens

**Always:**
- Validate file paths are within expected directories
- Check for uncommitted changes before destructive operations
- Provide clear error messages with actionable steps
- Quote command arguments to prevent injection

## Future Proofing

When contributing, consider:
- Go version compatibility (target 1.22+)
- Platform compatibility (Linux, macOS, Windows)
- Git version compatibility (2.5+)
- Graceful degradation when optional features unavailable
- Clear error messages when dependencies missing
