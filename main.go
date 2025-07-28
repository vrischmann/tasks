package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

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

	// Section styles
	sectionStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	sectionCollapsedStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Bold(true)

	// Task styles
	taskCompletedStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Strikethrough(true)

	taskPendingStyle = lipgloss.NewStyle().
				Foreground(textColor)

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
	visibleItems    []int     // indices of items that are currently visible (sections and tasks)
	inputMode       bool      // whether we're in input mode
	inputText       string    // text being typed
	editingIndex    int       // index of item being edited (-1 for new item)
	newSectionLevel int       // level of section being created (0 = task)
	hMode           bool      // whether we're waiting for a number after 'h'
	dirty           bool      // whether the file has unsaved changes
	fileModTime     time.Time // file modification time
	width           int       // terminal width
	height          int       // terminal height
}

// initialModel initializes the application model with data from a Markdown file
func initialModel(filename string) Model {
	items, err := parseMarkdownFile(filename)
	if err != nil {
		items = []Item{}
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
		inputMode:       false,
		inputText:       "",
		editingIndex:    -1,
		newSectionLevel: 0,
		hMode:           false,
		dirty:           false,
		fileModTime:     modTime,
		width:           80, // default width, will be updated by WindowSizeMsg
		height:          24, // default height, will be updated by WindowSizeMsg
	}

	m.updateVisibleItems()
	return m
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

	// Ensure cursor is within bounds
	if m.cursor >= len(m.visibleItems) {
		m.cursor = len(m.visibleItems) - 1
	}
	if m.cursor < 0 && len(m.visibleItems) > 0 {
		m.cursor = 0
	}
}

// getCurrentItemIndex returns the index of the currently selected item in the visible items list
func (m Model) getCurrentItemIndex() int {
	if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
		return m.visibleItems[m.cursor]
	}
	return -1
}

// handleInputMode processes key messages while in input mode (editing or creating items)
func (m Model) handleInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		// Save the input
		if m.editingIndex == -1 {
			// Creating new item
			itemIndex := m.getCurrentItemIndex()
			if itemIndex >= 0 {
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
					if m.items[itemIndex].Type == TypeSection {
						newItem = Item{
							Type:      TypeTask,
							Level:     0,
							Content:   m.inputText,
							Checked:   new(bool),
							Children:  []Item{},
							Collapsed: false,
						}
					} else {
						newItem = Item{
							Type:      TypeTask,
							Level:     m.items[itemIndex].Level,
							Content:   m.inputText,
							Checked:   new(bool),
							Children:  []Item{},
							Collapsed: false,
						}
					}
				}

				insertIndex := itemIndex + 1
				m.items = append(m.items[:insertIndex], append([]Item{newItem}, m.items[insertIndex:]...)...)
				m.updateVisibleItems()
				m.dirty = true

				// Find new position in visible items
				for i, idx := range m.visibleItems {
					if idx == insertIndex {
						m.cursor = i
						break
					}
				}
			}
		} else {
			// Editing existing item
			m.items[m.editingIndex].Content = m.inputText
			m.dirty = true
		}
		// Exit input mode
		m.inputMode = false
		m.inputText = ""
		m.editingIndex = -1
		m.newSectionLevel = 0
	case "esc":
		// Cancel input
		m.inputMode = false
		m.inputText = ""
		m.editingIndex = -1
		m.newSectionLevel = 0
	case "backspace":
		if len(m.inputText) > 0 {
			m.inputText = m.inputText[:len(m.inputText)-1]
		}
	default:
		// Add character to input
		if len(msg.String()) == 1 {
			m.inputText += msg.String()
		}
	}
	return m, nil
}

// handleHMode processes key messages while in h-mode (waiting for section level input)
func (m Model) handleHMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "1":
		m.inputMode = true
		m.inputText = ""
		m.editingIndex = -1
		m.newSectionLevel = 1
		m.hMode = false
	case "2":
		m.inputMode = true
		m.inputText = ""
		m.editingIndex = -1
		m.newSectionLevel = 2
		m.hMode = false
	case "3":
		m.inputMode = true
		m.inputText = ""
		m.editingIndex = -1
		m.newSectionLevel = 3
		m.hMode = false
	case "4":
		m.inputMode = true
		m.inputText = ""
		m.editingIndex = -1
		m.newSectionLevel = 4
		m.hMode = false
	case "5":
		m.inputMode = true
		m.inputText = ""
		m.editingIndex = -1
		m.newSectionLevel = 5
		m.hMode = false
	case "6":
		m.inputMode = true
		m.inputText = ""
		m.editingIndex = -1
		m.newSectionLevel = 6
		m.hMode = false
	default:
		// Cancel h-mode on any other key
		m.hMode = false
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
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
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

	for _, item := range m.items {
		switch item.Type {
		case TypeSection:
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
func renderBanner(w io.Writer) {
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
}

func (m Model) renderVisibleItems(w io.Writer) {
	var sectionStack []*Item // Stack to track sections for indentation

	for visIdx, itemIndex := range m.visibleItems {
		item := m.items[itemIndex]

		// Update section stack by finding all sections up to this item
		sectionStack = []*Item{}
		for i := range itemIndex {
			if m.items[i].Type == TypeSection {
				// Update stack based on level
				for len(sectionStack) > 0 && sectionStack[len(sectionStack)-1].Level >= m.items[i].Level {
					sectionStack = sectionStack[:len(sectionStack)-1]
				}
				sectionStack = append(sectionStack, &m.items[i])
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
			if m.cursor == visIdx {
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
			if m.cursor == visIdx {
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

	// Create input content with cursor inside the border
	inputContent := m.inputText + "│"
	inputField := inputStyle.Render(inputContent)
	io.WriteString(w, prompt+" "+inputField+"\n")
	io.WriteString(w, helpStyle.Render("Press Enter to save, Esc to cancel")+"\n")
}

// renderFooter renders the status footer
func (m Model) renderFooter(w io.Writer) {
	// Left side: dirty status
	var leftText string
	if m.dirty {
		leftText = dirtyIndicatorStyle.Render("● Modified")
	} else {
		leftText = lastUpdateStyle.Render("Saved")
	}
	io.WriteString(w, leftText)

	// Right side: filename
	filename := lastUpdateStyle.Render("File: " + m.filename)

	// Calculate spacing to right-align the filename
	leftTextPlain := ""
	if m.dirty {
		leftTextPlain = "● Modified"
	} else {
		leftTextPlain = "Saved"
	}

	rightText := "File: " + m.filename
	padding := max(m.width-len(leftTextPlain)-len(rightText), 1)

	io.WriteString(w, strings.Repeat(" ", padding))
	io.WriteString(w, filename)
}

// View renders the current model state as a string
func (m Model) View() string {
	var s strings.Builder

	// Render banner
	renderBanner(&s)
	s.WriteString("\n")

	if len(m.items) == 0 {
		noTasksMsg := taskPendingStyle.Render("No tasks found. Press 'q' to quit.")
		s.WriteString(noTasksMsg + "\n")
		return s.String()
	}

	// Render visible items
	m.renderVisibleItems(&s)

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
	var showVersion = flag.Bool("version", false, "show version information")
	flag.Parse()

	if *showVersion {
		fmt.Println(getVersion())
		return
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: tasks [--version] <markdown-file>")
		os.Exit(1)
	}

	filename := args[0]

	p := tea.NewProgram(initialModel(filename))

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
