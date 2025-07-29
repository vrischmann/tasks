package main

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
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

func TestInitialModel_SingleTask(t *testing.T) {
	// Create a test file with a single task
	content := "- [ ] Test task\n"
	filename := createTestFile(t, content)

	// Initialize the model
	model := initialModel(filename)

	// Verify initial state
	require.Len(t, model.items, 1)
	require.Equal(t, TypeTask, model.items[0].Type)
	require.Equal(t, "Test task", model.items[0].Content)
	require.False(t, *model.items[0].Checked)
	require.Equal(t, 0, model.cursor)
	require.Len(t, model.visibleItems, 1)
}

func TestTaskToggle(t *testing.T) {
	// Create a test file with a single unchecked task
	content := "- [ ] Test task\n"
	filename := createTestFile(t, content)

	// Create test model
	tm := teatest.NewTestModel(
		t,
		initialModel(filename),
		teatest.WithInitialTermSize(80, 24),
	)

	// Send space key to toggle the task
	tm.Send(tea.KeyMsg{
		Type:  tea.KeySpace,
		Runes: []rune{' '},
	})

	// Send quit to terminate the program
	tm.Send(tea.KeyMsg{
		Type:  tea.KeyCtrlC,
		Runes: []rune{},
	})

	// Get final model state
	fm := tm.FinalModel(t)
	finalModel, ok := fm.(Model)
	require.True(t, ok, "Expected Model type")

	// Verify the task was toggled
	require.True(t, *finalModel.items[0].Checked, "Expected task to be checked after toggle")
	require.True(t, finalModel.dirty, "Expected model to be marked as dirty after toggle")
}

func TestNavigationSingleTask(t *testing.T) {
	// Create a test file with a single task
	content := "- [ ] Test task\n"
	filename := createTestFile(t, content)

	// Create test model
	tm := teatest.NewTestModel(
		t,
		initialModel(filename),
		teatest.WithInitialTermSize(80, 24),
	)

	// Try to move down (should stay at 0 since there's only one item)
	tm.Send(tea.KeyMsg{
		Type:  tea.KeyDown,
		Runes: []rune{},
	})

	// Try to move up (should stay at 0)
	tm.Send(tea.KeyMsg{
		Type:  tea.KeyUp,
		Runes: []rune{},
	})

	// Send quit to terminate the program
	tm.Send(tea.KeyMsg{
		Type:  tea.KeyCtrlC,
		Runes: []rune{},
	})

	// Get final model state
	fm := tm.FinalModel(t)
	finalModel, ok := fm.(Model)
	require.True(t, ok, "Expected Model type")

	// Cursor should still be at 0
	require.Equal(t, 0, finalModel.cursor, "Expected cursor at 0")
}

func TestEmptyFile(t *testing.T) {
	// Create an empty test file
	filename := createTestFile(t, "")

	// Initialize the model
	model := initialModel(filename)

	// Verify initial state for empty file
	require.Empty(t, model.items, "Expected 0 items for empty file")
	require.Equal(t, 0, model.cursor, "Expected cursor at 0")
	require.Empty(t, model.visibleItems, "Expected 0 visible items")
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

func TestMultipleTaskNavigation(t *testing.T) {
	// Create a test file with multiple tasks
	content := `- [ ] First task
- [ ] Second task
- [ ] Third task
`
	filename := createTestFile(t, content)

	// Create test model
	tm := teatest.NewTestModel(
		t,
		initialModel(filename),
		teatest.WithInitialTermSize(80, 24),
	)

	// Move down twice
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})

	// Toggle the third task
	tm.Send(tea.KeyMsg{Type: tea.KeySpace})

	// Move back up once
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

	// Get final model state
	fm := tm.FinalModel(t)
	finalModel, ok := fm.(Model)
	require.True(t, ok, "Expected Model type")

	// Verify cursor position (should be at index 1 - second task)
	require.Equal(t, 1, finalModel.cursor, "Expected cursor at 1")

	// Verify third task was toggled
	require.True(t, *finalModel.items[2].Checked, "Expected third task to be checked")

	// Verify first and second tasks are still unchecked
	require.False(t, *finalModel.items[0].Checked, "Expected first task to be unchecked")
	require.False(t, *finalModel.items[1].Checked, "Expected second task to be unchecked")
}

func TestTaskToggleMultiple(t *testing.T) {
	// Create a test file with two tasks
	content := `- [ ] Task one
- [ ] Task two
`
	filename := createTestFile(t, content)

	// Create test model
	tm := teatest.NewTestModel(
		t,
		initialModel(filename),
		teatest.WithInitialTermSize(80, 24),
	)

	// Toggle first task
	tm.Send(tea.KeyMsg{Type: tea.KeySpace})

	// Move to second task
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})

	// Toggle second task
	tm.Send(tea.KeyMsg{Type: tea.KeySpace})

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

	// Get final model state
	fm := tm.FinalModel(t)
	finalModel, ok := fm.(Model)
	require.True(t, ok, "Expected Model type")

	// Verify both tasks are checked
	require.True(t, *finalModel.items[0].Checked, "Expected first task to be checked")
	require.True(t, *finalModel.items[1].Checked, "Expected second task to be checked")

	// Verify model is dirty
	require.True(t, finalModel.dirty, "Expected model to be dirty after changes")
}

func TestBoundaryNavigation(t *testing.T) {
	// Create a test file with two tasks
	content := `- [ ] First task
- [ ] Second task
`
	filename := createTestFile(t, content)

	// Create test model
	tm := teatest.NewTestModel(
		t,
		initialModel(filename),
		teatest.WithInitialTermSize(80, 24),
	)

	// Try to move up from first position (should stay at 0)
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})

	// Move to second task
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})

	// Try to move down past last item (should stay at 1)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

	// Get final model state
	fm := tm.FinalModel(t)
	finalModel, ok := fm.(Model)
	require.True(t, ok, "Expected Model type")

	// Cursor should be at the last item (index 1)
	require.Equal(t, 1, finalModel.cursor, "Expected cursor at 1")
}
