# Worktree Manager

A powerful CLI tool for managing Git worktrees with ease. Use it standalone as `wtm` or as a GitHub CLI extension `gh wtm`.

**Now written in Go** for better performance, type safety, and cross-platform support!

## What are Git Worktrees?

Git worktrees allow you to have multiple working directories for a single repository. This is incredibly useful when you need to:

- Work on multiple branches simultaneously
- Quickly switch between features without stashing
- Run tests on one branch while developing on another
- Review PRs without disrupting your current work

`wtm` makes working with worktrees simple and intuitive, with added features like templates and safety checks.

## Installation

### Recommended: Using install.sh

The recommended installation method uses the provided `install.sh` script:

```bash
# Download and run the installer
curl -fsSL https://raw.githubusercontent.com/vansdevcode/worktree-manager/main/install.sh | bash

# Or clone and run locally
git clone https://github.com/vansdevcode/worktree-manager.git
cd worktree-manager
./install.sh
```

This installs:

- **`wtm`** - Standalone command in `~/.local/bin` (works independently)
- **`gh wtm`** - GitHub CLI extension (if you have `gh` installed)

The installer automatically downloads the latest release and sets up both commands correctly.

### Manual Installation / Development

For development or manual installation with `mise`:

```bash
git clone https://github.com/vansdevcode/worktree-manager.git
cd worktree-manager
mise run install
```

### Alternative: Direct GitHub Extension Install

Installing directly from GitHub will use the repository name:

```bash
# This installs as 'gh worktree-manager' (not 'gh wtm')
gh extension install vansdevcode/worktree-manager
```

**Tip:** Use the `install.sh` script instead to get the correct `gh wtm` command name.

**Note:** Ensure `~/.local/bin` is in your `$PATH` for the standalone `wtm` command:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

## Quick Start

```bash
# Initialize a repository for worktree management
wtm init myorg/myrepo

# Add a new worktree for a feature
cd myrepo
wtm add main feature-awesome-thing

# Work in your new worktree
cd feature-awesome-thing
# ... make changes ...

# List all worktrees
wtm ls

# Remove a worktree when done
wtm rm feature-awesome-thing
```

**Note:** All commands can also be used with `gh wtm` if you prefer the GitHub CLI extension.

## Commands

All commands can be run with `wtm` (standalone) or `gh wtm` (GitHub CLI extension). Examples below use `wtm`.

### `wtm init`

Initialize a new worktree-managed repository.

```bash
wtm init <repo> [dir] [--new] [--no-hooks]
```

**Arguments:**

- `repo` - Repository in format `org/repo` or full git URL
- `dir` - Directory name (optional, defaults to repo name)
- `--new` - Create a new repository instead of cloning
- `--no-hooks` - Skip running the post-create hook

**Examples:**

```bash
# Clone an existing repository
wtm init myorg/myrepo

# Clone with custom directory name
wtm init myorg/myrepo my-project

# Clone using full git URL
wtm init git@github.com:myorg/myrepo.git

# Create a new repository
wtm init myorg/newproject --new
```

**What it does:**

1. Creates a directory structure with a bare repository in `.bare/`
2. Clones the repository (or creates a new one with `--new`)
3. Automatically creates a worktree for the default branch
4. Initializes submodules if present
5. Processes files from `.worktree/files/` if it exists

### `wtm add`

Add a new worktree from a base branch or pull request.

```bash
wtm add <base-branch> [new-branch] [directory] [--no-hooks]
# OR
wtm add pr/<number> [custom-name] [--no-hooks]
```

**Arguments:**

- `base-branch` - Branch, tag, or commit to base from (e.g., `main`, `origin/develop`, `v1.0.0`)
- `new-branch` - Name for the new branch (optional, uses base branch if omitted)
- `directory` - Custom directory name (optional, defaults to branch slug)
- `pr/<number>` - Pull request number to checkout (creates directory `pr-<number>`)
- `pr/<number>/<custom-name>` - Pull request with custom directory name
- `--no-hooks` - Skip running the post-create hook

**Examples:**

```bash
# Add worktree from main branch (creates new branch)
wtm add main feature-123

# Add worktree with custom directory
wtm add main feature-123 my-custom-dir

# Add existing remote branch
wtm add origin/develop

# Add from a tag
wtm add v1.0.0 hotfix-security

# Add from a specific commit
wtm add a1b2c3d experiment

# Checkout a pull request (creates directory pr-123 with branch from PR)
wtm add pr/123

# Checkout PR with custom directory name
wtm add pr/123/review

# Skip hooks when adding
wtm add main feature-456 --no-hooks
```

**What it does:**

1. Creates a new branch from the specified base (or checks out existing branch)
2. Creates a new directory (defaults to branch slug if not specified)
3. Initializes submodules automatically
4. Processes files from `.worktree/files/` (templates with `.tmpl` extension are processed, others copied as-is)
5. Runs post-create hook (unless `--no-hooks` is used)
6. Ready to start working immediately

**Pull Request Support:**

`wtm` has first-class support for working with GitHub pull requests with a **three-tier fallback system**:

- **Quick PR checkout**: Use `pr/123` syntax to checkout any pull request
- **Custom naming**: Add a custom directory name with `pr/123/my-name`
- **Works without gh CLI**: Falls back to git refspec if GitHub CLI is unavailable

The extension automatically tries these methods in order:

1. **GitHub CLI** (`gh`) - Fetches PR metadata including title and description
2. **GitHub API** - Uses git credentials for API access (future enhancement)
3. **Git refspec** - Direct `git fetch origin pull/$ID/head` (always works, no auth needed)

This means PR support works even without `gh` CLI installed!

### `wtm rm`

Remove a worktree and optionally its branch.

```bash
wtm rm [branch-name] [--force] [--delete-branch] [--no-hooks]
```

**Arguments:**

- `branch-name` - Name of the branch/worktree to remove (optional, uses current directory if not specified)
- `--force`, `-f` - Force removal even with uncommitted changes
- `--delete-branch`, `-d` - Also delete the git branch after removing worktree
- `--no-hooks` - Skip running the post-delete hook

**Examples:**

```bash
# Remove a specific worktree
wtm rm feature-123

# Force remove with uncommitted changes
wtm rm old-branch --force

# Remove worktree and delete the branch
wtm rm feature-123 --delete-branch

# Remove without running hooks
wtm rm feature-123 --no-hooks
```

**Safety features:**

- Checks for uncommitted changes
- Warns about untracked files
- Prevents removing worktree you're currently in
- Runs post-delete hook before removal (unless `--no-hooks` is used)
- Use `--force` to bypass safety checks

### `wtm ls`

List all worktrees in the current repository.

```bash
wtm ls
```

**Example output:**

```
Worktrees in /Users/you/projects/myrepo:

/Users/you/projects/myrepo/.bare           (bare)
/Users/you/projects/myrepo/main            a1b2c3d [main]
/Users/you/projects/myrepo/feature-123     d4e5f6g [feature-123]
```

### `wtm pr`

Convenience shorthand for adding a pull request worktree.

```bash
wtm pr <number> [directory]
```

**Arguments:**

- `number` - Pull request number
- `directory` - Optional custom directory name

**Examples:**

```bash
# Quick PR checkout (creates pr-123 directory)
wtm pr 123

# PR with custom directory
wtm pr 123 review-fixes
```

This is equivalent to `wtm add pr/<number>` but shorter for quick PR checkouts.

## Template Support

One of the most powerful features of `wtm` is **Go template support**. Templates allow you to automatically set up configuration files for each worktree with dynamic variable replacement.

### Setting Up Templates

Create a `.worktree/files/` directory in your repository root:

```bash
mkdir -p .worktree/files/
```

Files in this directory are handled based on their extension:

- **Files ending with `.tmpl`**: Processed as Go templates with variable replacement, then saved without the `.tmpl` extension
- **Files without `.tmpl`**: Copied as-is without any processing (safe for SQL files, scripts with `{{` syntax, etc.)

This allows you to mix template files and regular files in the same directory safely.

### Available Variables

Template files (ending with `.tmpl`) use Go template syntax with these context variables:

- `{{ .Branch }}` - The branch name of the worktree
- `{{ .Directory }}` - The absolute path to the worktree directory
- `{{ .RootDirectory }}` - The absolute path to the repository root (where `.bare` is located)

### Example Templates

**`.worktree/files/.env.tmpl`** (processed as template, saved as `.env`)

```bash
APP_URL={{ .Branch }}.myapp.test
APP_NAME={{ .Branch }}
APP_ENV=local
DATABASE_NAME=myapp_{{ .Branch }}
```

**`.worktree/files/config.json.tmpl`** (processed as template, saved as `config.json`)

```json
{
  "branch": "{{ .Branch }}",
  "directory": "{{ .Directory }}",
  "rootDirectory": "{{ .RootDirectory }}"
}
```

**`.worktree/files/.vscode/settings.json.tmpl`** (processed as template, saved as `.vscode/settings.json`)

```json
{
  "terminal.integrated.cwd": "{{ .Directory }}",
  "git.defaultBranchName": "{{ .Branch }}"
}
```

**`.worktree/files/init.sql`** (copied as-is, NOT processed)

```sql
-- This file contains {{ }} syntax and will be copied without processing
CREATE TABLE users (
  id INT PRIMARY KEY,
  data JSONB DEFAULT '{{}}'::jsonb
);
```

### Directory Structure in Files

You can create subdirectories in `.worktree/files/` and they'll be preserved:

```
.worktree/files/
├── .env.tmpl              # Processed as template → .env
├── config.json.tmpl       # Processed as template → config.json
├── init.sql               # Copied as-is (safe for {{ }} syntax)
├── setup.sh               # Copied as-is
├── .vscode/
│   └── settings.json.tmpl # Processed as template → .vscode/settings.json
└── config/
    └── local.yml.tmpl     # Processed as template → config/local.yml
```

Files with `.tmpl` extension are processed and saved without the extension. Other files are copied as-is.

## Hook Support

Hooks allow you to run custom scripts during worktree lifecycle events, similar to Git hooks. This is useful for automating setup and cleanup tasks.

**Hooks now support Go templates!** You can use the same template syntax and functions (including `gomplate`) in your hooks as you do in template files.

### Setting Up Hooks

Create a `.worktree/` directory in your repository root:

```bash
mkdir .worktree
```

Add scripts with specific hook names in this directory and start with the special shebang:

```bash
#!/usr/bin/env -S wtm hook
```

This shebang tells `wtm` to process the script as a Go template before executing it.

### Available Hooks

- **`post-create`** - Runs after a worktree is created
- **`post-delete`** - Runs before a worktree is deleted

### Available Template Variables

Hooks have access to the same Go template variables as template files:

- `{{ .Branch }}` - The branch name of the worktree
- `{{ .Directory }}` - Absolute path to the worktree directory
- `{{ .RootDirectory }}` - Absolute path to the repository root (where `.bare` is located)

### Available Template Functions

Hooks have access to all [gomplate functions](https://docs.gomplate.ca/functions/), including:

- `{{ .Branch | strings.Slug }}` - Convert branch name to URL-friendly slug
- `{{ .Branch | strings.ReplaceAll "/" "-" }}` - Replace characters in strings
- `{{ env.Getenv "HOME" }}` - Access environment variables
- And many more...

### Example Hooks

**`.worktree/post-create`**

```bash
#!/usr/bin/env -S wtm hook
set -e

echo "Setting up worktree for branch: {{ .Branch }}"

# Use gomplate's strings.Slug for URL-safe slug (much easier than sed!)
SLUG="{{ .Branch | strings.Slug }}"
echo "URL-safe slug: $SLUG"

# Install dependencies
npm install

# Copy environment file
if [ -f .env.example ]; then
    cp .env.example .env
    echo "Created .env from .env.example"
fi

# Set up database with URL-safe name
echo "Creating database: myapp_$SLUG"
mysql -e "CREATE DATABASE IF NOT EXISTS myapp_$SLUG"

echo "Setup complete!"
echo "Access your app at: https://$SLUG.myapp.test"
```

**`.worktree/post-delete`**

```bash
#!/usr/bin/env -S wtm hook
set -e

echo "Cleaning up worktree: {{ .Branch }}"

# Use gomplate's strings.Slug for URL-safe slug
SLUG="{{ .Branch | strings.Slug }}"

# Drop database
echo "Dropping database: myapp_$SLUG"
mysql -e "DROP DATABASE IF EXISTS myapp_$SLUG"

# Clean up any temporary files
rm -rf /tmp/myapp_$SLUG

echo "Cleanup complete!"
```

### Making Hooks Executable

After creating hook scripts, make them executable:

```bash
chmod +x .worktree/post-create
chmod +x .worktree/post-delete
```

### Making Hooks Executable

After creating hook scripts, make them executable:

```bash
chmod +x .worktree/post-create
chmod +x .worktree/post-delete
```

### Hook Execution

- Hooks must start with `#!/usr/bin/env -S wtm hook` shebang
- Hooks run in the worktree directory (not the root)
- If a hook fails (non-zero exit code), a warning is displayed but the operation continues
- Hooks are optional - if they don't exist or aren't executable, they're simply skipped
- The template is processed first, then the resulting script is executed

### More Hook Examples

**Python hook with dynamic setup:**

```python
#!/usr/bin/env -S wtm hook
import os
import subprocess

branch = "{{ .Branch }}"
slug = "{{ .Branch | strings.Slug }}"
directory = "{{ .Directory }}"

print(f"Setting up Python environment for {branch}")

# Create virtual environment
subprocess.run(["python3", "-m", "venv", ".venv"])

# Install dependencies
subprocess.run([".venv/bin/pip", "install", "-r", "requirements.txt"])

# Create config file
config = f"""
[database]
name = myapp_{slug}
host = localhost

[server]
port = 8000
"""

with open("config.ini", "w") as f:
    f.write(config)

print(f"Setup complete! Database: myapp_{slug}")
```

**Advanced bash hook with conditionals:**

```bash
#!/usr/bin/env -S wtm hook
set -e

BRANCH="{{ .Branch }}"
SLUG="{{ .Branch | strings.Slug }}"
IS_FEATURE="{{ .Branch | strings.HasPrefix "feature/" }}"

echo "Setting up $BRANCH"

# Install dependencies
npm install

# Feature branches get a test database
if [ "$IS_FEATURE" = "true" ]; then
    echo "Creating test database for feature branch"
    mysql -e "CREATE DATABASE IF NOT EXISTS test_${SLUG}"
else
    echo "Creating production database"
    mysql -e "CREATE DATABASE IF NOT EXISTS ${SLUG}"
fi

echo "Setup complete!"
```

### Combining Hooks with Template Files

Hooks and template files work great together:

1. **Template files** (`.worktree/files/*.tmpl`) set up static configuration files with Go templates
2. **Regular files** (`.worktree/files/*`) are copied as-is
3. **Hooks** (`.worktree/post-create`, `.worktree/post-delete`) perform dynamic setup tasks with Go templates

Example workflow:

```bash
# Template files create configuration (processed on copy)
.worktree/files/.env.tmpl → .env
.worktree/files/config.json.tmpl → config.json

# Regular files are copied as-is
.worktree/files/init.sql → init.sql

# Hooks install dependencies and set up services (processed on execution)
.worktree/post-create (with #!/usr/bin/env -S wtm hook)
```

Both template files and hooks now use the same Go template syntax and gomplate functions!

## Typical Workflow

Here's a typical development workflow using `wtm`:

```bash
# 1. Initialize your repository once
wtm init myorg/myrepo

# 2. Set up template files (one-time)
cd myrepo
mkdir -p .worktree/files
echo 'APP_URL={{ .Branch }}.myapp.test' > .worktree/files/.env.tmpl
echo 'DATABASE_NAME=myapp_{{ .Branch }}' >> .worktree/files/.env.tmpl
git add .worktree
git commit -m "Add worktree template files"
git push

# 3. Create worktrees for each feature/fix
wtm add main feature-user-auth
wtm add main feature-payment
wtm add main fix-login-bug

# 4. Work on each feature independently
cd feature-user-auth
# ... develop, commit, push ...

cd ../feature-payment
# ... develop, commit, push ...

# 5. List what you're working on
wtm ls

# 6. Clean up when done
wtm rm feature-user-auth
wtm rm feature-payment
```

## Tips & Best Practices

### 1. Use Descriptive Branch Names

Since the branch name becomes the directory name, use clear, descriptive names:

```bash
wtm add main feature-user-authentication
wtm add main fix-memory-leak-in-parser
wtm add main refactor-payment-module
```

### 2. Keep the Root Clean

Work inside the worktree directories, not in the root. The root directory should only contain:

- `.bare/` - The bare repository
- `.worktree/` - Hooks and template files
- `main/` (or your default branch)
- Other worktree directories

### 3. Use Template Files for Configuration

Use `.tmpl` files for configuration that needs branch-specific values:

- Environment files (`.env.tmpl`)
- Configuration files (`.json.tmpl`, `.yml.tmpl`)
- IDE settings (`.vscode/settings.json.tmpl`)
- Database connection strings

Use regular files (without `.tmpl`) for:

- SQL files with `{{` syntax
- Shell scripts
- Static configuration files

### 4. Regular Cleanup

Delete worktrees you're no longer using:

```bash
# List to see what you have
wtm ls

# Clean up old branches
wtm rm old-feature-1
wtm rm old-feature-2
```

### 5. Submodules Work Automatically

If your repository has submodules, they're automatically initialized in each worktree. No extra steps needed!

## Repository Structure

After running `wtm init`, your repository structure will look like this:

```
myrepo/
├── .bare/              # Bare repository (your .git folder)
├── .worktree/          # Hooks and template files (optional)
│   ├── files/          # Files to copy/process for each worktree
│   │   ├── .env.tmpl   # Template file (processed → .env)
│   │   ├── init.sql    # Regular file (copied as-is)
│   │   └── config/
│   │       └── local.yml.tmpl  # Template file (processed → local.yml)
│   ├── post-create     # Hook: runs after worktree creation
│   └── post-delete     # Hook: runs before worktree deletion
├── main/               # Default branch worktree
│   ├── .git            # Points to ../.bare
│   └── ... your code ...
├── feature-123/        # Feature branch worktree
│   ├── .git            # Points to ../.bare
│   ├── .env            # Generated from .env.tmpl
│   ├── init.sql        # Copied from files/
│   └── ... your code ...
└── fix-bug-456/        # Bug fix worktree
    ├── .git            # Points to ../.bare
    ├── .env            # Generated from .env.tmpl
    ├── init.sql        # Copied from files/
    └── ... your code ...
```

## Troubleshooting

### "Not in a worktree-managed repository"

Make sure you've initialized the repository with `wtm init` and you're in the repository root or one of its worktree directories.

### Worktree Already Exists

If you get an error that a worktree already exists, you can:

```bash
# List existing worktrees
wtm ls

# Delete the existing one
wtm rm existing-branch-name

# Or use a different branch name
wtm add main feature-123-v2
```

### Uncommitted Changes on Delete

If you try to delete a worktree with uncommitted changes:

```bash
# Commit or stash your changes first
cd feature-branch
git stash

# Or force delete
wtm rm feature-branch --force
```

## Requirements

- Git 2.5+ (for worktree support)
- [GitHub CLI](https://cli.github.com/) (`gh`) - optional, only needed for PR metadata via `wtm pr`

## Development

This project uses [mise](https://mise.jdx.dev/) for task automation and tool management.

### Setup

```bash
# Install mise if you haven't already
# See: https://mise.jdx.dev/getting-started.html

# Clone the repository
git clone https://github.com/vansdevcode/worktree-manager.git
cd worktree-manager

# Trust the mise configuration
mise trust

# mise will automatically install the required Go version
# based on .mise.toml configuration
```

### Available Tasks

```bash
# Build the binary
mise run build

# Run tests
mise run test

# Run tests with coverage
mise run test-coverage

# Clean build artifacts
mise run clean

# Install as gh extension (for testing)
mise run install

# Format Go code
mise run fmt

# Run linter
mise run lint
```

### Required Tools

Tools are automatically managed by mise (defined in `.mise.toml`):

- **Go 1.22+** - Auto-installed by mise
- **Git 2.5+** - Required for worktree support (must be installed separately)

### Running Tests

```bash
# Run all unit tests
mise run test

# Run with coverage report
mise run test-coverage

# Build and test
mise run clean && mise run build && mise run test
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

See [ROADMAP.md](ROADMAP.md) for planned features and ideas.

## License

MIT

## Credits

Created by [Vanderlei](https://github.com/vanderlei)

Inspired by the powerful but often underutilized Git worktrees feature.
