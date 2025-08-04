package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime/debug"
	"slices"
	"strings"

	"golang.org/x/term"
)

// ItemType represents the type of item in the task list
type ItemType int

const (
	TypeSection ItemType = iota // Section header
	TypeTask                    // Task item
)

// Item represents a task or section in the markdown file
type Item struct {
	Type       ItemType          // Whether this is a section or task
	Level      int               // Heading level (1-6) for sections, or indentation for tasks
	Content    string            // The actual text content (clean description for tasks)
	Checked    *bool             // nil for sections, true/false for tasks
	Children   []Item            // Child items (for hierarchical structure)
	LineNumber int               // Line number in the original file (1-based)
	Metadata   map[string]string // Task metadata (nil for sections)
}

// parseItemID parses a string ID and converts it to 0-based index
func parseItemID(idStr string) (int, error) {
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		return -1, fmt.Errorf("invalid ID '%s'", idStr)
	}
	if id < 1 {
		return -1, fmt.Errorf("ID must be greater than 0")
	}
	return id - 1, nil // Convert to 0-based
}

// formatItem formats an item for display with optional terminal colors
func formatItem(item Item, index int) string {
	// 1-based indexing for user-facing IDs
	id := index + 1
	idStr := fmt.Sprintf("%-5d", id)

	var result string

	switch item.Type {
	case TypeSection:
		headerStr := strings.Repeat("#", item.Level) + " " + item.Content
		if isTerminal() {
			result = fmt.Sprintf("\033[33m%s\033[0m %s", idStr, headerStr)
		} else {
			result = fmt.Sprintf("%s %s", idStr, headerStr)
		}

	case TypeTask:
		checkBox := "[ ]"
		if item.Checked != nil && *item.Checked {
			checkBox = "[x]"
		}
		taskStr := "- " + checkBox + " " + item.Content
		if isTerminal() {
			result = fmt.Sprintf("\033[33m%s\033[0m %s", idStr, taskStr)
		} else {
			result = fmt.Sprintf("%s %s", idStr, taskStr)
		}

	default:
		panic(fmt.Errorf("invalid item type %v", item.Type))
	}

	return result
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

			// Use parseTask to extract metadata and clean description
			parsedTask := parseTask(line)
			if parsedTask.Description == "" && len(parsedTask.Metadata) == 0 {
				// parseTask failed, fall back to original parsing
				checked := matches[2] == "x"
				content := matches[3]
				items = append(items, Item{
					Type:       TypeTask,
					Level:      indentation,
					Content:    content,
					Checked:    &checked,
					LineNumber: lineNumber,
					Metadata:   nil,
				})
			} else {
				// Use parsed result
				items = append(items, Item{
					Type:       TypeTask,
					Level:      indentation,
					Content:    parsedTask.Description,
					Checked:    &parsedTask.Completed,
					LineNumber: lineNumber,
					Metadata:   parsedTask.Metadata,
				})
			}
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
		return slices.Delete(items, index, deleteEnd)
	} else {
		// For tasks, just remove the single item
		return slices.Delete(items, index, index+1)
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

// version is set at build time using -ldflags
var version string

// getVersion returns version information from build info
func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	if version != "" {
		return version
	}

	version = info.Main.Version
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

// isTerminal checks if stdout is connected to a terminal
func isTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Define global flags
	var showVersion bool
	var showHelp bool
	var filePath string

	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&showVersion, "v", false, "Show version information")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.BoolVar(&showHelp, "h", false, "Show help message")
	flag.StringVar(&filePath, "file", "TODO.md", "Path to the markdown file")

	// Custom usage function to avoid default flag help
	flag.Usage = func() {
		printUsage()
	}

	flag.Parse()

	// Handle version flag
	if showVersion {
		fmt.Printf("tasks version %s\n", getVersion())
		return nil
	}

	// Handle help flag
	if showHelp {
		printUsage()
		return nil
	}

	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		return fmt.Errorf("no command provided")
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "ls":
		return handleList(filePath)
	case "add":
		return handleAdd(filePath, cmdArgs)
	case "done":
		return handleDone(filePath, cmdArgs)
	case "undo":
		return handleUndo(filePath, cmdArgs)
	case "rm":
		return handleRemove(filePath, cmdArgs)
	case "edit":
		return handleEdit(filePath, cmdArgs)
	case "search":
		return handleSearch(filePath, cmdArgs)
	default:
		printUsage()
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func printUsage() {
	fmt.Println("Usage: tasks [--file <path>] <command> [args]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  ls                    List all tasks and sections with line numbers")
	fmt.Println("  add [flags] <text>    Add a new task or section")
	fmt.Println("  done <id>             Mark task as completed")
	fmt.Println("  undo <id>             Mark task as incomplete")
	fmt.Println("  rm <id>               Remove task or section (with confirmation)")
	fmt.Println("  edit <id>             Edit task or section in $EDITOR")
	fmt.Println("  search <term> [...]   Search tasks and sections with fuzzy matching")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --file <path>         Specify markdown file (default: TODO.md)")
	fmt.Println("  --help, -h            Show this help message")
	fmt.Println("  --version, -v         Show version information")
	fmt.Println("")
	fmt.Println("Use 'tasks add --help' for detailed add command options.")
}

func handleList(filePath string) error {
	items, err := parseMarkdownFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	for i, item := range items {
		fmt.Println(formatItem(item, i))
	}
	return nil
}

func handleAdd(filePath string, args []string) error {
	// Create a new flag set for the add command
	addFlags := flag.NewFlagSet("add", flag.ContinueOnError)
	isSection := addFlags.Bool("section", false, "Add a section instead of a task")
	sectionLevel := addFlags.Int("level", 1, "Section level (1-6) when adding a section")
	afterID := addFlags.Int("after", 0, "Add after the specified item ID (1-based)")
	showHelp := addFlags.Bool("help", false, "Show help for the add command")

	// Custom usage function
	addFlags.Usage = func() {
		fmt.Println("Usage: tasks add [flags] <text>")
		fmt.Println("")
		fmt.Println("Add a new task or section to the markdown file.")
		fmt.Println("")
		fmt.Println("Flags:")
		fmt.Println("  --section         Add a section instead of a task")
		fmt.Println("  --level <n>       Section level (1-6), only used with --section (default: 1)")
		fmt.Println("  --after <id>      Add after the specified item ID (1-based)")
		fmt.Println("  --help            Show this help message")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  tasks add \"Review documentation\"")
		fmt.Println("  tasks add --after 3 \"Follow-up task\"")
		fmt.Println("  tasks add --section --level 2 \"New Project Phase\"")
		fmt.Println("  tasks add --section \"Main Section\"")
	}

	// Parse the flags
	if err := addFlags.Parse(args); err != nil {
		return err
	}

	// Check if help was requested
	if *showHelp {
		addFlags.Usage()
		return nil
	}

	// Get remaining arguments (the content)
	content := strings.Join(addFlags.Args(), " ")
	if content == "" {
		return fmt.Errorf("content is required")
	}

	// Validate section level
	if *isSection && (*sectionLevel < 1 || *sectionLevel > 6) {
		return fmt.Errorf("invalid section level %d (must be 1-6)", *sectionLevel)
	}

	// Create TaskManager and load items
	tm := &TaskManager{FilePath: filePath}
	if err := tm.Load(); err != nil {
		return fmt.Errorf("loading file: %w", err)
	}

	// Convert afterID to 0-based index (-1 means append at end)
	afterIndex := -1
	if *afterID > 0 {
		if *afterID > len(tm.Items) {
			return fmt.Errorf("item ID %d does not exist (max: %d)", *afterID, len(tm.Items))
		}
		afterIndex = *afterID - 1 // Convert to 0-based
	}

	if *isSection {
		// Add a section
		if err := tm.AddSection(content, *sectionLevel, afterIndex); err != nil {
			return err
		}

		if *afterID > 0 {
			fmt.Printf("Added section after item %d: %s %s\n", *afterID, strings.Repeat("#", *sectionLevel), content)
		} else {
			fmt.Printf("Added section: %s %s\n", strings.Repeat("#", *sectionLevel), content)
		}
	} else {
		// Add a task
		if err := tm.AddTask(content, afterIndex); err != nil {
			return err
		}

		if *afterID > 0 {
			fmt.Printf("Added task after item %d: %s\n", *afterID, content)
		} else {
			fmt.Printf("Added task: %s\n", content)
		}
	}

	// Save the changes
	if err := tm.Save(); err != nil {
		return fmt.Errorf("saving file: %w", err)
	}
	return nil
}

func handleDone(filePath string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("done command requires an item ID")
	}

	// Parse the ID
	index, err := parseItemID(args[0])
	if err != nil {
		return err
	}
	id := index + 1 // Keep original ID for display

	// Use withTaskManager to handle load-operation-save pattern
	err = withTaskManager(filePath, func(tm *TaskManager) error {
		return tm.ToggleTask(index, true)
	})

	if err != nil {
		return err
	}

	fmt.Printf("Marked task %d as completed\n", id)
	return nil
}

// confirmRemoval prompts the user for confirmation before removing an item
func confirmRemoval(itemDesc string) (bool, error) {
	fmt.Printf("Remove %s? [y/N] ", itemDesc)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes", nil
}

func handleUndo(filePath string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("undo command requires an item ID")
	}

	// Parse the ID
	index, err := parseItemID(args[0])
	if err != nil {
		return err
	}
	id := index + 1 // Keep original ID for display

	// Use withTaskManager to handle load-operation-save pattern
	err = withTaskManager(filePath, func(tm *TaskManager) error {
		return tm.ToggleTask(index, false)
	})

	if err != nil {
		return err
	}

	fmt.Printf("Marked task %d as incomplete\n", id)
	return nil
}

func handleRemove(filePath string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("rm command requires an item ID")
	}

	// Parse the ID
	index, err := parseItemID(args[0])
	if err != nil {
		return err
	}
	id := index + 1 // Keep original ID for display

	// Check for force flag
	force := false
	if len(args) > 1 && args[1] == "--force" {
		force = true
	}

	// Variables to capture item details before removal
	var itemContent string
	var itemType string

	// Use withTaskManager to handle load-operation-save pattern
	err = withTaskManager(filePath, func(tm *TaskManager) error {
		// Get the item before removing it (for confirmation message)
		item, err := tm.GetItem(index)
		if err != nil {
			return err
		}

		// Store the item details before removal to avoid pointer issues
		itemContent = item.Content
		itemType = "task"
		if item.Type == TypeSection {
			itemType = "section"
		}

		// Check if confirmation is needed
		if !force {
			itemDesc := fmt.Sprintf("%s %d: %s", itemType, id, itemContent)
			if item.Type == TypeSection {
				// Count children that will be removed
				childCount := 0
				for i := index + 1; i < len(tm.Items); i++ {
					nextItem := tm.Items[i]
					if nextItem.Type == TypeSection && nextItem.Level <= item.Level {
						break
					}
					childCount++
				}
				if childCount > 0 {
					itemDesc = fmt.Sprintf("%s %d: %s (and %d child items)", itemType, id, itemContent, childCount)
				}
			}

			confirmed, err := confirmRemoval(itemDesc)
			if err != nil {
				return fmt.Errorf("confirmation failed: %w", err)
			}
			if !confirmed {
				return fmt.Errorf("removal cancelled")
			}
		}

		// Remove the item
		return tm.RemoveItem(index)
	})

	if err != nil {
		return err
	}

	fmt.Printf("Removed %s %d: %s\n", itemType, id, itemContent)
	return nil
}

func handleEdit(filePath string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("edit command requires an item ID")
	}

	// Parse the ID
	index, err := parseItemID(args[0])
	if err != nil {
		return err
	}
	id := index + 1 // Keep original ID for display

	// Load TaskManager to get the line number
	tm, err := createAndLoadTaskManager(filePath)
	if err != nil {
		return fmt.Errorf("loading file: %w", err)
	}

	// Get the item to find its line number
	item, err := tm.GetItem(index)
	if err != nil {
		return err
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
		return fmt.Errorf("running editor: %w", err)
	}

	fmt.Printf("Edited item %d with %s\n", id, editor)
	return nil
}

func handleSearch(filePath string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("search command requires at least one search term")
	}

	// Load items from file
	items, err := parseMarkdownFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	// Perform search
	results := searchItems(items, args)

	if len(results) == 0 {
		fmt.Printf("No matches found for: %s\n", strings.Join(args, " "))
		return nil
	}

	// Display results
	fmt.Printf("Found %d match(es) for: %s\n", len(results), strings.Join(args, " "))
	fmt.Println()

	for _, result := range results {
		fmt.Println(formatItem(result.Item, result.Index))
	}
	return nil
}
