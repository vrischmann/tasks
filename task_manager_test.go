package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
