package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime/debug"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ItemType represents the type of item in the task list
type ItemType int

const (
	TypeSection ItemType = iota // Section header
	TypeTask                    // Task item
)

// Define color scheme and styles
var (
	// Colors
	primaryColor = lipgloss.Color("#7C3AED") // Purple
	accentColor  = lipgloss.Color("#EC4899") // Pink
	successColor = lipgloss.Color("#10B981") // Green
	mutedColor   = lipgloss.Color("#6B7280") // Gray
	textColor    = lipgloss.Color("#F9FAFB") // Light gray

	// Status line styles
	dirtyIndicatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F59E0B")).
				Bold(true)

	lastUpdateStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	// Footer right side styles with distinct gradient backgrounds
	footerFilenameStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#065F46")).
				Foreground(textColor).
				Padding(0, 1)

	footerTimeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#059669")).
			Foreground(textColor).
			Padding(0, 1)

	sectionCollapsedStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Bold(true)

	// Task styles
	taskCompletedStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Strikethrough(true)

	taskPendingStyle = lipgloss.NewStyle().
				Foreground(textColor)

	sectionStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	// Selection highlight (base style, width will be set dynamically)
	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#374151")).
			Bold(true)

	// Checkbox styles
	checkedBoxStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	uncheckedBoxStyle = lipgloss.NewStyle().
				Foreground(mutedColor)
	// Arrow styles
	arrowExpandedStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true)

	arrowCollapsedStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	// Help text style
	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Margin(1, 0, 0, 0)

	// Input field style
	inputStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#374151")).
			Foreground(textColor).
			Padding(0, 1)

	inputPromptStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	// Banner styles - darker gradient from green to white
	bannerGreen1 = lipgloss.NewStyle().Foreground(lipgloss.Color("#059669")) // Medium green
	bannerGreen2 = lipgloss.NewStyle().Foreground(lipgloss.Color("#10b981")) // Green
	bannerGreen3 = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e")) // Bright green
	bannerGreen4 = lipgloss.NewStyle().Foreground(lipgloss.Color("#34d399")) // Light green
	bannerGreen5 = lipgloss.NewStyle().Foreground(lipgloss.Color("#6ee7b7")) // Very light green
	bannerGreen6 = lipgloss.NewStyle().Foreground(lipgloss.Color("#a7f3d0")) // Pale green
	bannerWhite  = lipgloss.NewStyle().Foreground(lipgloss.Color("#f0f9ff")) // Off-white
)

// Item represents a single task or section in the task list
type Item struct {
	Type      ItemType // Type of item (section or task)
	Level     int      // Level of the section (0 for tasks, 1-6 for sections)
	Content   string   // Content of the item
	Checked   *bool    // Pointer to bool for task completion status (nil for sections)
	Children  []Item   // Children items (for sections, tasks are not nested)
	Collapsed bool     // Whether the section is collapsed
}

// parseMarkdownFile reads a Markdown file and returns a slice of Items
// Returns an error if the file cannot be read or parsed
func parseMarkdownFile(filename string) ([]Item, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	headerRegex := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	taskRegex := regexp.MustCompile(`^(\s*)-\s+\[([x\s])\]\s+(.+)$`)

	var items []Item
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r\n")

		if headerMatch := headerRegex.FindStringSubmatch(line); headerMatch != nil {
			level := len(headerMatch[1])
			content := headerMatch[2]
			items = append(items, Item{
				Type:      TypeSection,
				Level:     level,
				Content:   content,
				Checked:   nil,
				Children:  []Item{},
				Collapsed: false,
			})
		} else if taskMatch := taskRegex.FindStringSubmatch(line); taskMatch != nil {
			indent := len(taskMatch[1])
			checked := taskMatch[2] == "x"
			content := taskMatch[3]
			items = append(items, Item{
				Type:      TypeTask,
				Level:     indent,
				Content:   content,
				Checked:   &checked,
				Children:  []Item{},
				Collapsed: false,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}
	return items, nil
}

// Model represents the application state
type Model struct {
	items           []Item
	cursor          int
	filename        string
	visibleItems    []int      // indices of items that are currently visible (sections and tasks)
	scrollOffset    int        // first visible item index in the viewport
	inputMode       bool       // whether we're in input mode
	inputText       string     // text being typed
	inputCursor     int        // cursor position within input text
	editingIndex    int        // index of item being edited (-1 for new item)
	newSectionLevel int        // level of section being created (0 = task)
	hMode           bool       // whether we're waiting for a number after 'h'
	helpMode        bool       // whether we're showing the help screen
	dirty           bool       // whether the file has unsaved changes
	fileModTime     time.Time  // file modification time
	width           int        // terminal width
	height          int        // terminal height
	bannerMode      BannerMode // whether to hide the banner
}

// BannerMode represents the mode for displaying the banner
type BannerMode int

const (
	// BannerEnabled means the banner is shown
	BannerEnabled BannerMode = iota + 1
	// BannerDisabled means the banner is hidden
	BannerDisabled
)

// initialModel initializes the application model with data from a Markdown file
func initialModel(filename string, bannerMode BannerMode) (Model, error) {
	items, err := parseMarkdownFile(filename)
	if err != nil {
		return Model{}, fmt.Errorf("unable to parse file %q, err: %w", filename, err)
	}

	// Get file modification time
	var modTime time.Time
	if fileInfo, err := os.Stat(filename); err == nil {
		modTime = fileInfo.ModTime()
	} else {
		modTime = time.Now()
	}

	m := Model{
		items:           items,
		cursor:          0,
		filename:        filename,
		visibleItems:    []int{},
		scrollOffset:    0,
		inputMode:       false,
		inputText:       "",
		inputCursor:     0,
		editingIndex:    -1,
		newSectionLevel: 0,
		hMode:           false,
		dirty:           false,
		fileModTime:     modTime,
		width:           80, // default width, will be updated by WindowSizeMsg
		height:          24, // default height, will be updated by WindowSizeMsg
		bannerMode:      bannerMode,
	}

	m.updateVisibleItems()

	return m, nil
}

// Init returns a command to initialize the model
func (m Model) Init() tea.Cmd {
	return tea.WindowSize()
}

// updateVisibleItems updates the list of visible items based on the current state
func (m *Model) updateVisibleItems() {
	m.visibleItems = []int{}
	var sectionStack []*Item // Stack to track nested sections

	for i, item := range m.items {
		switch item.Type {
		case TypeSection:
			// Update section stack based on level
			for len(sectionStack) > 0 && sectionStack[len(sectionStack)-1].Level >= item.Level {
				sectionStack = sectionStack[:len(sectionStack)-1]
			}

			// Check if this section should be visible (no collapsed parent sections)
			visible := true
			for _, section := range sectionStack {
				if section.Collapsed {
					visible = false
					break
				}
			}

			if visible {
				m.visibleItems = append(m.visibleItems, i)
			}

			// Add current section to stack
			itemCopy := m.items[i]
			sectionStack = append(sectionStack, &itemCopy)

		case TypeTask:
			// Check if task should be visible (no collapsed sections in stack)
			visible := true
			for _, section := range sectionStack {
				if section.Collapsed {
					visible = false
					break
				}
			}

			if visible {
				m.visibleItems = append(m.visibleItems, i)
			}
		}
	}

	m.adjustCursor()
}

// adjustCursor ensures the cursor is within valid bounds and viewport
func (m *Model) adjustCursor() {
	if m.cursor >= len(m.visibleItems) && len(m.visibleItems) > 0 {
		m.cursor = len(m.visibleItems) - 1
	}
	m.ensureCursorInViewport()
}

// deleteItem deletes an item and all its children if it's a section
func (m *Model) deleteItem(itemIndex int) {
	if itemIndex < 0 || itemIndex >= len(m.items) {
		return
	}

	item := m.items[itemIndex]

	switch item.Type {
	case TypeSection:
		// Delete section and all items under it
		sectionLevel := item.Level
		deleteCount := 1 // Start with the section itself

		// Count items that should be deleted (all items under this section)
		for i := itemIndex + 1; i < len(m.items); i++ {
			nextItem := m.items[i]

			// If we encounter another section at the same or higher level, stop
			if nextItem.Type == TypeSection && nextItem.Level <= sectionLevel {
				break
			}

			// This item is under the section (either a subsection or task)
			deleteCount++
		}

		// Remove all items from itemIndex to itemIndex+deleteCount-1
		m.items = slices.Delete(m.items, itemIndex, itemIndex+deleteCount)

	case TypeTask:
		// Delete single task
		m.items = slices.Delete(m.items, itemIndex, itemIndex+1)
	}
}

// collapseAll sets all sections to collapsed state
func (m *Model) collapseAll() {
	for i := range m.items {
		if m.items[i].Type == TypeSection {
			m.items[i].Collapsed = true
		}
	}
}

// expandAll sets all sections to expanded state
func (m *Model) expandAll() {
	for i := range m.items {
		if m.items[i].Type == TypeSection {
			m.items[i].Collapsed = false
		}
	}
}

// getCurrentItemIndex returns the index of the currently selected item in the visible items list
func (m Model) getCurrentItemIndex() int {
	if len(m.visibleItems) == 0 {
		return -1
	}
	return m.visibleItems[m.cursor]
}

// calculateViewportHeight calculates how many lines are available for displaying items
func (m Model) calculateViewportHeight() int {
	usedHeight := 0

	// Banner takes 6 lines (5 lines of ASCII art + 1 blank line after)
	if m.bannerMode == BannerEnabled {
		usedHeight += 6
	}

	// Input field takes 3 lines when active (blank line + prompt+input + help text)
	if m.inputMode {
		usedHeight += 3
	}

	// Footer takes 2 lines (blank line + footer content)
	usedHeight += 2

	// Calculate available height, ensuring minimum of 3 lines for content
	availableHeight := m.height - usedHeight
	if availableHeight < 3 {
		availableHeight = 3
	}

	return availableHeight
}

// clampScrollOffset ensures scrollOffset is within valid bounds
func (m *Model) clampScrollOffset() {
	// Ensure scrollOffset doesn't go negative
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}

	// Ensure scrollOffset doesn't go past the last item
	maxScrollOffset := len(m.visibleItems) - 1
	if maxScrollOffset < 0 {
		maxScrollOffset = 0
	}
	if m.scrollOffset > maxScrollOffset {
		m.scrollOffset = maxScrollOffset
	}
}

// ensureCursorInViewport adjusts scrollOffset to ensure the cursor is visible in the viewport
func (m *Model) ensureCursorInViewport() {
	if len(m.visibleItems) == 0 {
		return
	}

	viewportHeight := m.calculateViewportHeight()

	// If cursor is above the viewport, scroll up
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}

	// If cursor is below the viewport, scroll down just enough to show it
	if m.cursor >= m.scrollOffset+viewportHeight {
		m.scrollOffset = m.cursor - viewportHeight + 1
	}

	m.clampScrollOffset()
}

// handleInputMode processes key messages while in input mode (editing or creating items)
func (m Model) handleInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	exitInputMode := func() {
		m.inputMode = false
		m.inputText = ""
		m.inputCursor = 0
		m.editingIndex = -1
		m.newSectionLevel = 0
	}

	// Define closure for saving input
	saveInput := func() {
		switch {
		case m.editingIndex >= 0:
			// Editing existing item
			m.items[m.editingIndex].Content = m.inputText
		default:
			// Creating new item
			var newItem Item

			if m.newSectionLevel > 0 {
				// Create new section
				newItem = Item{
					Type:      TypeSection,
					Level:     m.newSectionLevel,
					Content:   m.inputText,
					Checked:   nil,
					Children:  []Item{},
					Collapsed: false,
				}
			} else {
				// Create new task
				newItem = Item{
					Type:      TypeTask,
					Level:     0,
					Content:   m.inputText,
					Checked:   new(bool),
					Children:  []Item{},
					Collapsed: false,
				}
			}

			// Handle empty file case
			if len(m.items) == 0 {
				// Add as first item
				m.items = append(m.items, newItem)
				m.updateVisibleItems()
				m.cursor = 0
			} else {
				// Insert after current item
				itemIndex := m.getCurrentItemIndex()
				if itemIndex >= 0 {
					// Adjust task level based on context
					if m.newSectionLevel == 0 && m.items[itemIndex].Type == TypeSection {
						// Task after section should be at level 0
						newItem.Level = 0
					} else if m.newSectionLevel == 0 && m.items[itemIndex].Type == TypeTask {
						// Task after task should match its level
						newItem.Level = m.items[itemIndex].Level
					}

					insertIndex := itemIndex + 1
					m.items = append(m.items[:insertIndex], append([]Item{newItem}, m.items[insertIndex:]...)...)
					m.updateVisibleItems()

					// Find new position in visible items
					for i, idx := range m.visibleItems {
						if idx == insertIndex {
							m.cursor = i
							break
						}
					}
				}
			}
		}

		m.dirty = true
		exitInputMode()
	}

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		saveInput()
	case "esc":
		exitInputMode()
	case "left":
		// Move cursor left in input
		if m.inputCursor > 0 {
			m.inputCursor--
		}
	case "right":
		// Move cursor right in input
		if m.inputCursor < utf8.RuneCountInString(m.inputText) {
			m.inputCursor++
		}
	case "ctrl+a":
		// Move cursor to beginning
		m.inputCursor = 0
	case "ctrl+e":
		// Move cursor to end
		m.inputCursor = utf8.RuneCountInString(m.inputText)
	case "backspace":
		if m.inputCursor > 0 {
			// Convert to runes for proper Unicode handling
			runes := []rune(m.inputText)
			// Remove character before cursor
			m.inputText = string(runes[:m.inputCursor-1]) + string(runes[m.inputCursor:])
			m.inputCursor--
		}
	default:
		// Add character to input at cursor position
		if utf8.RuneCountInString(msg.String()) == 1 {
			// Convert to runes for proper Unicode handling
			runes := []rune(m.inputText)
			// Insert character at cursor position
			newRunes := make([]rune, 0, len(runes)+1)
			newRunes = append(newRunes, runes[:m.inputCursor]...)
			newRunes = append(newRunes, []rune(msg.String())...)
			newRunes = append(newRunes, runes[m.inputCursor:]...)
			m.inputText = string(newRunes)
			m.inputCursor++
		}
	}
	return m, nil
}

// handleHMode processes key messages while in h-mode (waiting for section level input)
func (m Model) handleHMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	handle := func(level int) {
		m.inputMode = true
		m.inputText = ""
		m.inputCursor = 0
		m.editingIndex = -1
		m.newSectionLevel = level
		m.hMode = false
	}

	switch msg.String() {
	case "1":
		handle(1)
	case "2":
		handle(2)
	case "3":
		handle(3)
	case "4":
		handle(4)
	case "5":
		handle(5)
	case "6":
		handle(6)
	default:
		// Cancel h-mode on any other key
		m.hMode = false
	}
	return m, nil
}

// handleHelpMode processes key messages while in help mode
func (m Model) handleHelpMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc", "?":
		// Close help screen
		m.helpMode = false
	}
	return m, nil
}

// handleNavigation processes key messages for navigation and actions
func (m Model) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(m.visibleItems)-1 {
			m.cursor++
			m.ensureCursorInViewport()
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.ensureCursorInViewport()
		}
	case " ":
		itemIndex := m.getCurrentItemIndex()
		if itemIndex >= 0 && m.items[itemIndex].Type == TypeTask {
			checked := !*m.items[itemIndex].Checked
			m.items[itemIndex].Checked = &checked
			m.dirty = true
		}
	case "enter":
		itemIndex := m.getCurrentItemIndex()
		if itemIndex >= 0 && m.items[itemIndex].Type == TypeSection {
			m.items[itemIndex].Collapsed = !m.items[itemIndex].Collapsed
			m.updateVisibleItems()
		}
	case "left":
		itemIndex := m.getCurrentItemIndex()
		if itemIndex >= 0 && m.items[itemIndex].Type == TypeSection {
			m.items[itemIndex].Collapsed = true
			m.updateVisibleItems()
		}
	case "right":
		itemIndex := m.getCurrentItemIndex()
		if itemIndex >= 0 && m.items[itemIndex].Type == TypeSection {
			m.items[itemIndex].Collapsed = false
			m.updateVisibleItems()
		}
	case "n":
		// Enter input mode for new task
		m.inputMode = true
		m.inputText = ""
		m.inputCursor = 0
		m.editingIndex = -1
		m.newSectionLevel = 0
	case "h":
		// Enter h-mode (waiting for number)
		m.hMode = true
	case "e":
		// Enter input mode for editing current item (task or section)
		itemIndex := m.getCurrentItemIndex()
		if itemIndex >= 0 {
			m.inputMode = true
			m.inputText = m.items[itemIndex].Content
			m.inputCursor = utf8.RuneCountInString(m.inputText) // Position cursor at end
			m.editingIndex = itemIndex
		}
	case "alt+j", "alt+down":
		itemIndex := m.getCurrentItemIndex()
		if itemIndex >= 0 && itemIndex < len(m.items)-1 {
			m.items[itemIndex], m.items[itemIndex+1] = m.items[itemIndex+1], m.items[itemIndex]
			m.updateVisibleItems()
			m.dirty = true
			if m.cursor < len(m.visibleItems)-1 {
				m.cursor++
			}
		}
	case "alt+k", "alt+up":
		itemIndex := m.getCurrentItemIndex()
		if itemIndex > 0 {
			m.items[itemIndex], m.items[itemIndex-1] = m.items[itemIndex-1], m.items[itemIndex]
			m.updateVisibleItems()
			m.dirty = true
			if m.cursor > 0 {
				m.cursor--
			}
		}
	case "d":
		// Delete current item
		itemIndex := m.getCurrentItemIndex()
		if itemIndex >= 0 {
			m.deleteItem(itemIndex)
			m.updateVisibleItems()
			m.dirty = true
			m.adjustCursor()
		}
	case "s":
		err := m.saveToFile()
		if err != nil {
			return m, nil
		}
		m.dirty = false
		// Update file modification time after saving
		if fileInfo, err := os.Stat(m.filename); err == nil {
			m.fileModTime = fileInfo.ModTime()
		}
	case "-":
		// Collapse all sections
		m.collapseAll()
		m.updateVisibleItems()
	case "+":
		// Expand all sections
		m.expandAll()
		m.updateVisibleItems()
	case "ctrl+f":
		// Page forward (down) - advance viewport to next page
		viewportHeight := m.calculateViewportHeight()
		if len(m.visibleItems) > 0 {
			// Calculate where we would scroll to
			newScrollOffset := m.scrollOffset + viewportHeight

			// If the new scroll position would go past the end, don't scroll
			if newScrollOffset < len(m.visibleItems) {
				m.scrollOffset = newScrollOffset
				m.cursor = m.scrollOffset
			}
			// If we're already showing the last page (partial or full), don't scroll
		}
	case "ctrl+b":
		// Page backward (up) - move viewport to previous page
		viewportHeight := m.calculateViewportHeight()
		if len(m.visibleItems) > 0 {
			// Move scrollOffset back by viewport height
			newScrollOffset := m.scrollOffset - viewportHeight
			m.scrollOffset = max(newScrollOffset, 0)

			// Position cursor at the top of the new viewport
			m.cursor = m.scrollOffset
		}
	case "?":
		// Show help screen
		m.helpMode = true
	}

	return m, nil
}

// Update handles user input and updates the model state
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update terminal dimensions
		m.width = msg.Width
		m.height = msg.Height
		// Clear screen to prevent UI corruption from wrapping when resizing smaller
		return m, tea.ClearScreen
	case tea.KeyMsg:
		// Handle help mode separately
		if m.helpMode {
			return m.handleHelpMode(msg)
		}

		// Handle input mode separately
		if m.inputMode {
			return m.handleInputMode(msg)
		}

		// Handle h-mode (waiting for number after 'h')
		if m.hMode {
			return m.handleHMode(msg)
		}

		// Normal navigation mode
		return m.handleNavigation(msg)
	}
	return m, nil
}

// saveToFile saves the current task list to the Markdown file
func (m Model) saveToFile() error {
	file, err := os.Create(m.filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for i, item := range m.items {
		switch item.Type {
		case TypeSection:
			// Add blank line before section (except for first item)
			if i > 0 {
				_, err := fmt.Fprintf(writer, "\n")
				if err != nil {
					return err
				}
			}
			_, err := fmt.Fprintf(writer, "%s %s\n", strings.Repeat("#", item.Level), item.Content)
			if err != nil {
				return err
			}
		case TypeTask:
			checkbox := "[ ]"
			if item.Checked != nil && *item.Checked {
				checkbox = "[x]"
			}
			indent := strings.Repeat(" ", item.Level)
			_, err := fmt.Fprintf(writer, "%s- %s %s\n", indent, checkbox, item.Content)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// renderBanner generates a stylized banner for the application
func (m Model) renderBanner(w io.Writer) {
	if m.bannerMode != BannerEnabled {
		return
	}

	// Block-style ASCII art for "Tasks" with left arrow pattern like Gemini CLI
	lines := []string{
		"███           ████████  █████   ███████ ██   ██ ███████",
		"   ███           ██    ██   ██  ██      ██  ██  ██     ",
		"     ███         ██    ███████  ███████ █████   ███████",
		"   ███           ██    ██   ██       ██ ██  ██       ██",
		"███              ██    ██   ██  ███████ ██   ██ ███████",
	}

	// Apply gradient coloring - each character gets a color based on position
	for _, line := range lines {
		totalWidth := len(line)
		for i, char := range line {
			// Calculate gradient position (0.0 to 1.0)
			position := float64(i) / float64(totalWidth-1)

			// Choose color based on position
			var style lipgloss.Style
			switch {
			case position < 0.16:
				style = bannerGreen1
			case position < 0.32:
				style = bannerGreen2
			case position < 0.48:
				style = bannerGreen3
			case position < 0.64:
				style = bannerGreen4
			case position < 0.80:
				style = bannerGreen5
			case position < 0.96:
				style = bannerGreen6
			default:
				style = bannerWhite
			}

			io.WriteString(w, style.Render(string(char)))
		}
		io.WriteString(w, "\n")
	}

	io.WriteString(w, "\n")
}

func (m Model) renderVisibleItems(w io.Writer) {
	var sectionStack []*Item // Stack to track sections for indentation

	viewportHeight := m.calculateViewportHeight()
	endIdx := m.scrollOffset + viewportHeight
	if endIdx > len(m.visibleItems) {
		endIdx = len(m.visibleItems)
	}

	// Only render items within the viewport
	for viewportIdx := m.scrollOffset; viewportIdx < endIdx; viewportIdx++ {
		itemIndex := m.visibleItems[viewportIdx]
		item := m.items[itemIndex]

		// Update section stack by finding all sections up to this item
		sectionStack = []*Item{}
		for j := range itemIndex {
			if m.items[j].Type == TypeSection {
				// Update stack based on level
				for len(sectionStack) > 0 && sectionStack[len(sectionStack)-1].Level >= m.items[j].Level {
					sectionStack = sectionStack[:len(sectionStack)-1]
				}
				sectionStack = append(sectionStack, &m.items[j])
			}
		}

		switch item.Type {
		case TypeSection:
			// Calculate indentation (level 1 = 0 spaces, level 2 = 2 spaces, etc.)
			indent := strings.Repeat("  ", item.Level-1)

			// Style the arrow based on collapsed state
			var arrow string
			if item.Collapsed {
				arrow = arrowCollapsedStyle.Render("▶")
			} else {
				arrow = arrowExpandedStyle.Render("▼")
			}

			// Style section text based on collapsed state
			var sectionText string
			if item.Collapsed {
				sectionText = sectionCollapsedStyle.Render(item.Content)
			} else {
				sectionText = sectionStyle.Render(item.Content)
			}

			sectionLine := fmt.Sprintf("%s %s", arrow, sectionText)

			// Calculate width accounting for indentation
			indentWidth := len(indent) + 2                 // indent + "  "
			availableWidth := max(m.width-indentWidth, 10) // minimum 10 chars

			// Highlight current section
			if m.cursor == viewportIdx {
				highlightStyle := selectedStyle.Width(availableWidth)

				var styledContent string
				if item.Collapsed {
					styledContent = highlightStyle.Render(arrowCollapsedStyle.Render("▶") + " " + sectionCollapsedStyle.Render(item.Content))
				} else {
					styledContent = highlightStyle.Render(arrowExpandedStyle.Render("▼") + " " + sectionStyle.Render(item.Content))
				}
				fmt.Fprintf(w, "%s  %s\n", indent, styledContent)
			} else {
				// Apply width constraint to non-highlighted items too
				normalStyle := lipgloss.NewStyle().Width(availableWidth)
				styledContent := normalStyle.Render(sectionLine)
				fmt.Fprintf(w, "%s  %s\n", indent, styledContent)
			}

		case TypeTask:
			// Style checkbox and task text based on completion status
			var checkbox, taskText string
			if item.Checked != nil && *item.Checked {
				checkbox = checkedBoxStyle.Render("☒")
				taskText = taskCompletedStyle.Render(item.Content)
			} else {
				checkbox = uncheckedBoxStyle.Render("☐")
				taskText = taskPendingStyle.Render(item.Content)
			}

			// Task indentation is based on the deepest section level + 1
			taskIndent := ""
			if len(sectionStack) > 0 {
				deepestLevel := sectionStack[len(sectionStack)-1].Level
				taskIndent = strings.Repeat("  ", deepestLevel)
			}

			taskLine := fmt.Sprintf("%s %s", checkbox, taskText)

			// Calculate width accounting for indentation
			indentWidth := len(taskIndent) + 2             // taskIndent + "  "
			availableWidth := max(m.width-indentWidth, 10) // minimum 10 chars

			// Style the current task differently
			if m.cursor == viewportIdx {
				highlightStyle := selectedStyle.Width(availableWidth)

				var styledContent string
				if item.Checked != nil && *item.Checked {
					styledContent = highlightStyle.Render(checkedBoxStyle.Render("☒") + " " + taskCompletedStyle.Render(item.Content))
				} else {
					styledContent = highlightStyle.Render(uncheckedBoxStyle.Render("☐") + " " + taskPendingStyle.Render(item.Content))
				}
				fmt.Fprintf(w, "%s  %s\n", taskIndent, styledContent)
			} else {
				// Apply width constraint to non-highlighted items too
				normalStyle := lipgloss.NewStyle().Width(availableWidth)
				styledContent := normalStyle.Render(taskLine)
				fmt.Fprintf(w, "%s  %s\n", taskIndent, styledContent)
			}
		}
	}
}

// renderInput renders the input field when in input mode
func (m Model) renderInput(w io.Writer) {
	if !m.inputMode {
		return
	}

	io.WriteString(w, "\n")

	var prompt string
	if m.editingIndex == -1 {
		if m.newSectionLevel > 0 {
			prompt = inputPromptStyle.Render(fmt.Sprintf("New h%d section:", m.newSectionLevel))
		} else {
			prompt = inputPromptStyle.Render("New task:")
		}
	} else {
		if m.items[m.editingIndex].Type == TypeSection {
			prompt = inputPromptStyle.Render("Edit section:")
		} else {
			prompt = inputPromptStyle.Render("Edit task:")
		}
	}

	// Create input content with cursor at the correct position
	runes := []rune(m.inputText)
	textBeforeCursor := string(runes[:m.inputCursor])
	textAfterCursor := string(runes[m.inputCursor:])
	inputContent := textBeforeCursor + "│" + textAfterCursor
	inputField := inputStyle.Render(inputContent)
	io.WriteString(w, prompt+" "+inputField+"\n")
	io.WriteString(w, helpStyle.Render("Press Enter to save, Esc to cancel, ←/→ to navigate, Ctrl+A/E for home/end")+"\n")
}

// renderFooter renders the status footer
func (m Model) renderFooter(w io.Writer) {
	// Left side: dirty status and help indicator
	var leftText string
	if m.dirty {
		leftText = dirtyIndicatorStyle.Render("● Modified")
	} else {
		leftText = lastUpdateStyle.Render("Saved")
	}

	// Add help indicator
	helpIndicator := lipgloss.NewStyle().Foreground(mutedColor).Render(" • Press ? for help")
	leftText += helpIndicator

	io.WriteString(w, leftText)

	// Right side: filename and modification time
	filename := m.filename
	modTime := m.fileModTime.Format("15:04:05") // 15=hour, 04=minute, 05=second (all zero-padded)

	filenameText := footerFilenameStyle.Render(filename)
	timeText := footerTimeStyle.Render(modTime)

	rightContent := filenameText + timeText
	// Calculate spacing to right-align the content
	// Use lipgloss.Width to get actual display width of styled content
	leftWidth := lipgloss.Width(leftText)
	rightWidth := lipgloss.Width(rightContent)
	padding := max(m.width-leftWidth-rightWidth, 1)

	io.WriteString(w, strings.Repeat(" ", padding))
	io.WriteString(w, rightContent)
}

// renderHelpScreen renders the full-screen help screen with all available shortcuts
func (m Model) renderHelpScreen(w io.Writer) {
	if !m.helpMode {
		return
	}

	// Full-screen background style
	backgroundStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#1F2937")).
		Foreground(textColor).
		Width(m.width).
		Height(m.height).
		Padding(2, 4)

	headerStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Align(lipgloss.Center)

	categoryStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(successColor).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(textColor)

	footerStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true).
		Align(lipgloss.Center)

	// Help content
	content := headerStyle.Render("📖 KEYBOARD SHORTCUTS") + "\n\n"

	// Helper function to format key-description pairs with consistent alignment
	formatKeyDesc := func(keys, desc string) string {
		styledKeys := keyStyle.Render(keys)
		// Use a fixed width of 20 characters for the key column
		keyColumn := lipgloss.NewStyle().Width(20).Render(styledKeys)
		return fmt.Sprintf("  %s %s", keyColumn, descStyle.Render(desc))
	}

	// Navigation section
	content += categoryStyle.Render("Navigation:") + "\n"
	content += formatKeyDesc("j / ↓", "Move cursor down") + "\n"
	content += formatKeyDesc("k / ↑", "Move cursor up") + "\n"
	content += formatKeyDesc("Ctrl+F", "Page forward (down)") + "\n"
	content += formatKeyDesc("Ctrl+B", "Page backward (up)") + "\n"
	content += "\n"

	// Task actions section
	content += categoryStyle.Render("Task Actions:") + "\n"
	content += formatKeyDesc("space", "Toggle task completion (☐/☒)") + "\n"
	content += formatKeyDesc("n", "Create new task") + "\n"
	content += formatKeyDesc("e", "Edit current task/section") + "\n"
	content += formatKeyDesc("d", "Delete current item") + "\n"
	content += "\n"

	// Section actions section
	content += categoryStyle.Render("Section Actions:") + "\n"
	content += formatKeyDesc("enter", "Toggle section expand/collapse") + "\n"
	content += formatKeyDesc("← / →", "Collapse/expand current section") + "\n"
	content += formatKeyDesc("h1-h6", "Create new section (level 1-6)") + "\n"
	content += formatKeyDesc("-", "Collapse all sections") + "\n"
	content += formatKeyDesc("+", "Expand all sections") + "\n"
	content += "\n"

	// Item movement section
	content += categoryStyle.Render("Item Movement:") + "\n"
	content += formatKeyDesc("Alt+j / Alt+↓", "Move item down") + "\n"
	content += formatKeyDesc("Alt+k / Alt+↑", "Move item up") + "\n"
	content += "\n"

	// File operations section
	content += categoryStyle.Render("File Operations:") + "\n"
	content += formatKeyDesc("s", "Save changes to file") + "\n"
	content += "\n"

	// Input navigation section
	content += categoryStyle.Render("Input Navigation:") + "\n"
	content += formatKeyDesc("← / →", "Move cursor left/right in input field") + "\n"
	content += formatKeyDesc("Ctrl+A", "Move cursor to beginning of input") + "\n"
	content += formatKeyDesc("Ctrl+E", "Move cursor to end of input") + "\n"
	content += "\n"

	// General section
	content += categoryStyle.Render("General:") + "\n"
	content += formatKeyDesc("?", "Show/hide this help") + "\n"
	content += formatKeyDesc("q / Ctrl+C", "Quit application") + "\n"

	content += "\n\n"
	content += footerStyle.Render("Press q, ? or Esc to close this help screen")

	// Render full-screen background with content
	fullScreen := backgroundStyle.Render(content)
	io.WriteString(w, fullScreen)
}

// View renders the current model state as a string
func (m Model) View() string {
	var s strings.Builder

	// If help mode is active, show only the help screen
	if m.helpMode {
		m.renderHelpScreen(&s)
		return s.String()
	}

	// Render banner if not disabled
	m.renderBanner(&s)

	if len(m.items) == 0 {
		noTasksMsg := taskPendingStyle.Render("No tasks found. Press 'q' to quit.")
		s.WriteString(noTasksMsg + "\n")
	} else {
		// Render visible items
		m.renderVisibleItems(&s)
	}

	// Render input if necessary
	m.renderInput(&s)

	// Render footer
	s.WriteString("\n")
	m.renderFooter(&s)

	return s.String()
}

// getVersion returns version information using build info
func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	version := info.Main.Version
	if version == "(devel)" || version == "" {
		// Try to get revision from build info
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				if len(setting.Value) >= 7 {
					return "dev-" + setting.Value[:7]
				}
				return "dev-" + setting.Value
			}
		}
		return "dev"
	}

	return version
}

func main() {
	defaultNoBanner := os.Getenv("TASKS_NO_BANNER") != ""

	var showVersion = flag.Bool("version", false, "show version information")
	var noBanner = flag.Bool("no-banner", defaultNoBanner, "disable the banner display (can be set with the TASKS_NO_BANNER environment variable)")
	flag.Parse()

	if *showVersion {
		fmt.Println(getVersion())
		return
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: tasks [--version] [--no-banner] <markdown-file>")
		os.Exit(1)
	}

	filename := args[0]

	// Determine banner mode
	bannerMode := BannerEnabled
	if *noBanner {
		bannerMode = BannerDisabled
	}

	model, err := initialModel(filename, bannerMode)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
