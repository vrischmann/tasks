package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// createTestFile creates a temporary markdown file with the given content
func createTestFile(t *testing.T, content string) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "tasks_test_*.md")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}

	tmpFile.Close()

	// Clean up the file when the test finishes
	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	return tmpFile.Name()
}

func TestParseMarkdownFile_EmptyFile(t *testing.T) {
	filename := createTestFile(t, "")

	items, err := parseMarkdownFile(filename)
	require.NoError(t, err)
	require.Empty(t, items, "Expected 0 items for empty file")
}

func TestParseMarkdownFile_SingleTask(t *testing.T) {
	content := "- [ ] Test task\n"
	filename := createTestFile(t, content)

	items, err := parseMarkdownFile(filename)
	require.NoError(t, err)
	require.Len(t, items, 1)

	item := items[0]
	require.Equal(t, TypeTask, item.Type)
	require.Equal(t, "Test task", item.Content)
	require.NotNil(t, item.Checked)
	require.False(t, *item.Checked, "Expected unchecked task")
}

func TestParseMarkdownFile_CompletedTask(t *testing.T) {
	content := "- [x] Completed task\n"
	filename := createTestFile(t, content)

	items, err := parseMarkdownFile(filename)
	require.NoError(t, err)
	require.Len(t, items, 1)

	item := items[0]
	require.Equal(t, "Completed task", item.Content)
	require.NotNil(t, item.Checked)
	require.True(t, *item.Checked, "Expected checked task")
}

func TestParseMarkdownFile_Section(t *testing.T) {
	content := "# Main Section\n"
	filename := createTestFile(t, content)

	items, err := parseMarkdownFile(filename)
	require.NoError(t, err)
	require.Len(t, items, 1)

	item := items[0]
	require.Equal(t, TypeSection, item.Type)
	require.Equal(t, "Main Section", item.Content)
	require.Equal(t, 1, item.Level)
	require.Nil(t, item.Checked, "Section should not have checked status")
}

func TestParseMarkdownFile_Mixed(t *testing.T) {
	content := `# Project Tasks
- [ ] Setup project
- [x] Create structure
## UI Components
- [ ] Button component
`
	filename := createTestFile(t, content)

	items, err := parseMarkdownFile(filename)
	require.NoError(t, err)
	require.Len(t, items, 5) // Fixed: there are 5 items total

	// First item: section
	require.Equal(t, TypeSection, items[0].Type)
	require.Equal(t, "Project Tasks", items[0].Content)
	require.Equal(t, 1, items[0].Level)

	// Second item: unchecked task
	require.Equal(t, TypeTask, items[1].Type)
	require.Equal(t, "Setup project", items[1].Content)
	require.False(t, *items[1].Checked)

	// Third item: checked task
	require.Equal(t, TypeTask, items[2].Type)
	require.Equal(t, "Create structure", items[2].Content)
	require.True(t, *items[2].Checked)

	// Fourth item: subsection
	require.Equal(t, TypeSection, items[3].Type)
	require.Equal(t, "UI Components", items[3].Content)
	require.Equal(t, 2, items[3].Level)

	// Fifth item: task under subsection
	require.Equal(t, TypeTask, items[4].Type)
	require.Equal(t, "Button component", items[4].Content)
	require.False(t, *items[4].Checked)
}

func TestSaveToFile(t *testing.T) {
	// Create test items
	items := []Item{
		{Type: TypeSection, Level: 1, Content: "Main Section", Checked: nil},
		{Type: TypeTask, Level: 0, Content: "First task", Checked: func() *bool { b := false; return &b }()},
		{Type: TypeTask, Level: 0, Content: "Second task", Checked: func() *bool { b := true; return &b }()},
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "save_test_*.md")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Save items to file
	err = saveToFile(tmpFile.Name(), items)
	require.NoError(t, err)

	// Read back and verify
	savedItems, err := parseMarkdownFile(tmpFile.Name())
	require.NoError(t, err)
	require.Len(t, savedItems, 3)

	require.Equal(t, "Main Section", savedItems[0].Content)
	require.Equal(t, "First task", savedItems[1].Content)
	require.False(t, *savedItems[1].Checked)
	require.Equal(t, "Second task", savedItems[2].Content)
	require.True(t, *savedItems[2].Checked)
}

func TestDeleteItem_Task(t *testing.T) {
	items := []Item{
		{Type: TypeTask, Content: "Task 1", Checked: func() *bool { b := false; return &b }()},
		{Type: TypeTask, Content: "Task 2", Checked: func() *bool { b := false; return &b }()},
		{Type: TypeTask, Content: "Task 3", Checked: func() *bool { b := false; return &b }()},
	}

	// Delete middle task
	result := deleteItem(items, 1)
	require.Len(t, result, 2)
	require.Equal(t, "Task 1", result[0].Content)
	require.Equal(t, "Task 3", result[1].Content)
}

func TestDeleteItem_Section(t *testing.T) {
	items := []Item{
		{Type: TypeSection, Level: 1, Content: "Section 1", Checked: nil},
		{Type: TypeTask, Level: 0, Content: "Task 1", Checked: func() *bool { b := false; return &b }()},
		{Type: TypeTask, Level: 0, Content: "Task 2", Checked: func() *bool { b := false; return &b }()},
		{Type: TypeSection, Level: 1, Content: "Section 2", Checked: nil},
		{Type: TypeTask, Level: 0, Content: "Task 3", Checked: func() *bool { b := false; return &b }()},
	}

	// Delete first section (should remove section and its tasks)
	result := deleteItem(items, 0)
	require.Len(t, result, 2)
	require.Equal(t, "Section 2", result[0].Content)
	require.Equal(t, "Task 3", result[1].Content)
}

func TestGetVersion(t *testing.T) {
	version := getVersion()
	require.NotEmpty(t, version)
	// Version should be either a semantic version, commit hash, or "dev"
	require.True(t, version != "unknown")
}

// TaskManager Tests

func TestTaskManager_LoadAndSave(t *testing.T) {
	content := `# Test Section
- [ ] Test task
- [x] Completed task
`
	filename := createTestFile(t, content)

	tm := &TaskManager{FilePath: filename}
	err := tm.Load()
	require.NoError(t, err)
	require.Len(t, tm.Items, 3)

	// Modify an item
	err = tm.ToggleTask(1, true) // Toggle "Test task" to completed
	require.NoError(t, err)

	// Save changes
	err = tm.Save()
	require.NoError(t, err)

	// Load again to verify persistence
	tm2 := &TaskManager{FilePath: filename}
	err = tm2.Load()
	require.NoError(t, err)
	require.True(t, *tm2.Items[1].Checked, "Task should be marked as completed")
}

func TestTaskManager_ToggleTask(t *testing.T) {
	content := `- [ ] Test task
- [x] Completed task
`
	filename := createTestFile(t, content)

	tm := &TaskManager{FilePath: filename}
	err := tm.Load()
	require.NoError(t, err)

	// Toggle incomplete task to complete
	err = tm.ToggleTask(0, true)
	require.NoError(t, err)
	require.True(t, *tm.Items[0].Checked)

	// Toggle complete task to incomplete
	err = tm.ToggleTask(1, false)
	require.NoError(t, err)
	require.False(t, *tm.Items[1].Checked)

	// Try to toggle a section (should fail)
	tm.Items = append(tm.Items, Item{Type: TypeSection, Level: 1, Content: "Section", Checked: nil})
	err = tm.ToggleTask(2, true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a task")
}

func TestTaskManager_AddTask(t *testing.T) {
	content := `- [ ] Existing task`
	filename := createTestFile(t, content)

	tm := &TaskManager{FilePath: filename}
	err := tm.Load()
	require.NoError(t, err)
	require.Len(t, tm.Items, 1)

	// Add task at the end
	err = tm.AddTask("New task", -1)
	require.NoError(t, err)
	require.Len(t, tm.Items, 2)
	require.Equal(t, "New task", tm.Items[1].Content)
	require.Equal(t, TypeTask, tm.Items[1].Type)
	require.False(t, *tm.Items[1].Checked)

	// Add task after index 0
	err = tm.AddTask("Middle task", 0)
	require.NoError(t, err)
	require.Len(t, tm.Items, 3)
	require.Equal(t, "Middle task", tm.Items[1].Content)
	require.Equal(t, "New task", tm.Items[2].Content)
}

func TestTaskManager_AddSection(t *testing.T) {
	content := `- [ ] Existing task`
	filename := createTestFile(t, content)

	tm := &TaskManager{FilePath: filename}
	err := tm.Load()
	require.NoError(t, err)

	// Add section at the end
	err = tm.AddSection("New Section", 2, -1)
	require.NoError(t, err)
	require.Len(t, tm.Items, 2)
	require.Equal(t, "New Section", tm.Items[1].Content)
	require.Equal(t, TypeSection, tm.Items[1].Type)
	require.Equal(t, 2, tm.Items[1].Level)
	require.Nil(t, tm.Items[1].Checked)

	// Test invalid section level
	err = tm.AddSection("Invalid", 7, -1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid section level")
}

func TestTaskManager_RemoveItem(t *testing.T) {
	content := `# Section
- [ ] Task 1
- [ ] Task 2
`
	filename := createTestFile(t, content)

	tm := &TaskManager{FilePath: filename}
	err := tm.Load()
	require.NoError(t, err)
	require.Len(t, tm.Items, 3)

	// Remove middle task
	err = tm.RemoveItem(1)
	require.NoError(t, err)
	require.Len(t, tm.Items, 2)
	require.Equal(t, "Task 2", tm.Items[1].Content)

	// Test invalid index
	err = tm.RemoveItem(10)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid item index")
}

func TestTaskManager_GetItem(t *testing.T) {
	content := `- [ ] Test task`
	filename := createTestFile(t, content)

	tm := &TaskManager{FilePath: filename}
	err := tm.Load()
	require.NoError(t, err)

	// Get valid item
	item, err := tm.GetItem(0)
	require.NoError(t, err)
	require.Equal(t, "Test task", item.Content)

	// Get invalid index
	_, err = tm.GetItem(10)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid item index")

	// Get negative index
	_, err = tm.GetItem(-1)
	require.Error(t, err)
}

// Integration Tests for CLI Commands

func TestIntegration_AddAndListTasks(t *testing.T) {
	// Create empty test file
	filename := createTestFile(t, "")

	// Test adding a task
	tm := &TaskManager{FilePath: filename}
	err := tm.Load()
	require.NoError(t, err)
	require.Empty(t, tm.Items)

	err = tm.AddTask("First task", -1)
	require.NoError(t, err)
	err = tm.AddSection("Main Section", 1, -1)
	require.NoError(t, err)
	err = tm.AddTask("Second task", -1)
	require.NoError(t, err)

	err = tm.Save()
	require.NoError(t, err)

	// Verify by loading again
	tm2 := &TaskManager{FilePath: filename}
	err = tm2.Load()
	require.NoError(t, err)
	require.Len(t, tm2.Items, 3)
	require.Equal(t, "First task", tm2.Items[0].Content)
	require.Equal(t, "Main Section", tm2.Items[1].Content)
	require.Equal(t, "Second task", tm2.Items[2].Content)
}

func TestIntegration_TaskLifecycle(t *testing.T) {
	// Create test file with a task
	content := `- [ ] Test task`
	filename := createTestFile(t, content)

	tm := &TaskManager{FilePath: filename}

	// Load the task
	err := tm.Load()
	require.NoError(t, err)
	require.Len(t, tm.Items, 1)
	require.False(t, *tm.Items[0].Checked)

	// Mark it as done
	err = tm.ToggleTask(0, true)
	require.NoError(t, err)
	err = tm.Save()
	require.NoError(t, err)

	// Reload and verify it's marked as done
	err = tm.Load()
	require.NoError(t, err)
	require.True(t, *tm.Items[0].Checked)

	// Mark it as undone
	err = tm.ToggleTask(0, false)
	require.NoError(t, err)
	err = tm.Save()
	require.NoError(t, err)

	// Reload and verify it's unmarked
	err = tm.Load()
	require.NoError(t, err)
	require.False(t, *tm.Items[0].Checked)

	// Remove the task
	err = tm.RemoveItem(0)
	require.NoError(t, err)
	err = tm.Save()
	require.NoError(t, err)

	// Reload and verify it's gone
	err = tm.Load()
	require.NoError(t, err)
	require.Empty(t, tm.Items)
}

func TestIntegration_SectionWithChildren(t *testing.T) {
	content := `# Main Section
- [ ] Task 1
- [ ] Task 2
## Sub Section
- [ ] Task 3
`
	filename := createTestFile(t, content)

	tm := &TaskManager{FilePath: filename}
	err := tm.Load()
	require.NoError(t, err)
	require.Len(t, tm.Items, 5)

	// Let's debug what we have first
	for i, item := range tm.Items {
		t.Logf("Item %d: Type=%d, Level=%d, Content=%s", i, item.Type, item.Level, item.Content)
	}

	// Remove the main section (should remove its children too)
	err = tm.RemoveItem(0) // Remove "Main Section" at index 0
	require.NoError(t, err)

	// Let's see what's left
	t.Logf("After removal, items count: %d", len(tm.Items))
	for i, item := range tm.Items {
		t.Logf("Remaining Item %d: Type=%d, Level=%d, Content=%s", i, item.Type, item.Level, item.Content)
	}

	// The deleteItem logic removes sections and all items until it finds another section at same or higher level
	// Since "Main Section" is level 1, it removes everything until it finds another level 1 section (or end of file)
	// So everything gets removed in this case. Let's adjust the test.
	require.Len(t, tm.Items, 0, "All items should be removed when removing the top-level section")
}

func TestIntegration_SectionDeletionWithSiblings(t *testing.T) {
	content := `# First Section
- [ ] Task 1
## Sub Section
- [ ] Task 2
# Second Section
- [ ] Task 3
`
	filename := createTestFile(t, content)

	tm := &TaskManager{FilePath: filename}
	err := tm.Load()
	require.NoError(t, err)
	require.Len(t, tm.Items, 6)

	// Remove the first section (should remove it and its children until the next same-level section)
	err = tm.RemoveItem(0) // Remove "First Section" at index 0
	require.NoError(t, err)

	// Should only have the second section and its task left
	require.Len(t, tm.Items, 2)
	require.Equal(t, "Second Section", tm.Items[0].Content)
	require.Equal(t, "Task 3", tm.Items[1].Content)
}

func TestErrorHandling_InvalidOperations(t *testing.T) {
	content := `# Section Only`
	filename := createTestFile(t, content)

	tm := &TaskManager{FilePath: filename}
	err := tm.Load()
	require.NoError(t, err)

	// Try to toggle a section (should fail)
	err = tm.ToggleTask(0, true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a task")

	// Try invalid indices
	err = tm.ToggleTask(10, true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid item index")

	err = tm.RemoveItem(-1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid item index")
}
