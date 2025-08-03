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

func TestLineNumberMapping(t *testing.T) {
	// Create a file with empty lines and spacing
	content := `# Main Section

This has empty lines for spacing.

## Subsection

- [ ] First task
- [x] Second task

More text here.

### Deep section
- [ ] Deep task
`
	filename := createTestFile(t, content)

	items, err := parseMarkdownFile(filename)
	require.NoError(t, err)
	require.Len(t, items, 6) // 3 sections + 3 tasks

	// Verify that line numbers are tracked correctly
	require.Equal(t, 1, items[0].LineNumber, "Main Section should be on line 1")
	require.Equal(t, 5, items[1].LineNumber, "Subsection should be on line 5")
	require.Equal(t, 7, items[2].LineNumber, "First task should be on line 7")
	require.Equal(t, 8, items[3].LineNumber, "Second task should be on line 8")
	require.Equal(t, 12, items[4].LineNumber, "Deep section should be on line 12")
	require.Equal(t, 13, items[5].LineNumber, "Deep task should be on line 13")
}

// Fuzzy Search Tests

func TestFuzzyMatch(t *testing.T) {
	t.Run("exact matches", func(t *testing.T) {
		t.Run("identical strings", func(t *testing.T) {
			score := fuzzyMatch("test", "test")
			require.Equal(t, 1.0, score, "Exact match should return score of 1.0")
		})

		t.Run("case insensitive", func(t *testing.T) {
			testCases := []struct {
				pattern, text string
			}{
				{"TEST", "test"},
				{"test", "TEST"},
				{"TeSt", "tEsT"},
			}

			for _, tc := range testCases {
				score := fuzzyMatch(tc.pattern, tc.text)
				require.Equal(t, 1.0, score, "Case insensitive exact match should return 1.0 for %s vs %s", tc.pattern, tc.text)
			}
		})
	})

	t.Run("substring matches", func(t *testing.T) {
		t.Run("exact substring", func(t *testing.T) {
			score := fuzzyMatch("test", "this is a test string")
			require.Equal(t, 0.8, score, "Exact substring match should return 0.8")
		})

		t.Run("case insensitive substring", func(t *testing.T) {
			testCases := []struct {
				pattern, text string
			}{
				{"TEST", "this is a test string"},
				{"test", "this is a TEST string"},
			}

			for _, tc := range testCases {
				score := fuzzyMatch(tc.pattern, tc.text)
				require.Equal(t, 0.8, score, "Case insensitive substring match should return 0.8 for %s vs %s", tc.pattern, tc.text)
			}
		})
	})

	t.Run("fuzzy character matches", func(t *testing.T) {
		t.Run("sequential characters", func(t *testing.T) {
			score := fuzzyMatch("btn", "button")
			require.GreaterOrEqual(t, score, 0.3, "Fuzzy match should return meaningful score")
			require.Less(t, score, 0.8, "Fuzzy match should be less than substring match")
		})

		t.Run("partial character match", func(t *testing.T) {
			score := fuzzyMatch("crate", "create")
			require.GreaterOrEqual(t, score, 0.3, "Partial character match should return meaningful score")
		})

		t.Run("order matters", func(t *testing.T) {
			score1 := fuzzyMatch("abc", "aabbcc") // in order
			score2 := fuzzyMatch("abc", "ccbbaa") // reverse order

			require.GreaterOrEqual(t, score1, 0.3, "In-order characters should match")
			require.Equal(t, 0.0, score2, "Out-of-order characters should not match")
		})

		t.Run("long text", func(t *testing.T) {
			longText := "this is a very long text string with many words in it for testing purposes"
			score := fuzzyMatch("test", longText)
			require.Greater(t, score, 0.3, "Should find pattern in long text")
		})
	})

	t.Run("no matches", func(t *testing.T) {
		t.Run("completely different", func(t *testing.T) {
			score := fuzzyMatch("xyz", "button")
			require.Equal(t, 0.0, score, "No character matches should return 0.0")
		})

		t.Run("incomplete pattern", func(t *testing.T) {
			score := fuzzyMatch("buttoncomponent", "button")
			require.Equal(t, 0.0, score, "Incomplete match (missing pattern chars) should return 0.0")
		})
	})

	t.Run("empty strings", func(t *testing.T) {
		testCases := []struct {
			name          string
			pattern, text string
			expected      float64
		}{
			{"empty pattern with text", "", "test", 0.0},
			{"pattern with empty text", "test", "", 0.0},
			{"both empty", "", "", 1.0},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				score := fuzzyMatch(tc.pattern, tc.text)
				require.Equal(t, tc.expected, score)
			})
		}
	})
}

func TestSearchItems(t *testing.T) {
	t.Run("edge cases", func(t *testing.T) {
		t.Run("empty items", func(t *testing.T) {
			var items []Item
			results := searchItems(items, []string{"test"})
			require.Empty(t, results, "Empty items should return no results")
		})

		t.Run("empty query", func(t *testing.T) {
			items := []Item{
				{Type: TypeTask, Content: "Test task", Checked: func() *bool { b := false; return &b }()},
			}
			results := searchItems(items, []string{})
			require.Empty(t, results, "Empty query should return no results")
		})

		t.Run("minimum score threshold", func(t *testing.T) {
			items := []Item{
				{Type: TypeTask, Content: "completely unrelated content xyz", Checked: func() *bool { b := false; return &b }()},
				{Type: TypeTask, Content: "another unrelated item abc", Checked: func() *bool { b := false; return &b }()},
			}

			results := searchItems(items, []string{"searchterm"})
			require.Empty(t, results, "Should not return results below minimum score threshold")
		})
	})

	t.Run("single matches", func(t *testing.T) {
		items := []Item{
			{Type: TypeTask, Content: "Setup React project", Checked: func() *bool { b := false; return &b }()},
			{Type: TypeTask, Content: "Create components", Checked: func() *bool { b := false; return &b }()},
			{Type: TypeSection, Level: 1, Content: "Frontend", Checked: nil},
		}

		results := searchItems(items, []string{"react"})
		require.Len(t, results, 1, "Should find one match")
		require.Equal(t, "Setup React project", results[0].Item.Content)
		require.Equal(t, 0, results[0].Index)
		require.Greater(t, results[0].Score, 0.3)
	})

	t.Run("multiple matches", func(t *testing.T) {
		t.Run("same search term in different items", func(t *testing.T) {
			items := []Item{
				{Type: TypeTask, Content: "Create form component", Checked: func() *bool { b := false; return &b }()},
				{Type: TypeTask, Content: "Login form", Checked: func() *bool { b := false; return &b }()},
				{Type: TypeTask, Content: "Registration form", Checked: func() *bool { b := false; return &b }()},
				{Type: TypeTask, Content: "Button component", Checked: func() *bool { b := false; return &b }()},
			}

			results := searchItems(items, []string{"form"})
			require.Len(t, results, 3, "Should find three form matches")

			// Results should be sorted by score (highest first), but some may have equal scores
			if len(results) >= 2 {
				require.GreaterOrEqual(t, results[0].Score, results[1].Score, "Results should be ordered by score")
			}
			if len(results) >= 3 {
				require.GreaterOrEqual(t, results[1].Score, results[2].Score, "Results should be ordered by score")
			}
		})

		t.Run("score ordering", func(t *testing.T) {
			items := []Item{
				{Type: TypeTask, Content: "button", Checked: func() *bool { b := false; return &b }()},                  // exact match
				{Type: TypeTask, Content: "create button component", Checked: func() *bool { b := false; return &b }()}, // substring match
				{Type: TypeTask, Content: "big unique task name", Checked: func() *bool { b := false; return &b }()},    // fuzzy match
			}

			results := searchItems(items, []string{"button"})
			require.GreaterOrEqual(t, len(results), 2, "Should find multiple matches")

			// Exact match should score highest, substring match should be next
			if len(results) >= 2 {
				require.Greater(t, results[0].Score, results[1].Score, "Results should be ordered by score")
				require.Equal(t, "button", results[0].Item.Content, "Exact match should be first")
			}
		})
	})

	t.Run("query variations", func(t *testing.T) {
		t.Run("multiple query terms", func(t *testing.T) {
			items := []Item{
				{Type: TypeTask, Content: "Setup authentication system", Checked: func() *bool { b := false; return &b }()},
				{Type: TypeTask, Content: "Password reset functionality", Checked: func() *bool { b := false; return &b }()},
				{Type: TypeTask, Content: "User login form", Checked: func() *bool { b := false; return &b }()},
			}

			results := searchItems(items, []string{"auth", "password"})
			require.Len(t, results, 2, "Should find matches for multi-term query")
		})

		t.Run("case insensitive", func(t *testing.T) {
			items := []Item{
				{Type: TypeTask, Content: "Setup React Project", Checked: func() *bool { b := false; return &b }()},
				{Type: TypeTask, Content: "create components", Checked: func() *bool { b := false; return &b }()},
			}

			testCases := []string{"REACT", "react", "React"}
			for _, query := range testCases {
				results := searchItems(items, []string{query})
				require.Len(t, results, 1, "Query %s should find match", query)
				require.Equal(t, "Setup React Project", results[0].Item.Content)
			}
		})
	})

	t.Run("item types", func(t *testing.T) {
		t.Run("sections and tasks", func(t *testing.T) {
			items := []Item{
				{Type: TypeSection, Level: 1, Content: "Authentication", Checked: nil},
				{Type: TypeTask, Content: "JWT implementation", Checked: func() *bool { b := false; return &b }()},
				{Type: TypeTask, Content: "Password authentication", Checked: func() *bool { b := false; return &b }()},
				{Type: TypeSection, Level: 2, Content: "UI Components", Checked: nil},
			}

			results := searchItems(items, []string{"auth"})
			require.GreaterOrEqual(t, len(results), 2, "Should find matches in both sections and tasks")

			// Should include both the section and the task
			foundSection := false
			foundTask := false
			for _, result := range results {
				if result.Item.Type == TypeSection && result.Item.Content == "Authentication" {
					foundSection = true
				}
				if result.Item.Type == TypeTask && result.Item.Content == "Password authentication" {
					foundTask = true
				}
			}
			require.True(t, foundSection, "Should find section match")
			require.True(t, foundTask, "Should find task match")
		})
	})
}

// Integration tests for handleSearch would require capturing output,
// which is more complex. These unit tests cover the core functionality.