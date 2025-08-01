# Fish Shell Functions for Tasks CLI

This directory contains Fish shell functions for interactive task management with fzf integration.

## Installation

### Option 1: Source individual functions
```fish
# Add to your ~/.config/fish/config.fish
source /path/to/tasks/fish/tls.fish
source /path/to/tasks/fish/tt.fish
source /path/to/tasks/fish/te.fish
source /path/to/tasks/fish/tmd.fish
source /path/to/tasks/fish/tr.fish
source /path/to/tasks/fish/ta.fish
source /path/to/tasks/fish/tl.fish
```

### Option 2: Copy to Fish functions directory
```bash
cp fish/*.fish ~/.config/fish/functions/
```

### Option 3: Symlink to Fish functions directory
```bash
ln -sf /path/to/tasks/fish/*.fish ~/.config/fish/functions/
```

## Functions

### Core Functions

- **`tls [file]`** - List incomplete tasks only
- **`tt [file]`** - Toggle task completion status (done/undone)
- **`te [file]`** - Edit task interactively
- **`tmd [file]`** - Mark multiple tasks as done (multi-select)
- **`tr [file]`** - Remove task interactively (with confirmation)

### Utility Functions

- **`ta 'description' [file]`** - Add task quickly
- **`tl [file]`** - List all tasks

## Usage Examples

```fish
# Show only incomplete tasks
tls
tls project.md

# Toggle task completion
tt daily.md

# Edit a task
te work.md

# Mark multiple tasks as done (use Tab/Shift+Tab in fzf)
tmd

# Remove a task (with confirmation)
tr notes.md

# Quick add
ta "Review documentation"
ta "Call client" work.md

# List all tasks
tl project.md
```

## Requirements

- [fzf](https://github.com/junegunn/fzf) - Command-line fuzzy finder
- [tasks](../README.md) - The tasks CLI tool (must be in PATH or current directory)

## Notes

- All functions default to `TODO.md` if no file is specified
- Functions that modify tasks provide confirmation messages
- The `tr` function asks for confirmation before removing tasks
- Multi-select is supported in `tmd` using fzf's `-m` flag (Tab/Shift+Tab to select)