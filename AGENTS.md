# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a stateless, composable CLI tool for managing markdown task lists built with Go. It provides Unix-friendly commands for manipulating tasks and sections stored in markdown files, designed for scripting and integration with other tools like `fzf` and shell workflows.

## Development Commands

### Build and Run
```bash
go build
./tasks --file demo.md ls
```

### Run CLI commands directly
```bash
go run main.go --file demo.md ls
go run main.go --file demo.md add "New task"
go run main.go --file demo.md done 5
```

### Development workflow
```bash
go fmt ./...                      # Format code
go mod tidy                       # Clean up dependencies
go test                          # Run all tests
go build && ./tasks --file demo.md ls  # Quick test
```

## Application Architecture

### Core Components (Single-file architecture, ~1,123 lines)

- **Module**: `dev.rischmann.fr/tasks`
- **Go Version**: 1.24.5
- **Dependencies**: Minimal external dependencies (golang.org/x/term for terminal support, testify for testing)
- **Fish Shell Integration**: Comprehensive Fish shell functions with fzf integration for interactive task management

### Key Components
- `ItemType` enum - Distinguishes between sections and tasks
- `Item` struct - Core data structure for tasks and sections
- `TaskManager` struct - Centralized data handling with Load/Save operations
- `parseMarkdownFile()` - Regex-based markdown parser
- `saveToFile()` - Write items back to markdown with proper formatting
- `deleteItem()` - Delete tasks or sections with hierarchical children removal
- `getVersion()` - Build info and version reporting

### Command Handlers
- `handleList()` - List all items with 1-based indexing
- `handleAdd()` - Add tasks or sections with positioning
- `handleDone()/handleUndo()` - Toggle task completion status
- `handleRemove()` - Remove items with section hierarchy support
- `handleEdit()` - Open items in $EDITOR with line positioning

## CLI Interface

### Usage Patterns
```bash
tasks [--file <path>] <command> [args]
```

### Available Commands
- `ls` - List all tasks and sections with line numbers
- `add <text>` - Add a new task
- `add --section <level> <text>` - Add a new section at level 1-6
- `done <id>` - Mark task as completed
- `undo <id>` - Mark task as incomplete
- `rm <id>` - Remove task or section (with children)
- `edit <id>` - Edit task or section in $EDITOR
- `search <term> [...]` - Search tasks and sections with fuzzy matching
- `--version` - Show version information

### Design Principles
1. **Stateless**: Each command loads, operates, and saves independently
2. **Composable**: Output is parsable and suitable for piping
3. **Unix-friendly**: Proper exit codes, error messages to stderr
4. **Editor Integration**: Respects $EDITOR environment variable

## File Structure

### Markdown format supported:
```markdown
# Main Section
- [ ] Pending task
- [x] Completed task

## Sub Section
- [ ] Another task

### Nested Section
- [ ] Deeply nested task
```

### Test files and additional components:
- `demo.md` - Complex hierarchical example with multiple levels
- `fish/` - Fish shell functions for interactive task management with fzf
- `Justfile` - Task automation using the just command runner
- `AGENTS.md` - Agent configuration documentation
- Test files are created dynamically in unit tests

## Development Notes

### Parser Implementation
- Uses regex-based parsing (not goldmark AST) for simplicity
- Supports headings: `^(#{1,6})\s+(.+)$`
- Supports tasks: `^(\s*)-\s+\[([x\s])\]\s+(.+)$`
- Maintains original file structure when saving

### TaskManager Pattern
- Centralized data operations with Load() and Save() methods
- Stateless - each operation creates new instance
- Error handling with proper Go error patterns
- Index validation and bounds checking

### Editor Integration
- Supports vim, nano, emacs, VS Code line positioning
- Falls back to basic file opening for unknown editors
- Inherits stdin/stdout/stderr for proper terminal interaction

### Known Limitations
- Single-line tasks only (no multiline content support)
- Regex-based parsing may miss edge cases
- Basic line number approximation for editor positioning
- No undo/redo functionality (stateless by design)
- No configuration file support

## Common Development Tasks

### Adding new commands
1. Add case in main() switch statement
2. Create handleXxx(filePath string, args []string) function
3. Use TaskManager pattern: Load() → Modify → Save()
4. Add proper error handling and user feedback
5. Add comprehensive tests

### Extending TaskManager functionality
1. Add method to TaskManager struct
2. Follow existing patterns for error handling
3. Ensure operations work with 0-based internal indexing
4. Add unit tests covering edge cases

### Modifying parser
1. Update regex patterns in `parseMarkdownFile()`
2. Modify `Item` struct if needed
3. Update `saveToFile()` to maintain format consistency
4. Test with various markdown edge cases

### Testing Strategy
- Unit tests for core functions (parsing, saving, deletion)
- TaskManager method tests with temporary files
- Integration tests for complete workflows
- Error handling tests for invalid operations
- All tests use createTestFile() helper for temporary files

## Fish Shell Integration

### Interactive Functions
The repository includes Fish shell functions for enhanced interactive task management:

- **`tlist [file]`** - List incomplete tasks only
- **`ttoggle [file]`** - Toggle task completion status with fzf selection
- **`tedit [file]`** - Edit task interactively with fzf selection
- **`tmark [file]`** - Mark multiple tasks as done (multi-select with fzf)
- **`tremove [file]`** - Remove task interactively with confirmation
- **`tadd [file]`** - Add task interactively after selecting position

### Installation
```fish
# Copy to Fish functions directory
cp fish/functions/*.fish ~/.config/fish/functions/

# Or symlink for development
ln -sf /path/to/tasks/fish/functions/*.fish ~/.config/fish/functions/
```

### Usage Examples
```fish
# Show only incomplete tasks
tlist project.md

# Toggle task completion interactively
ttoggle daily.md

# Mark multiple tasks as done
tmark work.md

# Add task with position selection
tadd TODO.md
```

## CLI Command Examples

### Basic Usage
```bash
# List all items
./tasks --file todo.md ls

# Add a task
./tasks --file todo.md add "Review documentation"

# Add a section
./tasks --file todo.md add --section 2 "New Project Phase"

# Mark task 5 as done
./tasks --file todo.md done 5

# Edit task 3 in $EDITOR
./tasks --file todo.md edit 3

# Remove task 7
./tasks --file todo.md rm 7
```

### Scripting Integration
```bash
# Find incomplete tasks
./tasks --file todo.md ls | grep "\[ \]"

# Count total tasks
./tasks --file todo.md ls | grep -c "^\d\+\s\+.*- \["

# Use with fzf for interactive selection
./tasks --file todo.md ls | fzf

# Pipe to other tools
./tasks --file todo.md ls | awk '/- \[ \]/ { print $1 }' | head -5
```