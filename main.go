package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	colorMode string
	filePath  string
)

// shouldUseColor checks if color output should be used
func shouldUseColor() bool {
	switch colorMode {
	case "always":
		return true
	case "never":
		return false
	default:
		return term.IsTerminal(int(os.Stdout.Fd()))
	}
}

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
	idStr := fmt.Sprintf("% -5d", id)

	var result string

	switch item.Type {
	case TypeSection:
		if id > 1 {
			fmt.Println()
		}

		headerStr := strings.Repeat("#", item.Level) + " " + item.Content
		if shouldUseColor() {
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

		// Add metadata if it exists
		if len(item.Metadata) > 0 {
			var metadataParts []string
			// Sort keys for consistent order
			keys := make([]string, 0, len(item.Metadata))
			for k := range item.Metadata {
				keys = append(keys, k)
			}
			slices.Sort(keys)

			for _, k := range keys {
				v := item.Metadata[k]
				if shouldUseColor() {
					// Green color for metadata
					metadataParts = append(metadataParts, fmt.Sprintf("\033[32m%s:%s\033[0m", k, v))
				} else {
					metadataParts = append(metadataParts, fmt.Sprintf("%s:%s", k, v))
				}
			}
			taskStr += " " + strings.Join(metadataParts, " ")
		}

		if shouldUseColor() {
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
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file '%s' does not exist", filePath)
		}
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

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	rootCmd := &cobra.Command{
		Use:   "tasks",
		Short: "A stateless, composable CLI tool for managing markdown task lists",
		Long: `A stateless, composable CLI tool for managing markdown task lists.
It provides Unix-friendly commands for manipulating tasks and sections stored in markdown files,
designed for scripting and integration with other tools like fzf and shell workflows.`,
		Version: getVersion(),
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&filePath, "file", "TODO.md", "Path to the markdown file")
	rootCmd.PersistentFlags().StringVar(&colorMode, "color", "auto", "When to use color output (always, never, auto)")

	// Add subcommands
	rootCmd.AddCommand(
		newListCommand(),
		newAddCommand(),
		newDoneCommand(),
		newUndoCommand(),
		newRemoveCommand(),
		newEditCommand(),
		newSearchCommand(),
	)

	return rootCmd.Execute()
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List all tasks and sections with line numbers",
		Long:  "List all tasks and sections in the markdown file with 1-based indexing for easy reference.",
		RunE: func(cmd *cobra.Command, args []string) error {
			items, err := parseMarkdownFile(filePath)
			if err != nil {
				return err
			}

			for i, item := range items {
				fmt.Println(formatItem(item, i))
			}
			return nil
		},
	}
}

func newAddCommand() *cobra.Command {
	var (
		isSection    bool
		sectionLevel int
		afterID      int
	)

	cmd := &cobra.Command{
		Use:   "add [text]",
		Short: "Add a new task or section",
		Long:  "Add a new task or section to the markdown file.",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			content := strings.Join(args, " ")
			if content == "" {
				return fmt.Errorf("content is required")
			}

			// Validate section level
			if isSection && (sectionLevel < 1 || sectionLevel > 6) {
				return fmt.Errorf("invalid section level %d (must be 1-6)", sectionLevel)
			}

			// Create TaskManager and load items
			tm, err := NewTaskManager(filePath)
			if err != nil {
				return fmt.Errorf("loading file: %w", err)
			}

			// Convert afterID to 0-based index (-1 means append at end)
			afterIndex := -1
			if afterID > 0 {
				if afterID > len(tm.Items) {
					return fmt.Errorf("item ID %d does not exist (max: %d)", afterID, len(tm.Items))
				}
				afterIndex = afterID - 1 // Convert to 0-based
			}

			if isSection {
				// Add a section
				if err := tm.AddSection(content, sectionLevel, afterIndex); err != nil {
					return err
				}

				if afterID > 0 {
					fmt.Printf("Added section after item %d: %s %s\n", afterID, strings.Repeat("#", sectionLevel), content)
				} else {
					fmt.Printf("Added section: %s %s\n", strings.Repeat("#", sectionLevel), content)
				}
			} else {
				// Add a task
				if err := tm.AddTask(content, afterIndex); err != nil {
					return err
				}

				if afterID > 0 {
					fmt.Printf("Added task after item %d: %s\n", afterID, content)
				} else {
					fmt.Printf("Added task: %s\n", content)
				}
			}

			// Save the changes
			if err := tm.Save(); err != nil {
				return fmt.Errorf("saving file: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&isSection, "section", "s", false, "Add a section instead of a task")
	cmd.Flags().IntVarP(&sectionLevel, "level", "l", 1, "Section level (1-6) when adding a section")
	cmd.Flags().IntVarP(&afterID, "after", "a", 0, "Add after the specified item ID (1-based)")

	return cmd
}

func newDoneCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "done <id>",
		Short: "Mark task as completed",
		Long:  "Mark a task as completed by specifying its ID.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse the ID
			index, err := parseItemID(args[0])
			if err != nil {
				return err
			}
			id := index + 1 // Keep original ID for display

			tm, err := NewTaskManager(filePath)
			if err != nil {
				return err
			}

			if err := tm.ToggleTask(index, true); err != nil {
				return err
			}

			if err := tm.Save(); err != nil {
				return fmt.Errorf("saving file: %w", err)
			}

			fmt.Printf("Marked task %d as completed\n", id)
			return nil
		},
	}
}

func newUndoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "undo <id>",
		Short: "Mark task as incomplete",
		Long:  "Mark a task as incomplete by specifying its ID.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse the ID
			index, err := parseItemID(args[0])
			if err != nil {
				return err
			}
			id := index + 1 // Keep original ID for display

			tm, err := NewTaskManager(filePath)
			if err != nil {
				return err
			}

			if err := tm.ToggleTask(index, false); err != nil {
				return err
			}

			if err := tm.Save(); err != nil {
				return fmt.Errorf("saving file: %w", err)
			}

			fmt.Printf("Marked task %d as incomplete\n", id)
			return nil
		},
	}
}

func newRemoveCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "rm <id>",
		Short: "Remove task or section",
		Long:  "Remove a task or section by specifying its ID. Sections will remove all child items.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse the ID
			index, err := parseItemID(args[0])
			if err != nil {
				return err
			}
			id := index + 1 // Keep original ID for display

			// Variables to capture item details before removal
			var itemContent string
			var itemType string

			tm, err := NewTaskManager(filePath)
			if err != nil {
				return err
			}

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
			if err := tm.RemoveItem(index); err != nil {
				return err
			}

			if err := tm.Save(); err != nil {
				return fmt.Errorf("saving file: %w", err)
			}

			fmt.Printf("Removed %s %d: %s\n", itemType, id, itemContent)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force removal without confirmation")

	return cmd
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

func newEditCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit task or section in $EDITOR",
		Long:  "Edit a task or section by opening the file in $EDITOR at the appropriate line.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse the ID
			index, err := parseItemID(args[0])
			if err != nil {
				return err
			}
			id := index + 1 // Keep original ID for display

			// Load TaskManager to get the line number
			tm, err := NewTaskManager(filePath)
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
			var execCmd *exec.Cmd

			// Different editors have different syntax for opening at a specific line
			switch {
			case strings.Contains(editor, "vim") || strings.Contains(editor, "vi"):
				execCmd = exec.Command(editor, fmt.Sprintf("+%d", lineNumber), filePath)
			case strings.Contains(editor, "nano"):
				execCmd = exec.Command(editor, fmt.Sprintf("+%d", lineNumber), filePath)
			case strings.Contains(editor, "emacs"):
				execCmd = exec.Command(editor, fmt.Sprintf("+%d", lineNumber), filePath)
			case strings.Contains(editor, "code"): // VS Code
				execCmd = exec.Command(editor, "--goto", fmt.Sprintf("%s:%d", filePath, lineNumber))
			default:
				// Fall back to just opening the file
				execCmd = exec.Command(editor, filePath)
			}

			// Inherit stdin, stdout, and stderr so the editor works properly
			execCmd.Stdin = os.Stdin
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr

			// Run the editor
			if err := execCmd.Run(); err != nil {
				return fmt.Errorf("running editor: %w", err)
			}

			fmt.Printf("Edited item %d with %s\n", id, editor)
			return nil
		},
	}
}

func newSearchCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "search [terms...]",
		Short: "Search tasks and sections",
		Long:  "Search tasks and sections with fuzzy matching. Multiple search terms can be provided.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load items from file
			items, err := parseMarkdownFile(filePath)
			if err != nil {
				return err
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
		},
	}
}
