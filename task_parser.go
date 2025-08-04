package main

import (
	"strings"
	"unicode"
)

// TaskParser represents a recursive descent parser for task lines
type TaskParser struct {
	input string
	pos   int
	len   int
}

// ParsedTask represents a parsed task with metadata
type ParsedTask struct {
	Description string            // Clean task description without metadata
	Completed   bool              // Task completion status
	Metadata    map[string]string // Key-value metadata pairs
}

// parseTask parses a single task line and extracts description, completion status, and metadata
func parseTask(taskLine string) ParsedTask {
	parser := &TaskParser{
		input: strings.TrimSpace(taskLine),
		pos:   0,
	}
	parser.len = len(parser.input)

	result := ParsedTask{
		Description: "",
		Completed:   false,
		Metadata:    make(map[string]string),
	}

	// Parse task prefix: "- [x]" or "- [ ]"
	if !parser.parseTaskPrefix(&result) {
		return result // Invalid task format
	}

	// Parse the content (description + metadata)
	parser.parseContent(&result)

	return result
}

// parseTaskPrefix parses "- [x]" or "- [ ]" and sets completion status
func (p *TaskParser) parseTaskPrefix(result *ParsedTask) bool {
	p.skipWhitespace()

	if !p.expect('-') {
		return false
	}

	p.skipWhitespace()

	if !p.expect('[') {
		return false
	}

	// Parse checkbox status
	if p.pos >= p.len {
		return false
	}

	ch := p.input[p.pos]
	switch ch {
	case 'x', 'X':
		result.Completed = true
		p.pos++
	case ' ':
		result.Completed = false
		p.pos++
	default:
		return false
	}

	if !p.expect(']') {
		return false
	}

	p.skipWhitespace()

	return true
}

// parseContent parses the task content (description + metadata)
func (p *TaskParser) parseContent(result *ParsedTask) {
	var tokens []string

	// Parse all tokens (words and metadata)
	for p.pos < p.len {
		p.skipWhitespace()

		// Try to parse metadata key:value pair
		if key, value, ok := p.parseMetadata(); ok {
			result.Metadata[key] = value
			continue
		}

		// Parse regular word (will always succeed for non-whitespace characters)
		word := p.parseWord()
		tokens = append(tokens, word)
	}

	// Join all non-metadata tokens as description
	result.Description = strings.TrimSpace(strings.Join(tokens, " "))
}

// parseMetadata tries to parse a key:value or key:"quoted value" pair
func (p *TaskParser) parseMetadata() (key, value string, ok bool) {
	start := p.pos

	// Parse key (must start with letter)
	key = p.parseIdentifier()
	if key == "" || !unicode.IsLetter(rune(key[0])) {
		p.pos = start
		return "", "", false
	}

	// Expect ":"
	if !p.expect(':') {
		p.pos = start
		return "", "", false
	}

	// Parse value (can be quoted or unquoted)
	if p.pos < p.len && p.input[p.pos] == '"' {
		// Parse quoted value
		value = p.parseQuotedString()
		if value == "" {
			p.pos = start
			return "", "", false
		}
	} else {
		// Parse unquoted value
		value = p.parseIdentifier()
		if value == "" {
			p.pos = start
			return "", "", false
		}
	}

	return key, value, true
}

// parseIdentifier parses an identifier (letters, digits, underscore, hyphen, dot)
func (p *TaskParser) parseIdentifier() string {
	start := p.pos

	for p.pos < p.len {
		ch := rune(p.input[p.pos])
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' && ch != '-' && ch != '.' {
			break
		}
		p.pos++
	}

	if p.pos == start {
		return ""
	}

	return p.input[start:p.pos]
}

// parseQuotedString parses a double-quoted string with escape support
func (p *TaskParser) parseQuotedString() string {
	// Consume opening quote (guaranteed by caller)
	p.pos++

	var result strings.Builder

	for p.pos < p.len {
		ch := p.input[p.pos]

		if ch == '"' {
			// End of string
			p.pos++
			return result.String()
		}

		if ch == '\\' && p.pos+1 < p.len {
			// Escape sequence
			p.pos++
			next := p.input[p.pos]
			switch next {
			case '"':
				result.WriteByte('"')
			case '\\':
				result.WriteByte('\\')
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			default:
				result.WriteByte(next)
			}
			p.pos++
		} else {
			result.WriteByte(ch)
			p.pos++
		}
	}

	// Unterminated string
	return ""
}

// parseWord parses a regular word (non-metadata)
func (p *TaskParser) parseWord() string {
	start := p.pos

	for p.pos < p.len {
		ch := p.input[p.pos]
		if ch == ' ' || ch == '\t' {
			break
		}
		p.pos++
	}

	return p.input[start:p.pos]
}

// expect checks for a specific character and advances if found
func (p *TaskParser) expect(expected byte) bool {
	if p.pos >= p.len || p.input[p.pos] != expected {
		return false
	}
	p.pos++
	return true
}

// skipWhitespace skips spaces and tabs
func (p *TaskParser) skipWhitespace() {
	for p.pos < p.len && (p.input[p.pos] == ' ' || p.input[p.pos] == '\t') {
		p.pos++
	}
}
