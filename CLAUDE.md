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

### Core Components (867 lines, single-file architecture)

- **Module**: `dev.rischmann.fr/tasks`
- **Go Version**: 1.24.5
- **Dependencies**: Bubble Tea (TUI), lipgloss (styling)

### Key Functions
- `parseMarkdownFile()` - Regex-based markdown parser for sections/tasks
- `initialModel()` - Initialize Bubble Tea model with file data
- `updateVisibleItems()` - Manage section collapse/expand state
- `Update()` - Handle all keyboard input and state changes
- `View()` - Render UI with lipgloss styling
- `saveToFile()` - Write changes back to markdown
- `renderBanner()` - Generate stylized ASCII art banner
- `renderVisibleItems()` - Render the task/section tree
- `renderInput()` - Render input field when editing
- `renderFooter()` - Render status footer with file info

### Data Structures
```go
type Item struct {
    Type      ItemType  // TypeSection or TypeTask
    Level     int       // Heading level (1-6) or indentation
    Content   string    // Text content
    Checked   *bool     // nil for sections, true/false for tasks
    Children  []Item    // Children items (unused in current implementation)
    Collapsed bool      // For sections only
}

type Model struct {
    items           []Item    // All parsed items
    cursor          int       // Current selection in visibleItems
    filename        string    // Path to markdown file
    visibleItems    []int     // Indices of currently visible items
    inputMode       bool      // Whether in text input mode
    inputText       string    // Current input text
    editingIndex    int       // Index of item being edited (-1 for new)
    newSectionLevel int       // Level of section being created (0 = task)
    hMode          bool       // Whether waiting for number after 'h'
    dirty          bool       // Whether file has unsaved changes
    fileModTime    time.Time  // File modification time
    width          int        // Terminal width
    height         int        // Terminal height
}
```

## Features Implemented

### Navigation & Controls
- `j`/`k` or `↑`/`↓` - Navigate between sections and tasks
- `space` - Toggle task completion (☒/☐)
- `enter` - Toggle section expand/collapse
- `←` - Collapse current section
- `→` - Expand current section
- `n` - Create new task (enters input mode)
- `h1`-`h6` - Create new section at specified level (two-key sequence)
- `e` - Edit current task or section content (enters input mode)
- `Alt+j`/`Alt+k` or `Alt+↓`/`Alt+↑` - Move items up/down
- `s` - Save changes to file
- `q` or `Ctrl+c` - Quit application
- `--version` - Show version information (command line flag)

### Input Mode
- Unified input system for tasks and sections
- Context-aware prompts ("New task:", "Edit section:", "New h2 section:")
- Text input with live cursor display (│)
- `Enter` - Save input and exit input mode
- `Esc` - Cancel input and exit input mode
- `Backspace` - Delete characters
- Styled input field with background highlighting

### Visual Design (lipgloss styling)
- **Color Scheme**: Purple primary (#7C3AED), pink accents (#EC4899), green success (#10B981), gray muted (#6B7280)
- **Sections**: Pink text when expanded, gray when collapsed, with ▼/▶ arrows
- **Tasks**: Green ☒ for completed (with strikethrough), gray ☐ for pending
- **Smart Highlighting**: Fixed-width background highlighting that adapts to indentation and terminal width
- **Tree Structure**: Proper indentation for nested sections (2 spaces per level)
- **Clean Selection**: Background-only highlighting without distracting arrows
- **ASCII Banner**: Stylized "TASKS" banner with green gradient effect
- **Status Footer**: Shows file modification status, filename, and last save time
- **Input Field**: Styled input with cursor indicator (│) and contextual prompts

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

### Styling Architecture
- All styles defined as global variables using lipgloss
- Consistent color scheme throughout application
- Smart highlighting system with dynamic width calculation based on terminal size
- Fixed-width selection highlighting that prevents text jumping
- Professional terminal UI with modern Unicode symbols (☐/☒)
- Background-only highlighting prevents text jumping
- Gradient banner with ASCII art using multiple green shades
- Responsive footer with file status and modification time
- Context-aware input prompts with styled backgrounds

### Recent Improvements
- **Smart Highlighting**: Fixed-width background highlighting that prevents text jumping and adapts to terminal width
- **Section Creation**: Two-key sequences (h1-h6) for precise section level creation
- **Enhanced Movement**: Support for both vim-style (Alt+j/k) and arrow (Alt+↑/↓) movement
- **Visual Polish**: Modern checkbox symbols (☐/☒) and clean background-only selection
- **Unified Input**: Single input system handles both task and section creation/editing
- **Responsive Design**: Terminal-aware layout that adapts to window resizing
- **Status Tracking**: File modification indicator and timestamp display
- **Version Support**: Command-line version flag with build info integration
- **Enhanced Banner**: ASCII art banner with gradient coloring effect

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