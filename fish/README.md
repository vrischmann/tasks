# Fish Shell Functions for Tasks CLI

This directory contains Fish shell functions for interactive task management with fzf integration.

## Installation

### Option 1: Source individual functions
```fish
# Add to your ~/.config/fish/config.fish
source /path/to/tasks/fish/tlist.fish
source /path/to/tasks/fish/ttoggle.fish
source /path/to/tasks/fish/tedit.fish
source /path/to/tasks/fish/tmark.fish
source /path/to/tasks/fish/tremove.fish
source /path/to/tasks/fish/tadd.fish
```

### Option 2: Copy to Fish functions directory
```bash
cp fish/*.fish ~/.config/fish/functions/
```

### Option 3: Symlink to Fish functions directory
```bash
ln -sf /path/to/tasks/fish/functions/*.fish ~/.config/fish/functions/
```

## Shell Completions

Fish shell completions are provided for the `tasks` CLI tool.

### Installation
```bash
# Copy completions file
cp fish/completions/tasks.fish ~/.config/fish/completions/

# Or symlink for development
ln -sf /path/to/tasks/fish/completions/tasks.fish ~/.config/fish/completions/
```

The completions provide:
- Command completion (ls, add, done, undo, rm, edit, search, completion)
- Flag completion (--file, --color, --section)
- Shell type completion for `tasks completion` command

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
