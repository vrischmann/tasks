# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a terminal-based markdown task manager built with Go, Bubble Tea, and lipgloss. It provides an interactive interface for viewing and managing tasks stored in markdown files with proper hierarchical structure.

## Development Commands

### Build and Run
```bash
go build
./tasks <markdown-file>
```

### Run directly
```bash
go run main.go demo.md
```

### Development workflow
```bash
go fmt ./...           # Format code
go mod tidy           # Clean up dependencies
go build && ./tasks demo.md  # Quick test
```

## Application Architecture

### Core Components (940 lines, single-file architecture)

- **Module**: `dev.rischmann.fr/tasks`
- **Go Version**: 1.24.5
- **Dependencies**: Bubble Tea (TUI), lipgloss (styling), unicode/utf8 (Unicode support)

### Key Functions
- `parseMarkdownFile()` - Regex-based markdown parser for sections/tasks
- `initialModel()` - Initialize Bubble Tea model with file data
- `updateVisibleItems()` - Manage section collapse/expand state
- `adjustCursor()` - Ensure cursor stays within valid bounds
- `deleteItem()` - Delete tasks or sections with all contents
- `collapseAll()` / `expandAll()` - Global section state management
- `Update()` - Handle all keyboard input and state changes
- `View()` - Render UI with lipgloss styling
- `saveToFile()` - Write changes back to markdown with improved formatting
- `renderBanner()` - Generate stylized ASCII art banner
- `renderVisibleItems()` - Render the task/section tree
- `renderInput()` - Render input field when editing (supports Unicode)
- `renderFooter()` - Render status footer with file info


## Features Implemented

### Navigation & Controls
- `j`/`k` or `↑`/`↓` - Navigate between sections and tasks
- `space` - Toggle task completion (☒/☐)
- `enter` - Toggle section expand/collapse
- `←` - Collapse current section
- `→` - Expand current section
- `-` - Collapse all sections (global overview mode)
- `+` - Expand all sections (global detailed mode)
- `n` - Create new task (enters input mode)
- `h1`-`h6` - Create new section at specified level (two-key sequence)
- `e` - Edit current task or section content (enters input mode)
- `d` - Delete current item (tasks or sections with all contents)
- `Alt+j`/`Alt+k` or `Alt+↓`/`Alt+↑` - Move items up/down
- `s` - Save changes to file
- `q` or `Ctrl+c` - Quit application
- `--version` - Show version information (command line flag)

### Input Mode
- Unified input system for tasks and sections
- Context-aware prompts ("New task:", "Edit section:", "New h2 section:")
- Text input with live cursor display (│)
- Full Unicode support for international characters and emojis
- Empty file support - can create first items in blank markdown files
- `Enter` - Save input and exit input mode
- `Esc` / `Ctrl+C` - Cancel input and exit input mode
- `Backspace` - Delete characters
- Styled input field with background highlighting


## File Structure

### Example markdown format supported:
```markdown
# Main Section
- [ ] Pending task
- [x] Completed task

## Sub Section
- [ ] Another task

### Nested Section
- [ ] Deeply nested task
```

### Test files included:
- `demo.md` - Complex hierarchical example with multiple levels
- `test.md` - Simple test file for basic functionality

## Development Notes

### Parser Implementation
- Uses regex-based parsing (not goldmark AST) for simplicity
- Supports headings: `^(#{1,6})\s+(.+)$`
- Supports tasks: `^(\s*)-\s+\[([x\s])\]\s+(.+)$`
- Maintains original file structure when saving

### UI State Management
- `visibleItems` tracks which items are shown (handles section collapse)
- Navigation cursor works on visible items only
- Input mode completely overrides normal navigation
- Two-key sequence handling with `hMode` state for section creation
- Section stack tracking for proper indentation display
- Dynamic highlight width calculation based on terminal width and indentation level
- Window size handling with responsive layout
- File modification tracking with dirty state indicator
- Terminal dimension awareness for proper rendering

### Known Limitations
- Single-line tasks only (no multiline content support)
- Regex-based parsing may miss edge cases
- No undo/redo functionality
- No search/filter capabilities
- No configuration file support
- No task due dates or priorities

## Common Development Tasks

### Adding new key bindings
1. Add case in `handleNavigation()`, `handleInputMode()`, or `handleHMode()` functions
2. Handle special modes (hMode, inputMode) if needed
3. Consider window resize effects with `tea.WindowSizeMsg`
4. Test with both navigation and input modes
5. Consider two-key sequences for complex operations

### Modifying visual styling
1. Update color variables at top of file (lines 28-116)
2. Modify relevant lipgloss styles
3. Test with different terminal color schemes and sizes
4. Consider responsive behavior for different terminal widths

### Extending parser
1. Update regex patterns in `parseMarkdownFile()` (lines 140-141)
2. Modify `Item` struct if needed (lines 120-127)
3. Update `saveToFile()` to maintain format (lines 549-581)
4. Test with various markdown edge cases