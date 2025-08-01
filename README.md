# Tasks - Composable CLI Markdown Task Manager

A stateless, Unix-friendly command-line tool for managing tasks stored in markdown files. Built with Go for composability, scripting, and integration with shell workflows.

## Features

- 🚀 **Stateless CLI** - Each command operates independently, perfect for scripting
- 📋 **Markdown Integration** - Works with standard markdown task lists
- 🌳 **Hierarchical Structure** - Support for nested sections and tasks
- 🔧 **Composable Output** - Clean, parsable output suitable for piping
- ⚡ **Task Management** - Add, complete, edit, and remove tasks and sections
- 📝 **Editor Integration** - Edit items directly in your preferred editor ($EDITOR)
- 🗑️ **Smart Deletion** - Remove sections with all their children
- 🌍 **Unicode Support** - Full support for international characters and emojis
- 📁 **Empty File Support** - Works with empty markdown files

## Installation

### Prerequisites
- Go 1.24.5 or later

### Install with Go
```bash
go install dev.rischmann.fr/tasks@latest
```

### Build from Source
```bash
git clone https://github.com/vrischmann/tasks
cd tasks
go build
```

## Quick Start

### Basic Commands
```bash
# List all tasks and sections
tasks --file todo.md ls

# Add a new task
tasks --file todo.md add "Review documentation"

# Add a new section
tasks --file todo.md add --section 2 "Development Phase"

# Mark task 5 as completed
tasks --file todo.md done 5

# Mark task 3 as incomplete
tasks --file todo.md undo 3

# Edit task 7 in your editor
tasks --file todo.md edit 7

# Remove task 4
tasks --file todo.md rm 4
```

### Try with Demo File
```bash
# Build and test with included demo
go build
./tasks --file demo.md ls
./tasks --file demo.md add "Try this new CLI tool"
```

## CLI Reference

### Usage
```bash
tasks [--file <path>] <command> [args]
```

### Global Options
- `--file <path>` - Specify markdown file (default: TODO.md)
- `--version`, `-v` - Show version information

### Commands

#### `ls` - List Items
Lists all tasks and sections with 1-based line numbers.
```bash
tasks --file todo.md ls
```

Example output:
```
1   # Project Tasks
2     ## Frontend
3   - [x] Setup React project
4   - [ ] Create components
5     ## Backend
6   - [ ] API design
```

#### `add` - Add Items
Add tasks or sections to the file.

**Add a task:**
```bash
tasks --file todo.md add "New task description"
```

**Add a section:**
```bash
tasks --file todo.md add --section 1 "Main Section"
tasks --file todo.md add --section 2 "Subsection"
```

#### `done` / `undo` - Toggle Completion
Mark tasks as completed or incomplete.
```bash
tasks --file todo.md done 3    # Mark task 3 as completed
tasks --file todo.md undo 3    # Mark task 3 as incomplete
```

#### `rm` - Remove Items
Remove tasks or sections. When removing sections, all child items are also removed.
```bash
tasks --file todo.md rm 5      # Remove item 5
```

#### `edit` - Edit in Editor
Open the specified item in your preferred editor ($EDITOR).
```bash
tasks --file todo.md edit 2    # Edit item 2 in $EDITOR
```

Supports line positioning for:
- vim/vi (`+line`)
- nano (`+line`)
- emacs (`+line`)
- VS Code (`--goto file:line`)

## Supported Markdown Format

The tool works with standard markdown task lists:

```markdown
# Project Tasks

## Frontend Development
- [ ] Setup React project
- [x] Create main components
- [ ] Implement routing

### UI Components
- [ ] Button component
- [ ] Form component

## Backend Development
- [x] API design
- [ ] Database setup
```

## Scripting and Integration

### Shell Integration
```bash
# Count incomplete tasks
tasks --file todo.md ls | grep -c "\[ \]"

# List only incomplete tasks
tasks --file todo.md ls | grep "\[ \]"

# Get task IDs for incomplete tasks
tasks --file todo.md ls | awk '/\[ \]/ {print $1}'
```

### fzf Integration
```bash
# Interactive task selection
TASK_ID=$(tasks --file todo.md ls | fzf | awk '{print $1}')
tasks --file todo.md done $TASK_ID
```

### Fish Shell Functions
```fish
# Add to your ~/.config/fish/config.fish
function td
    tasks --file ~/todo.md $argv
end

function tl
    tasks --file ~/todo.md ls
end

function ta
    tasks --file ~/todo.md add $argv
end
```

### Vim Integration
```vim
" Add to your .vimrc
command! -nargs=* TaskAdd execute '!tasks --file % add' shellescape(<q-args>)
command! TaskList !tasks --file % ls
```

## Examples

### Daily Workflow
```bash
# Morning: Check what needs to be done
tasks --file daily.md ls

# Add new tasks as they come up
tasks --file daily.md add "Review pull request #123"
tasks --file daily.md add "Call client about requirements"

# Mark tasks as complete throughout the day
tasks --file daily.md done 5
tasks --file daily.md done 7

# Evening: Review what's left
tasks --file daily.md ls | grep "\[ \]"
```

### Project Management
```bash
# Set up project structure
tasks --file project.md add --section 1 "Planning Phase"
tasks --file project.md add --section 2 "Development"
tasks --file project.md add --section 2 "Testing"
tasks --file project.md add --section 1 "Deployment"

# Add tasks under sections
tasks --file project.md add "Define requirements"
tasks --file project.md add "Create wireframes"
```

### Git Integration
```bash
# Create tasks from git issues
gh issue list --json number,title | \
  jq -r '.[] | "Issue #\(.number): \(.title)"' | \
  while read line; do
    tasks --file issues.md add "$line"
  done
```

## File Structure

```
tasks/
├── main.go           # Main application (~600 lines, single file)
├── main_test.go      # Comprehensive test suite
├── go.mod            # Go module (standard library only)
├── go.sum            # Go module checksums
├── demo.md           # Example markdown file
├── README.md         # This file
├── CLAUDE.md         # Developer documentation
└── PLAN.md           # Implementation plan
```

## Development

### Requirements
- Go 1.24.5+
- No external dependencies (uses standard library only)

### Building
```bash
go mod download
go build
```

### Testing
```bash
go test -v          # Run all tests
go test -cover      # Run with coverage
```

### Development Commands
```bash
go fmt ./...        # Format code
go mod tidy         # Clean dependencies
go vet              # Static analysis
```

## Design Philosophy

This tool follows Unix philosophy:
- **Do one thing well**: Manage markdown task lists
- **Composable**: Output works with pipes and other tools
- **Stateless**: Each command is independent
- **Text-based**: Works with standard markdown format
- **Scriptable**: Suitable for automation and workflows

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass: `go test`
6. Submit a pull request

## License

MIT License - see LICENSE file for details
