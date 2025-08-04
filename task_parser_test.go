package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// parseTask Function Tests

func TestParseTask_ValidTaskFormat(t *testing.T) {
	t.Run("incomplete task without metadata", func(t *testing.T) {
		result := parseTask("- [ ] Just a simple task")
		require.Equal(t, "Just a simple task", result.Description)
		require.False(t, result.Completed)
		require.Empty(t, result.Metadata)
	})

	t.Run("completed task without metadata", func(t *testing.T) {
		result := parseTask("- [x] A completed task")
		require.Equal(t, "A completed task", result.Description)
		require.True(t, result.Completed)
		require.Empty(t, result.Metadata)
	})

	t.Run("task with single metadata", func(t *testing.T) {
		result := parseTask("- [ ] Review documentation priority:A")
		require.Equal(t, "Review documentation", result.Description)
		require.False(t, result.Completed)
		require.Len(t, result.Metadata, 1)
		require.Equal(t, "A", result.Metadata["priority"])
	})

	t.Run("completed task with single metadata", func(t *testing.T) {
		result := parseTask("- [x] Fix bug project:urgent")
		require.Equal(t, "Fix bug", result.Description)
		require.True(t, result.Completed)
		require.Len(t, result.Metadata, 1)
		require.Equal(t, "urgent", result.Metadata["project"])
	})

	t.Run("task with multiple metadata", func(t *testing.T) {
		result := parseTask("- [x] Review the quarterly report project:work due:2025-08-10")
		require.Equal(t, "Review the quarterly report", result.Description)
		require.True(t, result.Completed)
		require.Len(t, result.Metadata, 2)
		require.Equal(t, "work", result.Metadata["project"])
		require.Equal(t, "2025-08-10", result.Metadata["due"])
	})

	t.Run("task with multiple metadata different order", func(t *testing.T) {
		result := parseTask("- [ ] Complete assignment due:2025-08-15 priority:B project:school")
		require.Equal(t, "Complete assignment", result.Description)
		require.False(t, result.Completed)
		require.Len(t, result.Metadata, 3)
		require.Equal(t, "2025-08-15", result.Metadata["due"])
		require.Equal(t, "B", result.Metadata["priority"])
		require.Equal(t, "school", result.Metadata["project"])
	})
}

func TestParseTask_DescriptionWithColons(t *testing.T) {
	t.Run("colon in description without metadata", func(t *testing.T) {
		result := parseTask("- [ ] Important: Call the supplier to confirm the order")
		require.Equal(t, "Important: Call the supplier to confirm the order", result.Description)
		require.False(t, result.Completed)
		require.Empty(t, result.Metadata)
	})

	t.Run("colon in description with metadata", func(t *testing.T) {
		result := parseTask("- [ ] Read chapter 3: The Empire priority:A")
		require.Equal(t, "Read chapter 3: The Empire", result.Description)
		require.False(t, result.Completed)
		require.Len(t, result.Metadata, 1)
		require.Equal(t, "A", result.Metadata["priority"])
	})

	t.Run("multiple colons in description with metadata", func(t *testing.T) {
		result := parseTask("- [x] Meeting: Q1 Review: Status update due:today")
		require.Equal(t, "Meeting: Q1 Review: Status update", result.Description)
		require.True(t, result.Completed)
		require.Len(t, result.Metadata, 1)
		require.Equal(t, "today", result.Metadata["due"])
	})

	t.Run("time format in description", func(t *testing.T) {
		result := parseTask("- [ ] Conference call at 3:30 PM priority:high")
		require.Equal(t, "Conference call at 3:30 PM", result.Description)
		require.False(t, result.Completed)
		require.Len(t, result.Metadata, 1)
		require.Equal(t, "high", result.Metadata["priority"])
	})
}

func TestParseTask_EdgeCases(t *testing.T) {
	t.Run("empty task content", func(t *testing.T) {
		result := parseTask("- [ ] ")
		require.Empty(t, result.Description)
		require.False(t, result.Completed)
		require.Empty(t, result.Metadata)
	})

	t.Run("only metadata no description", func(t *testing.T) {
		result := parseTask("- [ ] priority:A due:today")
		require.Empty(t, result.Description)
		require.False(t, result.Completed)
		require.Len(t, result.Metadata, 2)
		require.Equal(t, "A", result.Metadata["priority"])
		require.Equal(t, "today", result.Metadata["due"])
	})

	t.Run("malformed metadata (spaces in key)", func(t *testing.T) {
		result := parseTask("- [ ] Task content pri ority:A due:today")
		require.Equal(t, "Task content pri", result.Description)
		require.False(t, result.Completed)
		require.Len(t, result.Metadata, 2)
		require.Equal(t, "A", result.Metadata["ority"])
		require.Equal(t, "today", result.Metadata["due"])
	})

	t.Run("malformed metadata (spaces in value)", func(t *testing.T) {
		result := parseTask("- [ ] Task content priority:high level due:today")
		require.Equal(t, "Task content level", result.Description)
		require.False(t, result.Completed)
		require.Len(t, result.Metadata, 2)
		require.Equal(t, "high", result.Metadata["priority"])
		require.Equal(t, "today", result.Metadata["due"])
	})

	t.Run("metadata-like text in description", func(t *testing.T) {
		result := parseTask("- [ ] URL format is https://example.com priority:A")
		require.Equal(t, "URL format is https://example.com", result.Description)
		require.False(t, result.Completed)
		require.Len(t, result.Metadata, 1)
		require.Equal(t, "A", result.Metadata["priority"])
	})

	t.Run("colon at end of word", func(t *testing.T) {
		result := parseTask("- [ ] Note: this is important priority:high")
		require.Equal(t, "Note: this is important", result.Description)
		require.False(t, result.Completed)
		require.Len(t, result.Metadata, 1)
		require.Equal(t, "high", result.Metadata["priority"])
	})

	t.Run("empty key in metadata", func(t *testing.T) {
		result := parseTask("- [ ] Task content :value priority:A")
		require.Equal(t, "Task content :value", result.Description)
		require.False(t, result.Completed)
		require.Len(t, result.Metadata, 1)
		require.Equal(t, "A", result.Metadata["priority"])
	})

	t.Run("empty value in metadata", func(t *testing.T) {
		result := parseTask("- [ ] Task content key: priority:A")
		require.Equal(t, "Task content key:", result.Description)
		require.False(t, result.Completed)
		require.Len(t, result.Metadata, 1)
		require.Equal(t, "A", result.Metadata["priority"])
	})
}

func TestParseTask_InvalidFormat(t *testing.T) {
	t.Run("not a task format", func(t *testing.T) {
		result := parseTask("Just some text")
		require.Empty(t, result.Description)
		require.False(t, result.Completed)
		require.Empty(t, result.Metadata)
	})

	t.Run("missing checkbox", func(t *testing.T) {
		result := parseTask("- Task without checkbox")
		require.Empty(t, result.Description)
		require.False(t, result.Completed)
		require.Empty(t, result.Metadata)
	})

	t.Run("malformed checkbox", func(t *testing.T) {
		result := parseTask("- [y] Invalid checkbox")
		require.Empty(t, result.Description)
		require.False(t, result.Completed)
		require.Empty(t, result.Metadata)
	})

	t.Run("section header", func(t *testing.T) {
		result := parseTask("# This is a section")
		require.Empty(t, result.Description)
		require.False(t, result.Completed)
		require.Empty(t, result.Metadata)
	})

	t.Run("empty string", func(t *testing.T) {
		result := parseTask("")
		require.Empty(t, result.Description)
		require.False(t, result.Completed)
		require.Empty(t, result.Metadata)
	})
}

func TestParseTask_WhitespaceHandling(t *testing.T) {
	t.Run("extra spaces around task", func(t *testing.T) {
		result := parseTask("   - [ ] Task with spaces   ")
		require.Equal(t, "Task with spaces", result.Description)
		require.False(t, result.Completed)
		require.Empty(t, result.Metadata)
	})

	t.Run("extra spaces in task content", func(t *testing.T) {
		result := parseTask("- [ ]   Task   with   extra   spaces   priority:A   ")
		require.Equal(t, "Task with extra spaces", result.Description)
		require.False(t, result.Completed)
		require.Len(t, result.Metadata, 1)
		require.Equal(t, "A", result.Metadata["priority"])
	})

	t.Run("tabs and mixed whitespace", func(t *testing.T) {
		result := parseTask("-\t[ ]\tTask\twith\ttabs\tdue:today")
		require.Equal(t, "Task with tabs", result.Description)
		require.False(t, result.Completed)
		require.Len(t, result.Metadata, 1)
		require.Equal(t, "today", result.Metadata["due"])
	})
}

// Tests for parseTaskPrefix edge cases to improve coverage
func TestParseTaskPrefix_EdgeCases(t *testing.T) {
	t.Run("incomplete checkbox - ends at opening bracket", func(t *testing.T) {
		result := parseTask("- [")
		// This hits the p.pos >= p.len check when trying to read checkbox status
		require.Empty(t, result.Description)
		require.False(t, result.Completed)
		require.Empty(t, result.Metadata)
	})

	t.Run("malformed checkbox - no closing bracket", func(t *testing.T) {
		result := parseTask("- [ no closing bracket")
		// This hits the expect(']') failure case
		require.Empty(t, result.Description)
		require.False(t, result.Completed)
		require.Empty(t, result.Metadata)
	})

	t.Run("malformed checkbox but valid task prefix", func(t *testing.T) {
		result := parseTask("- [x")
		// The parser actually succeeds here - it parses 'x' as completed
		// but the expect(']') failure happens after parsing checkbox status
		require.Empty(t, result.Description)
		require.True(t, result.Completed) // 'x' was parsed
		require.Empty(t, result.Metadata)
	})
}

// Additional tests for parseQuotedString coverage - focus on working cases
func TestParseQuotedString_EdgeCases(t *testing.T) {
	t.Run("escape sequences", func(t *testing.T) {
		result := parseTask(`- [ ] Task with priority:"escaped \"quote\" inside"`)
		require.Equal(t, "Task with", result.Description)
		require.Equal(t, `escaped "quote" inside`, result.Metadata["priority"])
	})

	t.Run("escape characters", func(t *testing.T) {
		result := parseTask(`- [ ] Task content note:"line1\nline2\ttab\\backslash"`)
		require.Equal(t, "Task content", result.Description)
		require.Equal(t, "line1\nline2\ttab\\backslash", result.Metadata["note"])
	})

	t.Run("unterminated quote", func(t *testing.T) {
		result := parseTask(`- [ ] Task with unterminated quote status:"incomplete`)
		require.Equal(t, `Task with unterminated quote status:"incomplete`, result.Description)
		require.Empty(t, result.Metadata)
	})

	t.Run("escape sequence default case", func(t *testing.T) {
		// Test the default case in escape sequence handling (other than \", \\, \n, \t)
		result := parseTask(`- [ ] Task with note:"text\r\x41"`)
		require.Equal(t, "Task with", result.Description)
		require.Equal(t, "textrx41", result.Metadata["note"]) // \r and \x41 are treated as literal characters
	})
}

// Additional tests for parseContent coverage (whitespace handling)
func TestParseContent_EdgeCases(t *testing.T) {
	t.Run("task ending with whitespace", func(t *testing.T) {
		// Test whitespace handling at end of content
		result := parseTask("- [ ] content   ")
		require.Equal(t, "content", result.Description)
		require.False(t, result.Completed)
		require.Empty(t, result.Metadata)
	})

	t.Run("empty task content with spaces", func(t *testing.T) {
		// Edge case: only whitespace after checkbox
		result := parseTask("- [ ]     ")
		require.Empty(t, result.Description)
		require.False(t, result.Completed)
		require.Empty(t, result.Metadata)
	})

	t.Run("task with special characters", func(t *testing.T) {
		// Special characters are included in words by parseWord()
		result := parseTask("- [ ] content @ & $ #")
		require.Equal(t, "content @ & $ #", result.Description)
		require.False(t, result.Completed)
		require.Empty(t, result.Metadata)
	})

}

func TestParseTask_RealWorldExamples(t *testing.T) {
	t.Run("bug report task", func(t *testing.T) {
		result := parseTask("- [ ] Fix issue #123: Button not responding on mobile devices priority:high due:2025-08-05 project:mobile")
		require.Equal(t, "Fix issue #123: Button not responding on mobile devices", result.Description)
		require.False(t, result.Completed)
		require.Len(t, result.Metadata, 3)
		require.Equal(t, "high", result.Metadata["priority"])
		require.Equal(t, "2025-08-05", result.Metadata["due"])
		require.Equal(t, "mobile", result.Metadata["project"])
	})

	t.Run("meeting task", func(t *testing.T) {
		result := parseTask("- [x] Attend daily standup: Discuss sprint progress and blockers due:today type:meeting")
		require.Equal(t, "Attend daily standup: Discuss sprint progress and blockers", result.Description)
		require.True(t, result.Completed)
		require.Len(t, result.Metadata, 2)
		require.Equal(t, "today", result.Metadata["due"])
		require.Equal(t, "meeting", result.Metadata["type"])
	})

	t.Run("research task", func(t *testing.T) {
		result := parseTask("- [ ] Research OAuth 2.0 vs JWT: Compare security implications estimate:4h priority:medium")
		require.Equal(t, "Research OAuth 2.0 vs JWT: Compare security implications", result.Description)
		require.False(t, result.Completed)
		require.Len(t, result.Metadata, 2)
		require.Equal(t, "4h", result.Metadata["estimate"])
		require.Equal(t, "medium", result.Metadata["priority"])
	})

	t.Run("complex description with URLs", func(t *testing.T) {
		result := parseTask("- [ ] Review API documentation at https://api.example.com/docs priority:low")
		require.Equal(t, "Review API documentation at https://api.example.com/docs", result.Description)
		require.False(t, result.Completed)
		require.Len(t, result.Metadata, 1)
		require.Equal(t, "low", result.Metadata["priority"])
	})

	t.Run("quoted metadata values", func(t *testing.T) {
		result := parseTask(`- [ ] Task with quoted metadata priority:"high priority" status:"in progress"`)
		require.Equal(t, "Task with quoted metadata", result.Description)
		require.False(t, result.Completed)
		require.Len(t, result.Metadata, 2)
		require.Equal(t, "high priority", result.Metadata["priority"])
		require.Equal(t, "in progress", result.Metadata["status"])
	})
}
