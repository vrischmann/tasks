package main

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"slices"
	"strings"
)

// createEmptyFile creates an empty markdown file at the specified path.
// It ensures the parent directories exist and creates an empty file.
func createEmptyFile(filePath string) error {
	// Ensure parent directory exists
	if dir := strings.TrimSpace(strings.TrimSuffix(filePath, "/")); dir != "" {
		parent := ""
		if idx := strings.LastIndex(dir, "/"); idx != -1 {
			parent = dir[:idx]
		}
		if parent != "" {
			if err := os.MkdirAll(parent, 0o755); err != nil {
				return err
			}
		}
	}

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}

// TaskManager handles loading, modifying, and saving markdown files
type TaskManager struct {
	FilePath string
	Items    []Item
}

// Load reads and parses the markdown file
func (tm *TaskManager) Load() error {
	items, err := parseMarkdownFile(tm.FilePath)

	switch {
	case errors.Is(err, fs.ErrNotExist):
		// Create empty file with no items
		if err := createEmptyFile(tm.FilePath); err != nil {
			return fmt.Errorf("failed to create file '%s': %w", tm.FilePath, err)
		}

		// Return empty items
		tm.Items = []Item{}
		return nil

	case err != nil:
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
func (tm *TaskManager) AddTask(description string, metadata map[string]string, afterIndex int) error {
	newTask := Item{
		Type:       TypeTask,
		Level:      0, // Default to no indentation
		Content:    description,
		Checked:    func() *bool { b := false; return &b }(),
		LineNumber: 0, // Will be set to proper value when saved
		Metadata:   metadata,
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
		tm.Items = slices.Insert(tm.Items, insertPos, newTask)
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
		LineNumber: 0,   // Will be set to proper value when saved
		Metadata:   nil, // Sections don't have metadata
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
		tm.Items = slices.Insert(tm.Items, insertPos, newSection)
	}

	return nil
}

// NewTaskManager creates a new TaskManager and loads the file.
func NewTaskManager(filePath string) (*TaskManager, error) {
	tm := &TaskManager{FilePath: filePath}
	if err := tm.Load(); err != nil {
		return nil, fmt.Errorf("error loading file: %w", err)
	}
	return tm, nil
}

// saveToFile writes the items back to the markdown file
func saveToFile(filePath string, items []Item) error {
	var buf strings.Builder

	for i, item := range items {
		var line string

		switch item.Type {
		case TypeSection:
			// Add empty line before section header (except for first item)
			if i > 0 {
				buf.WriteString("\n")
			}

			// Format section header
			line = strings.Repeat("#", item.Level) + " " + item.Content
			buf.WriteString(line)
			buf.WriteString("\n")

			// Add empty line after section header (if not last item and next item is not a section)
			if i < len(items)-1 && items[i+1].Type != TypeSection {
				buf.WriteString("\n")
			}

		case TypeTask:
			// Format task item without any indentation
			checkBox := "[ ]"
			if item.Checked != nil && *item.Checked {
				checkBox = "[x]"
			}

			// Build the content with metadata
			content := item.Content
			if len(item.Metadata) > 0 {
				// Add metadata to the end of the content in sorted order
				keys := slices.Sorted(maps.Keys(item.Metadata))
				for _, key := range keys {
					value := item.Metadata[key]
					// Quote values that contain spaces
					if strings.Contains(value, " ") {
						value = `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
					}
					content += " " + key + ":" + value
				}
			}

			line = "- " + checkBox + " " + content
			buf.WriteString(line)
			buf.WriteString("\n")

		default:
			panic(fmt.Errorf("invalid item type %v", item.Type))
		}
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(buf.String())
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}
