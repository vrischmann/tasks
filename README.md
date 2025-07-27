# Tasks - Terminal Markdown Task Manager

A beautiful, interactive terminal application for managing tasks stored in markdown files. Built with Go, Bubble Tea, and lipgloss for a modern TUI experience.

![Tasks Demo](https://img.shields.io/badge/Go-1.24.5-blue) ![License](https://img.shields.io/badge/license-MIT-green)

## Features

✨ **Interactive Terminal UI** - Navigate with Vim-style keys  
📋 **Markdown Integration** - Works with standard markdown task lists  
🌳 **Hierarchical Structure** - Support for nested sections and tasks  
🎨 **Beautiful Styling** - Modern colors and typography  
⚡ **Live Editing** - Create and edit tasks in-place  
💾 **Auto-save** - Save changes back to your markdown files  
🔄 **Task Management** - Toggle completion, move tasks, collapse sections  

## Installation

### Prerequisites
- Go 1.24.5 or later

### Build from Source
```bash
git clone <repository-url>
cd tasks
go build
```

## Quick Start

1. **Run with a markdown file:**
   ```bash
   ./tasks demo.md
   ```

2. **Try the included examples:**
   ```bash
   ./tasks demo.md    # Complex hierarchical example
   ./tasks test.md    # Simple test file
   ```

3. **Use with your own files:**
   ```bash
   ./tasks path/to/your/todo.md
   ```

## Supported Markdown Format

The application works with standard markdown task lists:

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

## Controls

### Navigation
| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `q` | Quit application |

### Task Management
| Key | Action |
|-----|--------|
| `space` | Toggle task completion (✓/○) |
| `n` | Create new task |
| `e` | Edit current task |
| `Alt+j` | Move item down |
| `Alt+k` | Move item up |

### Section Control
| Key | Action |
|-----|--------|
| `enter` | Toggle section expand/collapse |
| `←` | Collapse current section |
| `→` | Expand current section |

### File Operations
| Key | Action |
|-----|--------|
| `s` | Save changes to file |

### Input Mode
When creating or editing tasks:

| Key | Action |
|-----|--------|
| `Enter` | Save and exit input mode |
| `Esc` | Cancel and exit input mode |
| `Backspace` | Delete characters |

## Visual Elements

- **✓** Completed tasks (green, with strikethrough)
- **○** Pending tasks (gray)
- **▼** Expanded sections (pink)
- **▶** Collapsed sections (gray)
- **►** Current selection indicator
- **│** Text input cursor

## Examples

### Creating a New Task
1. Navigate to where you want to add a task
2. Press `n`
3. Type your task description
4. Press `Enter` to save

### Organizing with Sections
- Use markdown headers (`#`, `##`, `###`) to create sections
- Navigate to a section header and press `Enter` to collapse/expand
- Use `←`/`→` for quick collapse/expand

### Editing Existing Tasks
1. Navigate to the task you want to edit
2. Press `e`
3. Modify the text
4. Press `Enter` to save changes

## File Structure

```
tasks/
├── main.go           # Main application code
├── go.mod            # Go module definition
├── demo.md           # Complex example file
├── test.md           # Simple example file
├── README.md         # This file
└── CLAUDE.md         # Developer documentation
```

## Development

### Requirements
- Go 1.24.5+
- Terminal with color support

### Dependencies
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [lipgloss](https://github.com/charmbracelet/lipgloss) - Styling and layout

### Building
```bash
go mod download
go build
```

### Testing
```bash
go run main.go demo.md
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## Known Limitations

- Single-line tasks only (no multiline content)
- No undo/redo functionality
- No search or filtering
- No configuration file support

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Built with [Charm](https://charm.sh/) tools (Bubble Tea, lipgloss)
- Inspired by terminal-based productivity tools
- Thanks to the Go community for excellent tooling