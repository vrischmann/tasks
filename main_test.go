package main

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// createTestFile creates a temporary markdown file with the given content
func createTestFile(t *testing.T, content string) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "tasks_test_*.md")
	require.NoError(t, err)

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

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

// Integration Tests for CLI Commands

func TestIntegration_AddAndListTasks(t *testing.T) {
	// Create empty test file
	filename := createTestFile(t, "")

	// Test adding a task
	tm := &TaskManager{FilePath: filename}
	err := tm.Load()
	require.NoError(t, err)
	require.Empty(t, tm.Items)

	err = tm.AddTask("First task", nil, -1)
	require.NoError(t, err)
	err = tm.AddSection("Main Section", 1, -1)
	require.NoError(t, err)
	err = tm.AddTask("Second task", nil, -1)
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

// TestTaskManager_AddSection_Complete tests both success and error cases for AddSection
func TestTaskManager_AddSection_Complete(t *testing.T) {
	t.Run("error cases", func(t *testing.T) {
		testCases := []struct {
			name        string
			level       int
			afterIndex  int
			errContains string
		}{
			{"level zero", 0, -1, "invalid section level"},
			{"level negative", -1, -1, "invalid section level"},
			{"level too high", 7, -1, "invalid section level"},
			{"afterIndex negative", 1, -2, "invalid after index"},
			{"afterIndex too large", 1, 10, "invalid after index"},
			{"afterIndex equals length", 1, 2, "invalid after index"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				filename := createTestFile(t, "# Initial Section\n- [ ] Task\n")
				tm := &TaskManager{FilePath: filename}

				err := tm.Load()
				require.NoError(t, err)
				require.Len(t, tm.Items, 2)

				err = tm.AddSection("Test Section", tc.level, tc.afterIndex)
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			})
		}
	})

	t.Run("successful insertion at end", func(t *testing.T) {
		filename := createTestFile(t, "# First Section\n- [ ] Task 1\n")
		tm := &TaskManager{FilePath: filename}

		err := tm.Load()
		require.NoError(t, err)
		require.Len(t, tm.Items, 2)

		// Add section at the end
		err = tm.AddSection("New Section", 2, -1)
		require.NoError(t, err)
		require.Len(t, tm.Items, 3)
		require.Equal(t, "New Section", tm.Items[2].Content)
		require.Equal(t, 2, tm.Items[2].Level)
	})

	t.Run("successful insertion after specific index", func(t *testing.T) {
		filename := createTestFile(t, `# First Section
- [ ] Task 1
## Second Section
- [ ] Task 2
`)
		tm := &TaskManager{FilePath: filename}

		err := tm.Load()
		require.NoError(t, err)
		require.Len(t, tm.Items, 4)

		// Insert section after index 1 (Task 1)
		err = tm.AddSection("Inserted Section", 3, 1)
		require.NoError(t, err)
		require.Len(t, tm.Items, 5)

		// Check positions
		require.Equal(t, "First Section", tm.Items[0].Content)
		require.Equal(t, "Task 1", tm.Items[1].Content)
		require.Equal(t, "Inserted Section", tm.Items[2].Content)
		require.Equal(t, "Second Section", tm.Items[3].Content)
		require.Equal(t, "Task 2", tm.Items[4].Content)

		// Verify the inserted section properties
		inserted := tm.Items[2]
		require.Equal(t, TypeSection, inserted.Type)
		require.Equal(t, 3, inserted.Level)
		require.Equal(t, "Inserted Section", inserted.Content)
	})

	t.Run("successful insertion at beginning", func(t *testing.T) {
		filename := createTestFile(t, "# Existing Section\n- [ ] Task 1\n")
		tm := &TaskManager{FilePath: filename}

		err := tm.Load()
		require.NoError(t, err)
		require.Len(t, tm.Items, 2)

		// Insert section at beginning (after index -1 is not valid, so use 0 for after first item)
		err = tm.AddSection("Top Section", 1, 0)
		require.NoError(t, err)
		require.Len(t, tm.Items, 3)

		// Should be inserted at position 1 (after index 0)
		require.Equal(t, "Existing Section", tm.Items[0].Content)
		require.Equal(t, "Top Section", tm.Items[1].Content)
		require.Equal(t, "Task 1", tm.Items[2].Content)
	})

	t.Run("insert with different levels", func(t *testing.T) {
		filename := createTestFile(t, "# Main\n## Sub\n- [ ] Task\n")
		tm := &TaskManager{FilePath: filename}

		err := tm.Load()
		require.NoError(t, err)
		require.Len(t, tm.Items, 3)

		// Test different section levels
		levels := []int{1, 2, 3, 4, 5, 6}
		for _, level := range levels {
			t.Run(fmt.Sprintf("level %d", level), func(t *testing.T) {
				tmCopy := *tm // Work with a copy
				tmCopy.FilePath = createTestFile(t, "# Main\n## Sub\n- [ ] Task\n")

				err := tmCopy.Load()
				require.NoError(t, err)

				err = tmCopy.AddSection(fmt.Sprintf("Level %d", level), level, 1)
				require.NoError(t, err)
				require.Len(t, tmCopy.Items, 4)
				require.Equal(t, fmt.Sprintf("Level %d", level), tmCopy.Items[2].Content)
				require.Equal(t, level, tmCopy.Items[2].Level)
			})
		}
	})
}

// TestTaskManager_AddTask_ErrorCases tests error handling in AddTask
func TestTaskManager_AddTask_ErrorCases(t *testing.T) {
	filename := createTestFile(t, "# Test\n- [ ] Task 1\n")
	tm := &TaskManager{FilePath: filename}

	err := tm.Load()
	require.NoError(t, err)
	require.Len(t, tm.Items, 2)

	testCases := []struct {
		name        string
		afterIndex  int
		errContains string
	}{
		{"negative afterIndex", -2, "invalid after index"},
		{"afterIndex too large", 10, "invalid after index"},
		{"afterIndex equals length", 2, "invalid after index"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tm.AddTask("Test Task", nil, tc.afterIndex)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errContains)
		})
	}
}

// TestTaskManager_Load_ErrorHandling tests error cases in Load method
func TestTaskManager_Load_ErrorHandling(t *testing.T) {
	t.Run("non-existent file", func(t *testing.T) {
		tm := &TaskManager{FilePath: "/nonexistent/file.md"}
		err := tm.Load()
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("malformed markdown", func(t *testing.T) {
		// Create file with content that might cause issues
		filename := createTestFile(t, "# Valid section\nInvalid task format\n")
		tm := &TaskManager{FilePath: filename}

		// Should not error, just skip invalid lines
		err := tm.Load()
		require.NoError(t, err)
		require.Len(t, tm.Items, 1) // Only the valid section
	})
}

// TestTaskManager_Save_ErrorHandling tests error cases in Save method
func TestTaskManager_Save_ErrorHandling(t *testing.T) {
	tm := &TaskManager{
		FilePath: "/invalid/path/readonly/file.md",
		Items: []Item{
			{Type: TypeSection, Level: 1, Content: "Test"},
		},
	}

	err := tm.Save()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create file")
}

// TestSaveToFile_ErrorHandling tests error cases in saveToFile function
func TestSaveToFile_ErrorHandling(t *testing.T) {
	items := []Item{
		{Type: TypeSection, Level: 1, Content: "Test Section"},
		{Type: TypeTask, Content: "Test Task", Checked: func() *bool { b := false; return &b }()},
	}

	t.Run("invalid directory path", func(t *testing.T) {
		err := saveToFile("/invalid/path/does/not/exist.md", items)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to create file")
	})

	t.Run("readonly directory", func(t *testing.T) {
		// Create a read-only temporary directory
		tmpDir, err := os.MkdirTemp("", "readonly-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Make directory read-only
		err = os.Chmod(tmpDir, 0o444)
		if err == nil {
			defer os.Chmod(tmpDir, 0o755) // Restore permissions

			err = saveToFile(tmpDir+"/test.md", items)
			require.Error(t, err)
		} else {
			t.Skip("Cannot test readonly directory on this system")
		}
	})
}

// TestNewTaskManager tests the helper function
func TestNewTaskManager(t *testing.T) {
	t.Run("successful load", func(t *testing.T) {
		content := `# Test Section
- [ ] Test task
`
		filename := createTestFile(t, content)

		tm, err := NewTaskManager(filename)
		require.NoError(t, err)
		require.NotNil(t, tm)
		require.Len(t, tm.Items, 2)
		require.Equal(t, "Test Section", tm.Items[0].Content)
		require.Equal(t, "Test task", tm.Items[1].Content)
	})

	t.Run("file not found", func(t *testing.T) {
		tm, err := NewTaskManager("/nonexistent/file.md")
		require.Error(t, err)
		require.Nil(t, tm)
		require.Contains(t, err.Error(), "error loading file")
	})
}

// TestMetadataRoundTrip tests that metadata survives save/load cycles
func TestMetadataRoundTrip(t *testing.T) {
	filename := createTestFile(t, "# Section\n")

	tm := &TaskManager{FilePath: filename}
	err := tm.Load()
	require.NoError(t, err)

	// Add task with metadata
	task := Item{
		Type:    TypeTask,
		Content: "Task with metadata",
		Checked: func() *bool { b := false; return &b }(),
		Metadata: map[string]string{
			"priority": "high",
			"due":      "2024-12-25",
			"assignee": "test user",
			"tag":      "important",
		},
	}

	tm.Items = append(tm.Items, task)
	err = tm.Save()
	require.NoError(t, err)

	// Load and verify
	tm2 := &TaskManager{FilePath: filename}
	err = tm2.Load()
	require.NoError(t, err)
	require.Len(t, tm2.Items, 2)

	savedTask := tm2.Items[1]
	require.Equal(t, "Task with metadata", savedTask.Content)
	require.Equal(t, "high", savedTask.Metadata["priority"])
	require.Equal(t, "2024-12-25", savedTask.Metadata["due"])
	require.Equal(t, "test user", savedTask.Metadata["assignee"])
	require.Equal(t, "important", savedTask.Metadata["tag"])
}

// TestSaveToFile_MetadataQuoting tests metadata value quoting
func TestSaveToFile_MetadataQuoting(t *testing.T) {
	items := []Item{
		{
			Type:    TypeTask,
			Content: "Test task",
			Checked: func() *bool { b := false; return &b }(),
			Metadata: map[string]string{
				"nospace":    "value",
				"withspace":  "value with spaces",
				"withquotes": `value with "quotes"`,
				"withcolon":  "value:with:colons",
			},
		},
	}

	filename := createTestFile(t, "# Section\n")

	err := saveToFile(filename, items)
	require.NoError(t, err)

	// Read the file and check content
	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	fileContent := string(content)
	require.Contains(t, fileContent, "nospace:value")
	require.Contains(t, fileContent, `"value with spaces"`)
	require.Contains(t, fileContent, `"value with \"quotes\""`)
	require.Contains(t, fileContent, "withcolon:value:with:colons")
}

// Tests for parseItemID function (currently 0% coverage)
func TestParseItemID(t *testing.T) {
	t.Run("valid IDs", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected int
		}{
			{"1", 0},    // 1-based to 0-based
			{"5", 4},    // 1-based to 0-based
			{"100", 99}, // Large number
		}

		for _, tc := range testCases {
			result, err := parseItemID(tc.input)
			require.NoError(t, err, "Input: %s", tc.input)
			require.Equal(t, tc.expected, result, "Input: %s", tc.input)
		}
	})

	t.Run("invalid IDs", func(t *testing.T) {
		testCases := []string{
			"0",   // Zero is invalid (must be > 0)
			"-1",  // Negative numbers
			"-5",  // More negative numbers
			"abc", // Non-numeric
			"",    // Empty string
			" ",   // Whitespace
		}

		for _, input := range testCases {
			result, err := parseItemID(input)
			require.Error(t, err, "Input should be invalid: %s", input)
			require.Equal(t, -1, result, "Invalid input should return -1: %s", input)
		}
	})

	t.Run("edge cases that succeed", func(t *testing.T) {
		// These cases actually succeed because fmt.Sscanf parses the integer part
		testCases := []struct {
			input    string
			expected int
		}{
			{"1.5", 0}, // Parses as "1", then converts to 0-based
			{"1a", 0},  // Parses as "1", ignores the "a"
			{"1 2", 0}, // Parses as "1", ignores the rest
		}

		for _, tc := range testCases {
			result, err := parseItemID(tc.input)
			require.NoError(t, err, "Input: %s", tc.input)
			require.Equal(t, tc.expected, result, "Input: %s", tc.input)
		}
	})
}

// Tests for formatItem function (currently 0% coverage)
func TestFormatItem(t *testing.T) {
	t.Run("section formatting", func(t *testing.T) {
		item := Item{
			Type:    TypeSection,
			Level:   1,
			Content: "Main Section",
			Checked: nil,
		}

		result := formatItem(item, 0) // Index 0 = ID 1
		require.Contains(t, result, "1    ")
		require.Contains(t, result, "# Main Section")
	})

	t.Run("task formatting", func(t *testing.T) {
		t.Run("unchecked task", func(t *testing.T) {
			checked := false
			item := Item{
				Type:    TypeTask,
				Content: "Test task",
				Checked: &checked,
			}

			result := formatItem(item, 4) // Index 4 = ID 5
			require.Contains(t, result, "5    ")
			require.Contains(t, result, "- [ ] Test task")
		})

		t.Run("checked task", func(t *testing.T) {
			checked := true
			item := Item{
				Type:    TypeTask,
				Content: "Completed task",
				Checked: &checked,
			}

			result := formatItem(item, 9) // Index 9 = ID 10
			require.Contains(t, result, "10   ")
			require.Contains(t, result, "- [x] Completed task")
		})
	})

	t.Run("various section levels", func(t *testing.T) {
		for level := 1; level <= 6; level++ {
			item := Item{
				Type:    TypeSection,
				Level:   level,
				Content: fmt.Sprintf("Level %d Section", level),
				Checked: nil,
			}

			result := formatItem(item, 0)
			expectedHeader := strings.Repeat("#", level) + " " + fmt.Sprintf("Level %d Section", level)
			require.Contains(t, result, expectedHeader)
		}
	})

	t.Run("terminal vs non-terminal formatting", func(t *testing.T) {
		// Test both terminal and non-terminal output
		// Since isTerminal() depends on actual terminal state, we test the logic paths
		item := Item{
			Type:    TypeTask,
			Content: "Test task",
			Checked: func() *bool { b := false; return &b }(),
		}

		result := formatItem(item, 0)
		// Should contain the basic components regardless of terminal state
		require.Contains(t, result, "1    ")
		require.Contains(t, result, "- [ ] Test task")
	})
}

// Additional tests for parseMarkdownFile edge cases
func TestParseMarkdownFile_MoreEdgeCases(t *testing.T) {
	t.Run("file does not exist", func(t *testing.T) {
		items, err := parseMarkdownFile("/nonexistent/file.md")
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.Empty(t, items)
	})

	t.Run("complex mixed content", func(t *testing.T) {
		content := `# Project Overview

This is some regular text that should be ignored.

## Phase 1
- [ ] Initialize project structure
    Some indented text that's not a task

### Sub-phase 1.1
- [x] Create repository due:yesterday
- [ ] Setup CI/CD pipeline priority:high

## Phase 2

More regular text here.

- [ ] Implement core features
- [x] Write documentation

# Different Project

- [ ] Start different project
`
		filename := createTestFile(t, content)

		items, err := parseMarkdownFile(filename)
		require.NoError(t, err)

		// Should only capture headers and tasks, ignoring regular text
		expectedTypes := []ItemType{
			TypeSection, // Project Overview
			TypeSection, // Phase 1
			TypeTask,    // Initialize project structure
			TypeSection, // Sub-phase 1.1
			TypeTask,    // Create repository
			TypeTask,    // Setup CI/CD pipeline
			TypeSection, // Phase 2
			TypeTask,    // Implement core features
			TypeTask,    // Write documentation
			TypeSection, // Different Project
			TypeTask,    // Start different project
		}

		require.Len(t, items, len(expectedTypes))

		for i, expectedType := range expectedTypes {
			require.Equal(t, expectedType, items[i].Type, "Item %d should be type %d", i, expectedType)
		}

		// Check specific items with metadata
		require.Equal(t, "Create repository", items[4].Content)
		require.True(t, *items[4].Checked)
		require.Equal(t, "yesterday", items[4].Metadata["due"])

		require.Equal(t, "Setup CI/CD pipeline", items[5].Content)
		require.False(t, *items[5].Checked)
		require.Equal(t, "high", items[5].Metadata["priority"])
	})

	t.Run("very long lines", func(t *testing.T) {
		longContent := strings.Repeat("very long content ", 100)
		content := fmt.Sprintf("# Very Long Section %s\n- [ ] Very long task %s priority:high", longContent, longContent)
		filename := createTestFile(t, content)

		items, err := parseMarkdownFile(filename)
		require.NoError(t, err)
		require.Len(t, items, 2)
		require.Contains(t, items[0].Content, "Very Long Section")
		require.Contains(t, items[1].Content, "Very long task")
		require.Equal(t, "high", items[1].Metadata["priority"])
	})
}
