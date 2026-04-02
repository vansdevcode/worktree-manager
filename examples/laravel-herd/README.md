# Laravel Herd Example

This example demonstrates how to use Worktree Manager with Laravel Herd.

- `.wtm.toml` is committed to the repository with hooks, files, and default variables.
- Template and static files referenced in `[files]` live in the `.wtm/` directory.
- `post-create` hook installs dependencies, links to Herd, and clones the database.
- `pre-delete` hook drops the branch database and unlinks from Herd.
