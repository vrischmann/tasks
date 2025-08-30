# Fish Shell Functions for Tasks CLI

This directory contains Fish shell functions for interactive task management with fzf integration.

## Installation

**Note:** If you installed `tasks` via Homebrew, the Fish functions and completions are already installed automatically.

For manual installation, copy the Fish functions and completions to your Fish configuration directory:

```bash
# Copy functions
cp fish/functions/*.fish ~/.config/fish/functions/

# Copy completions
cp fish/completions/tasks.fish ~/.config/fish/completions/
```

The completions provide command, flag, and argument completion for the `tasks` CLI tool.

## Functions

### Core Functions

- **`tlist [file]`** - List incomplete tasks only
- **`ttoggle [file]`** - Toggle task completion status (done/undone)
- **`tedit [file]`** - Edit task interactively
- **`tmark [file]`** - Mark multiple tasks as done (multi-select)
- **`tremove [file]`** - Remove task interactively (with confirmation)
- **`tadd [file]`** - Add task interactively after selecting position with fzf

## Usage Examples

```fish
# Show only incomplete tasks
tlist
tlist project.md

# Toggle task completion
ttoggle daily.md

# Edit a task
tedit work.md

# Mark multiple tasks as done (use Space to select in fzf)
tmark

# Remove a task (with confirmation)
tremove notes.md

# Add a task interactively (select position with fzf)
tadd
tadd project.md
```

## Requirements

- [fzf](https://github.com/junegunn/fzf) - Command-line fuzzy finder
- [tasks](../README.md) - The tasks CLI tool (must be in PATH or current directory)

## Notes

- All functions default to `TODO.md` if no file is specified
- Functions that modify tasks provide confirmation messages
- The `tremove` function asks for confirmation before removing tasks
- Multi-select is supported in `tmark` using fzf's `-m` flag (Space to select)
