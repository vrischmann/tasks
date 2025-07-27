package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ItemType int

const (
	TypeSection ItemType = iota
	TypeTask
)

// Define color scheme and styles
var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED")   // Purple
	accentColor    = lipgloss.Color("#EC4899")   // Pink
	successColor   = lipgloss.Color("#10B981")   // Green
	mutedColor     = lipgloss.Color("#6B7280")   // Gray
	backgroundColor = lipgloss.Color("#1F2937")  // Dark gray
	textColor      = lipgloss.Color("#F9FAFB")   // Light gray
	
	// Title style
	titleStyle = lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor)
	
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
	
	// Main container style
	containerStyle = lipgloss.NewStyle().
		Width(80).
		Padding(0, 2)
	
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
)

type Item struct {
	Type      ItemType
	Level     int
	Content   string
	Checked   *bool
	Children  []Item
	Collapsed bool
}

type Model struct {
	items       []Item
	cursor      int
	filename    string
	visibleItems []int // indices of items that are currently visible (sections and tasks)
	inputMode   bool   // whether we're in input mode
	inputText   string // text being typed
	editingIndex int   // index of item being edited (-1 for new item)
	newSectionLevel int // level of section being created (0 = task)
	hMode       bool   // whether we're waiting for a number after 'h'
}

func parseMarkdownFile(filename string) ([]Item, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var items []Item
	scanner := bufio.NewScanner(file)
	
	headerRegex := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	taskRegex := regexp.MustCompile(`^(\s*)-\s+\[([x\s])\]\s+(.+)$`)
	
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
	
	return items, scanner.Err()
}

func initialModel(filename string) Model {
	items, err := parseMarkdownFile(filename)
	if err != nil {
		items = []Item{}
	}
	
	m := Model{
		items:        items,
		cursor:       0,
		filename:     filename,
		visibleItems: []int{},
		inputMode:    false,
		inputText:    "",
		editingIndex: -1,
		newSectionLevel: 0,
		hMode:       false,
	}
	
	m.updateVisibleItems()
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) updateVisibleItems() {
	m.visibleItems = []int{}
	var sectionStack []*Item // Stack to track nested sections
	
	for i, item := range m.items {
		if item.Type == TypeSection {
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
			
		} else if item.Type == TypeTask {
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

func (m Model) getCurrentItemIndex() int {
	if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
		return m.visibleItems[m.cursor]
	}
	return -1
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle input mode separately
		if m.inputMode {
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
		
		// Handle h-mode (waiting for number after 'h')
		if m.hMode {
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
		
		// Normal navigation mode
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
				if m.cursor < len(m.visibleItems)-1 {
					m.cursor++
				}
			}
		case "alt+k", "alt+up":
			itemIndex := m.getCurrentItemIndex()
			if itemIndex > 0 {
				m.items[itemIndex], m.items[itemIndex-1] = m.items[itemIndex-1], m.items[itemIndex]
				m.updateVisibleItems()
				if m.cursor > 0 {
					m.cursor--
				}
			}
		case "s":
			err := m.saveToFile()
			if err != nil {
				return m, nil
			}
		}
	}
	return m, nil
}

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
			_, err := writer.WriteString(fmt.Sprintf("%s %s\n", strings.Repeat("#", item.Level), item.Content))
			if err != nil {
				return err
			}
		case TypeTask:
			checkbox := "[ ]"
			if item.Checked != nil && *item.Checked {
				checkbox = "[x]"
			}
			indent := strings.Repeat(" ", item.Level)
			_, err := writer.WriteString(fmt.Sprintf("%s- %s %s\n", indent, checkbox, item.Content))
			if err != nil {
				return err
			}
		}
	}

	return nil
}


func (m Model) View() string {
	var s strings.Builder
	
	// Title
	title := titleStyle.Render(fmt.Sprintf("📋 Tasks - %s", m.filename))
	s.WriteString(title + "\n\n")
	
	if len(m.items) == 0 {
		noTasksMsg := taskPendingStyle.Render("No tasks found. Press 'q' to quit.")
		s.WriteString(noTasksMsg + "\n")
		return s.String()
	}

	var sectionStack []*Item // Stack to track sections for indentation
	
	for visIdx, itemIndex := range m.visibleItems {
		item := m.items[itemIndex]
		
		// Update section stack by finding all sections up to this item
		sectionStack = []*Item{}
		for i := 0; i < itemIndex; i++ {
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
			
			// Highlight current section
			if m.cursor == visIdx {
				// Calculate fixed width accounting for indentation
				indentWidth := len(indent) + 2 // indent + "  "
				highlightWidth := 70 - indentWidth
				highlightStyle := selectedStyle.Copy().Width(highlightWidth)
				
				var styledContent string
				if item.Collapsed {
					styledContent = highlightStyle.Render(arrowCollapsedStyle.Render("▶") + " " + sectionCollapsedStyle.Render(item.Content))
				} else {
					styledContent = highlightStyle.Render(arrowExpandedStyle.Render("▼") + " " + sectionStyle.Render(item.Content))
				}
				s.WriteString(fmt.Sprintf("%s  %s\n", indent, styledContent))
			} else {
				s.WriteString(fmt.Sprintf("%s  %s\n", indent, sectionLine))
			}
			
		case TypeTask:
			// Style checkbox and task text based on completion status
			var checkbox, taskText string
			if item.Checked != nil && *item.Checked {
				checkbox = checkedBoxStyle.Render("✓")
				taskText = taskCompletedStyle.Render(item.Content)
			} else {
				checkbox = uncheckedBoxStyle.Render("○")
				taskText = taskPendingStyle.Render(item.Content)
			}
			
			// Task indentation is based on the deepest section level + 1
			taskIndent := ""
			if len(sectionStack) > 0 {
				deepestLevel := sectionStack[len(sectionStack)-1].Level
				taskIndent = strings.Repeat("  ", deepestLevel)
			}
			
			taskLine := fmt.Sprintf("%s %s", checkbox, taskText)
			
			// Style the current task differently
			if m.cursor == visIdx {
				// Calculate fixed width accounting for indentation
				indentWidth := len(taskIndent) + 2 // taskIndent + "  "
				highlightWidth := 70 - indentWidth
				highlightStyle := selectedStyle.Copy().Width(highlightWidth)
				
				var styledContent string
				if item.Checked != nil && *item.Checked {
					styledContent = highlightStyle.Render(checkedBoxStyle.Render("✓") + " " + taskCompletedStyle.Render(item.Content))
				} else {
					styledContent = highlightStyle.Render(uncheckedBoxStyle.Render("○") + " " + taskPendingStyle.Render(item.Content))
				}
				s.WriteString(fmt.Sprintf("%s  %s\n", taskIndent, styledContent))
			} else {
				s.WriteString(fmt.Sprintf("%s  %s\n", taskIndent, taskLine))
			}
		}
	}

	// Show input field if in input mode
	if m.inputMode {
		s.WriteString("\n")
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
		s.WriteString(prompt + " " + inputField + "\n")
		s.WriteString(helpStyle.Render("Press Enter to save, Esc to cancel") + "\n")
	} else {
		// Help text with better styling
		var helpText string
		if m.hMode {
			helpText = helpStyle.Render("\nPress 1-6 to create section level (h1-h6), any other key to cancel")
		} else {
			helpText = helpStyle.Render("\n" +
				"Controls: " +
				"j/k (navigate) • " +
				"space (toggle) • " +
				"enter/←/→ (collapse/expand) • " +
				"n (new task) • " +
				"h1-h6 (new section) • " +
				"e (edit) • " +
				"alt+j/k (move) • " +
				"s (save) • " +
				"q (quit)")
		}
		
		s.WriteString(helpText + "\n")
	}
	
	return s.String()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: tasks <markdown-file>")
		os.Exit(1)
	}

	filename := os.Args[1]
	
	p := tea.NewProgram(initialModel(filename))
	
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
