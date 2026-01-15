# gh-wt

A GitHub CLI extension for managing Git worktrees with ease.

## What are Git Worktrees?

Git worktrees allow you to have multiple working directories for a single repository. This is incredibly useful when you need to:

- Work on multiple branches simultaneously
- Quickly switch between features without stashing
- Run tests on one branch while developing on another
- Review PRs without disrupting your current work

`gh-wt` makes working with worktrees simple and intuitive, with added features like templates and safety checks.

## Installation

```bash
gh extension install vansdevcode/gh-wt
```

## Quick Start

```bash
# Initialize a repository for worktree management
gh wt init myorg/myrepo

# Create a new worktree for a feature
cd myrepo
gh wt create main feature-awesome-thing

# Work in your new worktree
cd feature-awesome-thing
# ... make changes ...

# List all worktrees
gh wt list

# Delete a worktree when done
gh wt delete feature-awesome-thing
```

## Commands

### `gh wt init`

Initialize a new worktree-managed repository.

```bash
gh wt init <repo> [dir] [--new]
```

**Arguments:**
- `repo` - Repository in format `org/repo` or full git URL
- `dir` - Directory name (optional, defaults to repo name)
- `--new` - Create a new repository instead of cloning

**Examples:**

```bash
# Clone an existing repository
gh wt init myorg/myrepo

# Clone with custom directory name
gh wt init myorg/myrepo my-project

# Clone using full git URL
gh wt init git@github.com:myorg/myrepo.git

# Create a new repository
gh wt init myorg/newproject --new
```

**What it does:**
1. Creates a directory structure with a bare repository in `.bare/`
2. Clones the repository (or creates a new one with `--new`)
3. Automatically creates a worktree for the default branch
4. Initializes submodules if present
5. Applies templates if `.worktree/templates/` exists

### `gh wt create`

Create a new worktree from a base branch.

```bash
gh wt create <base-branch> <new-branch-name>
```

**Arguments:**
- `base-branch` - Branch, tag, or commit to base from (e.g., `main`, `origin/develop`, `v1.0.0`)
- `new-branch-name` - Name for the new branch and directory

**Examples:**

```bash
# Create from main branch
gh wt create main feature-123

# Create from remote branch
gh wt create origin/develop fix-bug-456

# Create from a tag
gh wt create v1.0.0 hotfix-security

# Create from a specific commit
gh wt create a1b2c3d experiment
```

**What it does:**
1. Creates a new branch from the specified base
2. Creates a new directory with the branch name
3. Initializes submodules automatically
4. Processes and copies template files with variable replacement
5. Ready to start working immediately

### `gh wt delete`

Delete a worktree and its branch.

```bash
gh wt delete [branch-name] [--force]
```

**Arguments:**
- `branch-name` - Name of the branch/worktree to delete (optional, uses current directory if not specified)
- `--force`, `-f` - Force deletion even with uncommitted changes

**Examples:**

```bash
# Delete a specific worktree
gh wt delete feature-123

# Force delete with uncommitted changes
gh wt delete old-branch --force

# Delete current worktree (when inside a worktree directory)
gh wt delete
```

**Safety features:**
- Checks for uncommitted changes
- Warns about untracked files
- Prevents deleting worktree you're currently in
- Prompts for confirmation when there are untracked files

### `gh wt list`

List all worktrees in the current repository.

```bash
gh wt list
```

**Example output:**
```
Worktrees in /Users/you/projects/myrepo:

/Users/you/projects/myrepo/.bare           (bare)
/Users/you/projects/myrepo/main            a1b2c3d [main]
/Users/you/projects/myrepo/feature-123     d4e5f6g [feature-123]
```

## Template Support

One of the most powerful features of `gh-wt` is template support. Templates allow you to automatically set up configuration files for each worktree.

### Setting Up Templates

Create a `.worktree/templates/` directory in your repository root:

```bash
mkdir -p .worktree/templates
```

Any files you place in this directory will be copied to new worktrees with variable replacement.

### Available Variables

- `${WT_BRANCH}` - The branch name of the worktree
- `${WT_BRANCH_SLUG}` - URL-safe version of the branch name (lowercase, alphanumeric + hyphens only)
- `${WT_DIRECTORY}` - The absolute path to the worktree directory
- `${WT_ROOT_DIRECTORY}` - The absolute path to the repository root (where `.bare` is located)

**Slug Transformation Rules:**

The `WT_BRANCH_SLUG` variable transforms branch names to be URL-safe:
- Converts to lowercase
- Replaces `/` and `_` with `-`
- Replaces any non-alphanumeric characters (except `-`) with `-`
- Collapses multiple consecutive hyphens into one
- Trims leading and trailing hyphens

**Examples:**
- `feature/user-auth` → `feature-user-auth`
- `Feature/User-Auth` → `feature-user-auth`
- `fix_bug_123` → `fix-bug-123`
- `hot-fix/v2.0.1` → `hot-fix-v2-0-1`

### Example Templates

**`.worktree/templates/.env`**
```bash
APP_URL=${WT_BRANCH_SLUG}.myapp.test
APP_NAME=${WT_BRANCH}
APP_ENV=local
DATABASE_NAME=myapp_${WT_BRANCH_SLUG}
```

**`.worktree/templates/docker-compose.override.yml`**
```yaml
version: '3.8'
services:
  app:
    container_name: myapp_${WT_BRANCH_SLUG}
    ports:
      - "8000:8000"
    environment:
      - BRANCH=${WT_BRANCH}
      - BRANCH_SLUG=${WT_BRANCH_SLUG}
```

**`.worktree/templates/.vscode/settings.json`**
```json
{
  "terminal.integrated.cwd": "${WT_DIRECTORY}",
  "git.defaultBranchName": "${WT_BRANCH}"
}
```

### Directory Structure in Templates

You can create subdirectories in `.worktree/templates/` and they'll be preserved:

```
.worktree/templates/
├── .env
├── .vscode/
│   └── settings.json
└── config/
    └── local.yml
```

All of these will be copied to each new worktree with variables replaced.

## Hook Support

Hooks allow you to run custom scripts during worktree lifecycle events, similar to Git hooks. This is useful for automating setup and cleanup tasks.

### Setting Up Hooks

Create a `.worktree/` directory in your repository root:

```bash
mkdir .worktree
```

Add executable scripts with specific hook names in this directory.

### Available Hooks

- **`post-create`** - Runs after a worktree is created
- **`post-delete`** - Runs before a worktree is deleted

### Environment Variables

Hooks have access to these environment variables:

- `WT_ROOT_DIRECTORY` - Repository root directory (where `.bare` is located)
- `WT_DIRECTORY` - Worktree directory path
- `WT_BRANCH` - Branch name
- `WT_BRANCH_SLUG` - URL-safe branch name (lowercase, alphanumeric + hyphens)

### Example Hooks

**`.worktree/post-create`**
```bash
#!/usr/bin/env bash
set -e

echo "Setting up worktree for branch: $WT_BRANCH"
echo "URL-safe slug: $WT_BRANCH_SLUG"

# Install dependencies
npm install

# Copy environment file
if [ -f .env.example ]; then
    cp .env.example .env
    echo "Created .env from .env.example"
fi

# Set up database with URL-safe name
echo "Creating database: myapp_$WT_BRANCH_SLUG"
mysql -e "CREATE DATABASE IF NOT EXISTS myapp_$WT_BRANCH_SLUG"

echo "Setup complete!"
echo "Access your app at: https://$WT_BRANCH_SLUG.myapp.test"
```

**`.worktree/post-delete`**
```bash
#!/usr/bin/env bash
set -e

echo "Cleaning up worktree: $WT_BRANCH"

# Drop database
echo "Dropping database: myapp_$WT_BRANCH_SLUG"
mysql -e "DROP DATABASE IF EXISTS myapp_$WT_BRANCH_SLUG"

# Clean up any temporary files
rm -rf /tmp/myapp_$WT_BRANCH_SLUG

echo "Cleanup complete!"
```

### Making Hooks Executable

After creating hook scripts, make them executable:

```bash
chmod +x .worktree/post-create
chmod +x .worktree/post-delete
```

### Hook Execution

- Hooks run in the worktree directory (not the root)
- If a hook fails (non-zero exit code), a warning is displayed but the operation continues
- Hooks are optional - if they don't exist or aren't executable, they're simply skipped

### Hook Examples: Manual File Copying

Hooks can manually copy and process files, giving you full control over the setup process.

#### Approach 1: Simple Copy

Copy files directly without variable substitution:

**`.worktree/post-create`**
```bash
#!/usr/bin/env bash
set -e

echo "Copying configuration files for: $WT_BRANCH"

# Copy environment file
cp "$WT_ROOT_DIRECTORY/config/template.env" .env

# Copy other config files
cp "$WT_ROOT_DIRECTORY/config/database.yml" config/

echo "Files copied successfully!"
```

#### Approach 2: Copy with envsubst

Use `envsubst` to substitute environment variables in copied files:

**`.worktree/post-create`**
```bash
#!/usr/bin/env bash
set -e

echo "Processing templates for: $WT_BRANCH"

# Copy and substitute variables using envsubst
envsubst < "$WT_ROOT_DIRECTORY/config/template.env" > .env

# Verify variables were substituted
echo "Created .env with:"
echo "  - Branch: $WT_BRANCH"
echo "  - Directory: $WT_DIRECTORY"

echo "Template processing complete!"
```

#### Approach 3: Copy with Bash Variable Replacement

Use bash string manipulation for custom substitution logic:

**`.worktree/post-create`**
```bash
#!/usr/bin/env bash
set -e

echo "Generating configuration for: $WT_BRANCH"

# Read template
template=$(cat "$WT_ROOT_DIRECTORY/config/template.env")

# Replace variables manually
template="${template//\$\{WT_BRANCH\}/$WT_BRANCH}"
template="${template//\$\{WT_DIRECTORY\}/$WT_DIRECTORY}"
template="${template//\$\{WT_ROOT_DIRECTORY\}/$WT_ROOT_DIRECTORY}"

# Add dynamic content
random_port=$((8000 + RANDOM % 1000))
template="${template//\$\{PORT\}/$random_port}"

# Write output
echo "$template" > .env

echo "Configuration generated with random port: $random_port"
```

**When to use each approach:**
- **Simple copy**: Static files, no variables needed
- **envsubst**: Standard variable substitution, simple templates
- **Bash replacement**: Complex logic, conditional substitution, dynamic values

### Combining Hooks with Templates

Hooks and templates work great together:

1. **Templates** set up static configuration files
2. **Hooks** perform dynamic setup tasks

Example workflow:

```bash
# Templates create .env files
.worktree/templates/.env

# Hooks install dependencies and set up services
.worktree/post-create
```

## Typical Workflow

Here's a typical development workflow using `gh-wt`:

```bash
# 1. Initialize your repository once
gh wt init myorg/myrepo

# 2. Set up templates (one-time)
cd myrepo
mkdir -p .worktree/templates
echo 'APP_URL=${WT_BRANCH_SLUG}.myapp.test' > .worktree/templates/.env
git add .worktree
git commit -m "Add worktree templates"
git push

# 3. Create worktrees for each feature/fix
gh wt create main feature-user-auth
gh wt create main feature-payment
gh wt create main fix-login-bug

# 4. Work on each feature independently
cd feature-user-auth
# ... develop, commit, push ...

cd ../feature-payment
# ... develop, commit, push ...

# 5. List what you're working on
gh wt list

# 6. Clean up when done
gh wt delete feature-user-auth
gh wt delete feature-payment
```

## Tips & Best Practices

### 1. Use Descriptive Branch Names

Since the branch name becomes the directory name, use clear, descriptive names:

```bash
gh wt create main feature-user-authentication
gh wt create main fix-memory-leak-in-parser
gh wt create main refactor-payment-module
```

### 2. Keep the Root Clean

Work inside the worktree directories, not in the root. The root directory should only contain:
- `.bare/` - The bare repository
- `.worktree/` - Hooks and template files
- `main/` (or your default branch)
- Other worktree directories

### 3. Template Everything

Use templates for:
- Environment files (`.env`)
- Docker configurations
- IDE settings
- Database connection strings
- Local configuration files

### 4. Regular Cleanup

Delete worktrees you're no longer using:

```bash
# List to see what you have
gh wt list

# Clean up old branches
gh wt delete old-feature-1
gh wt delete old-feature-2
```

### 5. Submodules Work Automatically

If your repository has submodules, they're automatically initialized in each worktree. No extra steps needed!

## Repository Structure

After running `gh wt init`, your repository structure will look like this:

```
myrepo/
├── .bare/              # Bare repository (your .git folder)
├── .worktree/          # Hooks and templates (optional)
│   ├── templates/      # Template files
│   │   ├── .env
│   │   └── config/
│   │       └── local.yml
│   ├── post-create     # Hook: runs after worktree creation
│   └── post-delete     # Hook: runs before worktree deletion
├── main/               # Default branch worktree
│   ├── .git            # Points to ../.bare
│   └── ... your code ...
├── feature-123/        # Feature branch worktree
│   ├── .git            # Points to ../.bare
│   ├── .env            # Generated from template
│   └── ... your code ...
└── fix-bug-456/        # Bug fix worktree
    ├── .git            # Points to ../.bare
    ├── .env            # Generated from template
    └── ... your code ...
```

## Troubleshooting

### "Not in a worktree-managed repository"

Make sure you've initialized the repository with `gh wt init` and you're in the repository root or one of its worktree directories.

### Worktree Already Exists

If you get an error that a worktree already exists, you can:

```bash
# List existing worktrees
gh wt list

# Delete the existing one
gh wt delete existing-branch-name

# Or use a different branch name
gh wt create main feature-123-v2
```

### Uncommitted Changes on Delete

If you try to delete a worktree with uncommitted changes:

```bash
# Commit or stash your changes first
cd feature-branch
git stash

# Or force delete
gh wt delete feature-branch --force
```

## Requirements

- [GitHub CLI](https://cli.github.com/) (`gh`) installed
- Git 2.5+ (for worktree support)
- macOS (Linux support coming soon)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

See [ROADMAP.md](ROADMAP.md) for planned features and ideas.

## License

MIT

## Credits

Created by [Vanderlei](https://github.com/vanderlei)

Inspired by the powerful but often underutilized Git worktrees feature.
