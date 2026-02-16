# Worktree Manager - Roadmap

This document outlines the current status and future plans for Worktree Manager.

## ✅ Completed Features (v0.1.0)

### Core Functionality
- ✅ **Go CLI Implementation** - Migrated from Bash to Go for better performance and cross-platform support
- ✅ **Bare Repository Management** - Clone and initialize bare repos in `.bare/` directory
- ✅ **Worktree Operations** - Add, remove, list worktrees
- ✅ **Branch Management** - Create new branches or use existing ones
- ✅ **PR Support** - Three-tier fallback (gh CLI → GitHub API → git refspec)
- ✅ **Template System** - Go templates for `.worktree/files/` with `{{.Variables}}`
- ✅ **Hook System** - Executable scripts (post-create, post-delete) with environment variables
- ✅ **Safety Checks** - Prevent data loss (uncommitted changes, untracked files)
- ✅ **Dual Installation** - Standalone `wtm` + GitHub CLI extension `gh wtm`
- ✅ **mise Integration** - Build tasks and Go version management
- ✅ **CI/CD Pipeline** - Automated testing and releases via GitHub Actions
- ✅ **Install Script** - One-liner installation from GitHub

### Commands
- ✅ `init` - Initialize worktree repository (clone or create new)
- ✅ `add` - Add worktree from branch or PR
- ✅ `rm` - Remove worktree with safety checks
- ✅ `ls` - List all worktrees
- ✅ `pr` - Shorthand for adding PR worktree

### Developer Experience
- ✅ Comprehensive documentation (README, AGENTS.md, CLI-PLAN.md)
- ✅ Unit tests with good coverage
- ✅ Clean codebase structure
- ✅ Professional error handling and colored output

## 🚧 In Progress

### Testing
- ⏳ Integration tests (E2E testing with real git repos)
- ⏳ Increase test coverage to 90%+

## 📋 Planned Features

### High Priority

#### `switch` Command
**Status:** Planned  
**Description:** Quick navigation between worktrees  
**Use Case:** Switch to a different worktree directory
```bash
wt switch feature-auth    # cd to feature-auth worktree
wt switch -               # Switch to previous worktree
```

#### `status` Command
**Status:** Planned  
**Description:** Show status of all worktrees  
**Use Case:** See which worktrees have uncommitted changes, untracked files, or are out of sync
```bash
wt status
# main/          ✓ Clean
# feature-auth/  ⚠ 2 uncommitted files
# bugfix-login/  ⚠ 3 untracked files
```

### Medium Priority

#### `sync` Command
**Status:** Planned  
**Description:** Fetch/rebase/merge worktrees  
**Use Case:** Keep worktrees up to date with remote
```bash
wt sync                   # Sync all worktrees
wt sync feature-auth      # Sync specific worktree
wt sync --rebase          # Rebase instead of merge
```

#### `archive` Command
**Status:** Planned  
**Description:** Archive worktree before deletion  
**Use Case:** Create backup before removing worktree
```bash
wt archive feature-auth   # Create tar.gz before deletion
wt archive --keep         # Archive but don't delete
```

#### `clone` Command
**Status:** Planned  
**Description:** Create worktree from existing worktree's branch  
**Use Case:** Work on same branch in multiple directories
```bash
wt clone feature-auth feature-auth-2
```

### Low Priority

#### Interactive Mode
**Status:** Planned  
**Description:** Use fzf or similar for interactive selection  
**Use Case:** Browse and select worktrees interactively
```bash
wt switch --interactive   # Show interactive list
wt rm --interactive       # Select worktree to remove
```

#### Per-Worktree Configuration
**Status:** Planned  
**Description:** `.wt.yml` configuration file  
**Use Case:** Customize behavior per worktree
```yaml
# .wt.yml
templates:
  enabled: true
  source: .worktree/templates
hooks:
  post-create: .worktree/hooks/setup
variables:
  PROJECT_NAME: "my-app"
```

#### Branch Naming Conventions
**Status:** Planned  
**Description:** Enforce branch naming patterns  
**Use Case:** Maintain consistent branch names
```yaml
# .wt.yml
naming:
  pattern: "{type}/{ticket}-{description}"
  types: [feature, bugfix, hotfix, chore]
```

### Template Enhancements

#### Additional Variables
**Status:** Planned  
**Variables:**
- `{{.Date}}` - Current date
- `{{.User}}` - Git user name
- `{{.Email}}` - Git user email
- `{{.BaseBranch}}` - Branch created from
- `{{.BranchUpper}}` - Uppercase branch name
- `{{.BranchLower}}` - Lowercase branch name

#### Conditional Templates
**Status:** Planned  
**Description:** Process templates based on branch patterns  
**Use Case:** Different templates for feature/bugfix branches
```yaml
# .wt.yml
templates:
  - pattern: "feature/*"
    source: .worktree/templates/feature
  - pattern: "bugfix/*"
    source: .worktree/templates/bugfix
```

#### Template Functions
**Status:** Planned  
**Description:** Built-in template functions  
**Use Case:** Transform variables in templates
```
{{.Branch | upper}}        # FEATURE-AUTH
{{.Branch | replace "/" "-"}}  # feature-auth
{{.Date "2006-01-02"}}    # Custom date format
```

### GitHub Integration

#### Auto-Create Draft PR
**Status:** Planned  
**Description:** Create draft PR when creating worktree  
**Use Case:** Start PR early for visibility
```bash
wt add main feature-auth --pr
# Creates worktree + opens draft PR
```

#### Link to Issues
**Status:** Planned  
**Description:** Auto-link to GitHub issues  
**Use Case:** Connect worktree to issue tracking
```bash
wt add main feature-auth --issue 123
# Links PR to issue #123
```

#### PR Metadata
**Status:** Planned  
**Description:** Use PR title/description in templates  
**Use Case:** Auto-generate documentation from PR info
```
# PR #{{.PRNumber}}: {{.PRTitle}}
Created by: {{.PRAuthor}}
```

### Docker Integration

#### Auto-Start Containers
**Status:** Planned  
**Description:** Start Docker containers when creating worktree  
**Use Case:** Isolated development environments
```bash
# post-create hook
docker-compose -f .worktree/docker-compose.yml up -d
```

### IDE Integration

#### Workspace File Generation
**Status:** Planned  
**Description:** Generate VS Code/IntelliJ workspace files  
**Use Case:** Pre-configured IDE settings per worktree
```
# .worktree/templates/.vscode/settings.json
{
  "workbench.name": "{{.Branch}}"
}
```

## 🎯 Version Milestones

### v0.1.0 (Current)
- Core CLI functionality
- PR support with fallback
- Template and hook systems
- Dual installation modes
- CI/CD pipeline

### v0.2.0 (Next)
- `switch` command
- `status` command
- Integration tests
- 90%+ test coverage

### v0.3.0
- `sync` command
- `archive` command
- Interactive mode
- Per-worktree configuration

### v1.0.0
- All planned features complete
- Comprehensive documentation
- Production-ready stability
- Package manager integration (Homebrew, apt, etc.)

## 📝 Notes

### Design Principles
- **Simplicity First** - Easy commands for common workflows
- **Safety** - Prevent data loss with safety checks
- **Flexibility** - Support various workflows and preferences
- **No Lock-in** - Standard git worktrees, can be used without tool
- **Fast** - Go performance, minimal dependencies

### Breaking Changes
We will avoid breaking changes where possible. If necessary:
- Deprecation warnings in one version
- Breaking change in next major version
- Migration guides provided

### Community Feedback
Feature priorities may change based on user feedback and usage patterns.

## 🤝 Contributing

See feature requests and discuss priorities in GitHub Issues.

Want to implement a feature? Check AGENTS.md for development guidelines.
