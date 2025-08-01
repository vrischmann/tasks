package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
)

// ItemType represents the type of item in the task list
type ItemType int

const (
	TypeSection ItemType = iota // Section header
	TypeTask                    // Task item
)

// Item represents a task or section in the markdown file
type Item struct {
	Type       ItemType // Whether this is a section or task
	Level      int      // Heading level (1-6) for sections, or indentation for tasks
	Content    string   // The actual text content
	Checked    *bool    // nil for sections, true/false for tasks
	Children   []Item   // Child items (for hierarchical structure)
	LineNumber int      // Line number in the original file (1-based)
}

// TaskManager handles loading, modifying, and saving markdown files
type TaskManager struct {
	FilePath string
	Items    []Item
}

// Load reads and parses the markdown file
func (tm *TaskManager) Load() error {
	items, err := parseMarkdownFile(tm.FilePath)
	if err != nil {
		return err
	}
	tm.Items = items
	return nil
}

// Save writes the current items back to the file
func (tm *TaskManager) Save() error {
	return saveToFile(tm.FilePath, tm.Items)
}

// GetItem returns the item at the specified index (0-based)
func (tm *TaskManager) GetItem(index int) (*Item, error) {
	if index < 0 || index >= len(tm.Items) {
		return nil, fmt.Errorf("invalid item index: %d", index)
	}
	return &tm.Items[index], nil
}

// ToggleTask marks a task as completed or incomplete
func (tm *TaskManager) ToggleTask(index int, completed bool) error {
	item, err := tm.GetItem(index)
	if err != nil {
		return err
	}

	if item.Type != TypeTask {
		return fmt.Errorf("item at index %d is not a task", index)
	}

	*item.Checked = completed
	return nil
}

// RemoveItem removes an item and its children from the list
func (tm *TaskManager) RemoveItem(index int) error {
	if index < 0 || index >= len(tm.Items) {
		return fmt.Errorf("invalid item index: %d", index)
	}

	tm.Items = deleteItem(tm.Items, index)
	return nil
}

// AddTask adds a new task to the list
func (tm *TaskManager) AddTask(content string, afterIndex int) error {
	newTask := Item{
		Type:       TypeTask,
		Level:      0, // Default to no indentation
		Content:    content,
		Checked:    func() *bool { b := false; return &b }(),
		LineNumber: 0, // Will be set to proper value when saved
	}

	if afterIndex == -1 {
		// Add at the end
		tm.Items = append(tm.Items, newTask)
	} else {
		// Insert after the specified index
		if afterIndex < 0 || afterIndex >= len(tm.Items) {
			return fmt.Errorf("invalid after index: %d", afterIndex)
		}

		// Insert at afterIndex + 1
		insertPos := afterIndex + 1
		tm.Items = append(tm.Items[:insertPos], append([]Item{newTask}, tm.Items[insertPos:]...)...)
	}

	return nil
}

// AddSection adds a new section to the list
func (tm *TaskManager) AddSection(content string, level int, afterIndex int) error {
	if level < 1 || level > 6 {
		return fmt.Errorf("invalid section level: %d (must be 1-6)", level)
	}

	newSection := Item{
		Type:       TypeSection,
		Level:      level,
		Content:    content,
		Checked:    nil,
		LineNumber: 0, // Will be set to proper value when saved
	}

	if afterIndex == -1 {
		// Add at the end
		tm.Items = append(tm.Items, newSection)
	} else {
		// Insert after the specified index
		if afterIndex < 0 || afterIndex >= len(tm.Items) {
			return fmt.Errorf("invalid after index: %d", afterIndex)
		}

		// Insert at afterIndex + 1
		insertPos := afterIndex + 1
		tm.Items = append(tm.Items[:insertPos], append([]Item{newSection}, tm.Items[insertPos:]...)...)
	}

	return nil
}

// parseMarkdownFile reads a markdown file and extracts tasks and sections
func parseMarkdownFile(filePath string) ([]Item, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var items []Item
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	// Regex patterns for parsing
	sectionRegex := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	taskRegex := regexp.MustCompile(`^(\s*)-\s+\[([x\s])\]\s+(.+)$`)

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimRight(scanner.Text(), " \t")

		if line == "" {
			continue
		}

		// Check if it's a section header
		if matches := sectionRegex.FindStringSubmatch(line); matches != nil {
			level := len(matches[1])
			content := matches[2]
			items = append(items, Item{
				Type:       TypeSection,
				Level:      level,
				Content:    content,
				Checked:    nil,
				LineNumber: lineNumber,
			})
			continue
		}

		// Check if it's a task item
		if matches := taskRegex.FindStringSubmatch(line); matches != nil {
			indentation := len(matches[1])
			checked := matches[2] == "x"
			content := matches[3]
			items = append(items, Item{
				Type:       TypeTask,
				Level:      indentation,
				Content:    content,
				Checked:    &checked,
				LineNumber: lineNumber,
			})
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return items, nil
}

// deleteItem removes an item and all its children from the slice
func deleteItem(items []Item, index int) []Item {
	if index < 0 || index >= len(items) {
		return items
	}

	currentItem := items[index]

	// If it's a section, we need to remove all child items too
	if currentItem.Type == TypeSection {
		// Find the range of items to delete (current item + all children)
		deleteEnd := index + 1
		for deleteEnd < len(items) {
			nextItem := items[deleteEnd]
			// Stop when we find an item at the same or higher level (lower number)
			if nextItem.Type == TypeSection && nextItem.Level <= currentItem.Level {
				break
			}
			deleteEnd++
		}
		// Remove the range of items
		return append(items[:index], items[deleteEnd:]...)
	} else {
		// For tasks, just remove the single item
		return append(items[:index], items[index+1:]...)
	}
}

// fuzzyMatch performs case-insensitive fuzzy matching
// Returns a score between 0 and 1, where 1 is a perfect match
func fuzzyMatch(pattern, text string) float64 {
	pattern = strings.ToLower(pattern)
	text = strings.ToLower(text)
	
	// Handle empty strings
	if len(pattern) == 0 && len(text) == 0 {
		return 1.0
	}
	if len(pattern) == 0 || len(text) == 0 {
		return 0.0
	}
	
	if pattern == text {
		return 1.0
	}
	
	if strings.Contains(text, pattern) {
		// Exact substring match gets high score
		return 0.8
	}
	
	// Character-by-character fuzzy matching
	patternIdx := 0
	matches := 0
	
	for _, char := range text {
		if patternIdx < len(pattern) && char == rune(pattern[patternIdx]) {
			matches++
			patternIdx++
		}
	}
	
	// Must match all characters in the pattern to be considered a match
	if matches < len(pattern) {
		return 0.0
	}
	
	// Score based on how tightly the characters are packed together
	// and the length ratio between pattern and text
	charMatchRatio := float64(matches) / float64(len(pattern))
	lengthPenalty := float64(len(pattern)) / float64(len(text))
	
	// Calculate base score
	score := charMatchRatio * lengthPenalty * 0.6
	
	// Add a small bonus to distinguish exact character order matches
	if score > 0 {
		score = score + 0.05
	}
	
	// Only return a meaningful score if we have a decent match ratio
	if score < 0.3 {
		return 0.0
	}
	
	return score
}

// SearchResult represents a search match with score
type SearchResult struct {
	Item  Item
	Index int
	Score float64
}

// searchItems performs fuzzy search across all items
func searchItems(items []Item, queries []string) []SearchResult {
	var results []SearchResult
	
	// Return empty if no queries provided
	if len(queries) == 0 {
		return results
	}
	
	for i, item := range items {
		totalScore := 0.0
		matchCount := 0
		
		// Combine all query terms into one search pattern
		searchPattern := strings.Join(queries, " ")
		
		// Search in item content
		contentScore := fuzzyMatch(searchPattern, item.Content)
		if contentScore > 0 {
			totalScore += contentScore
			matchCount++
		}
		
		// Also try matching individual query terms
		for _, query := range queries {
			if strings.TrimSpace(query) == "" {
				continue
			}
			queryScore := fuzzyMatch(query, item.Content)
			if queryScore > contentScore {
				totalScore = queryScore
				matchCount = 1
				break
			}
		}
		
		// Only include results with a minimum score
		if totalScore > 0.3 {
			avgScore := totalScore / float64(matchCount)
			results = append(results, SearchResult{
				Item:  item,
				Index: i,
				Score: avgScore,
			})
		}
	}
	
	// Sort results by score (highest first)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Score < results[j].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
	
	return results
}

// saveToFile writes the items back to the markdown file
func saveToFile(filePath string, items []Item) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	for _, item := range items {
		var line string
		if item.Type == TypeSection {
			// Format section header
			line = strings.Repeat("#", item.Level) + " " + item.Content
		} else {
			// Format task item
			indentation := strings.Repeat(" ", item.Level)
			checkBox := "[ ]"
			if item.Checked != nil && *item.Checked {
				checkBox = "[x]"
			}
			line = indentation + "- " + checkBox + " " + item.Content
		}

		if _, err := fmt.Fprintln(file, line); err != nil {
			return fmt.Errorf("failed to write line: %w", err)
		}
	}

	return nil
}

// getVersion returns version information from build info
func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	version := info.Main.Version
	if version == "(devel)" || version == "" {
		// Try to get commit hash from build settings
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				if len(setting.Value) >= 7 {
					return setting.Value[:7]
				}
				return setting.Value
			}
		}
		return "dev"
	}

	return version
}

func main() {
	// Handle version flag early
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("tasks version %s\n", getVersion())
		return
	}

	// Define global file flag
	filePath := flag.String("file", "TODO.md", "Path to the markdown file")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "ls":
		handleList(*filePath)
	case "add":
		handleAdd(*filePath, cmdArgs)
	case "done":
		handleDone(*filePath, cmdArgs)
	case "undo":
		handleUndo(*filePath, cmdArgs)
	case "rm":
		handleRemove(*filePath, cmdArgs)
	case "edit":
		handleEdit(*filePath, cmdArgs)
	case "search":
		handleSearch(*filePath, cmdArgs)
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: tasks [--file <path>] <command> [args]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  ls                    List all tasks and sections with line numbers")
	fmt.Println("  add <text>            Add a new task")
	fmt.Println("  add --section <n> <text>  Add a new section at level n (1-6)")
	fmt.Println("  done <id>             Mark task as completed")
	fmt.Println("  undo <id>             Mark task as incomplete")
	fmt.Println("  rm <id>               Remove task or section")
	fmt.Println("  edit <id>             Edit task or section in $EDITOR")
	fmt.Println("  search <term> [...]   Search tasks and sections with fuzzy matching")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --file <path>         Specify markdown file (default: TODO.md)")
	fmt.Println("  --version, -v         Show version information")
}

func handleList(filePath string) {
	items, err := parseMarkdownFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	for i, item := range items {
		// 1-based indexing for user-facing IDs
		id := i + 1

		if item.Type == TypeSection {
			// Format section header with appropriate indentation
			indent := ""
			if item.Level > 1 {
				indent = strings.Repeat("  ", item.Level-1)
			}
			fmt.Printf("%d   %s%s %s\n", id, indent, strings.Repeat("#", item.Level), item.Content)
		} else {
			// Format task item
			indent := strings.Repeat("  ", item.Level/2) // Convert spaces to visual indentation
			checkBox := "[ ]"
			if item.Checked != nil && *item.Checked {
				checkBox = "[x]"
			}
			fmt.Printf("%d   %s- %s %s\n", id, indent, checkBox, item.Content)
		}
	}
}

func handleAdd(filePath string, args []string) {
	if len(args) < 1 {
		fmt.Println("Error: add command requires content")
		fmt.Println("Usage: tasks add <text>")
		fmt.Println("       tasks add --section <level> <text>")
		os.Exit(1)
	}

	// Create TaskManager and load items
	tm := &TaskManager{FilePath: filePath}
	if err := tm.Load(); err != nil {
		fmt.Printf("Error loading file: %v\n", err)
		os.Exit(1)
	}

	// Check if this is a section creation
	if args[0] == "--section" {
		if len(args) < 3 {
			fmt.Println("Error: --section requires level and text")
			fmt.Println("Usage: tasks add --section <level> <text>")
			os.Exit(1)
		}

		level, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Printf("Error: invalid section level '%s'\n", args[1])
			os.Exit(1)
		}

		content := strings.Join(args[2:], " ")

		if err := tm.AddSection(content, level, -1); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Added section: %s %s\n", strings.Repeat("#", level), content)
	} else {
		// Regular task addition
		content := strings.Join(args, " ")

		if err := tm.AddTask(content, -1); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Added task: %s\n", content)
	}

	// Save the changes
	if err := tm.Save(); err != nil {
		fmt.Printf("Error saving file: %v\n", err)
		os.Exit(1)
	}
}

func handleDone(filePath string, args []string) {
	if len(args) < 1 {
		fmt.Println("Error: done command requires an item ID")
		fmt.Println("Usage: tasks done <id>")
		os.Exit(1)
	}

	// Parse the ID (convert from 1-based to 0-based)
	var id int
	if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
		fmt.Printf("Error: invalid ID '%s'\n", args[0])
		os.Exit(1)
	}
	index := id - 1 // Convert to 0-based

	// Create TaskManager and load items
	tm := &TaskManager{FilePath: filePath}
	if err := tm.Load(); err != nil {
		fmt.Printf("Error loading file: %v\n", err)
		os.Exit(1)
	}

	// Toggle the task to completed
	if err := tm.ToggleTask(index, true); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Save the changes
	if err := tm.Save(); err != nil {
		fmt.Printf("Error saving file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Marked task %d as completed\n", id)
}

func handleUndo(filePath string, args []string) {
	if len(args) < 1 {
		fmt.Println("Error: undo command requires an item ID")
		fmt.Println("Usage: tasks undo <id>")
		os.Exit(1)
	}

	// Parse the ID (convert from 1-based to 0-based)
	var id int
	if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
		fmt.Printf("Error: invalid ID '%s'\n", args[0])
		os.Exit(1)
	}
	index := id - 1 // Convert to 0-based

	// Create TaskManager and load items
	tm := &TaskManager{FilePath: filePath}
	if err := tm.Load(); err != nil {
		fmt.Printf("Error loading file: %v\n", err)
		os.Exit(1)
	}

	// Toggle the task to incomplete
	if err := tm.ToggleTask(index, false); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Save the changes
	if err := tm.Save(); err != nil {
		fmt.Printf("Error saving file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Marked task %d as incomplete\n", id)
}

func handleRemove(filePath string, args []string) {
	if len(args) < 1 {
		fmt.Println("Error: rm command requires an item ID")
		fmt.Println("Usage: tasks rm <id>")
		os.Exit(1)
	}

	// Parse the ID (convert from 1-based to 0-based)
	var id int
	if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
		fmt.Printf("Error: invalid ID '%s'\n", args[0])
		os.Exit(1)
	}
	index := id - 1 // Convert to 0-based

	// Create TaskManager and load items
	tm := &TaskManager{FilePath: filePath}
	if err := tm.Load(); err != nil {
		fmt.Printf("Error loading file: %v\n", err)
		os.Exit(1)
	}

	// Get the item before removing it (for confirmation message)
	item, err := tm.GetItem(index)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	itemType := "task"
	if item.Type == TypeSection {
		itemType = "section"
	}

	// Remove the item
	if err := tm.RemoveItem(index); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Save the changes
	if err := tm.Save(); err != nil {
		fmt.Printf("Error saving file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Removed %s %d: %s\n", itemType, id, item.Content)
}

func handleEdit(filePath string, args []string) {
	if len(args) < 1 {
		fmt.Println("Error: edit command requires an item ID")
		fmt.Println("Usage: tasks edit <id>")
		os.Exit(1)
	}

	// Parse the ID (convert from 1-based to 0-based)
	var id int
	if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
		fmt.Printf("Error: invalid ID '%s'\n", args[0])
		os.Exit(1)
	}
	index := id - 1 // Convert to 0-based

	// Load items to get the actual line number
	tm := &TaskManager{FilePath: filePath}
	if err := tm.Load(); err != nil {
		fmt.Printf("Error loading file: %v\n", err)
		os.Exit(1)
	}

	// Get the item to find its line number
	item, err := tm.GetItem(index)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	lineNumber := item.LineNumber

	// Get editor from environment, default to vi
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	// Construct the command to open the file at the specific line
	var cmd *exec.Cmd

	// Different editors have different syntax for opening at a specific line
	switch {
	case strings.Contains(editor, "vim") || strings.Contains(editor, "vi"):
		cmd = exec.Command(editor, fmt.Sprintf("+%d", lineNumber), filePath)
	case strings.Contains(editor, "nano"):
		cmd = exec.Command(editor, fmt.Sprintf("+%d", lineNumber), filePath)
	case strings.Contains(editor, "emacs"):
		cmd = exec.Command(editor, fmt.Sprintf("+%d", lineNumber), filePath)
	case strings.Contains(editor, "code"): // VS Code
		cmd = exec.Command(editor, "--goto", fmt.Sprintf("%s:%d", filePath, lineNumber))
	default:
		// Fall back to just opening the file
		cmd = exec.Command(editor, filePath)
	}

	// Inherit stdin, stdout, and stderr so the editor works properly
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the editor
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running editor: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Edited item %d with %s\n", id, editor)
}

func handleSearch(filePath string, args []string) {
	if len(args) < 1 {
		fmt.Println("Error: search command requires at least one search term")
		fmt.Println("Usage: tasks search <term1> [term2] [...]")
		os.Exit(1)
	}

	// Load items from file
	items, err := parseMarkdownFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Perform search
	results := searchItems(items, args)

	if len(results) == 0 {
		fmt.Printf("No matches found for: %s\n", strings.Join(args, " "))
		return
	}

	// Display results
	fmt.Printf("Found %d match(es) for: %s\n", len(results), strings.Join(args, " "))
	fmt.Println()

	for _, result := range results {
		// 1-based indexing for user-facing IDs
		id := result.Index + 1
		item := result.Item

		if item.Type == TypeSection {
			// Format section header with appropriate indentation
			indent := ""
			if item.Level > 1 {
				indent = strings.Repeat("  ", item.Level-1)
			}
			fmt.Printf("%d   %s%s %s\n", id, indent, strings.Repeat("#", item.Level), item.Content)
		} else {
			// Format task item
			indent := strings.Repeat("  ", item.Level/2) // Convert spaces to visual indentation
			checkBox := "[ ]"
			if item.Checked != nil && *item.Checked {
				checkBox = "[x]"
			}
			fmt.Printf("%d   %s- %s %s\n", id, indent, checkBox, item.Content)
		}
	}
}
