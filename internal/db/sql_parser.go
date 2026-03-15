package db

import "strings"

// extractUpSQL extracts the SQL between '-- migrate:up' and '-- migrate:down' markers
func extractUpSQL(content string) string {
	lines := strings.Split(content, "\n")
	var upLines []string
	inUpBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "-- migrate:up" {
			inUpBlock = true
			continue
		}

		if trimmed == "-- migrate:down" {
			break
		}

		if inUpBlock && trimmed != "" && !strings.HasPrefix(trimmed, "--") {
			upLines = append(upLines, line)
		}
	}

	return strings.TrimSpace(strings.Join(upLines, "\n"))
}

// scanDollarTag checks if runes[i] starts a dollar-quoted tag ($$ or $tag$).
// Returns the full tag string and true if found, or ("", false) otherwise.
func scanDollarTag(runes []rune, i int) (string, bool) {
	if i >= len(runes) || runes[i] != '$' {
		return "", false
	}
	// Look for closing '$' — tag content is optional ($$) or an identifier.
	j := i + 1
	for j < len(runes) && isSQLIdentifierRune(runes[j]) {
		j++
	}
	if j < len(runes) && runes[j] == '$' {
		tag := string(runes[i : j+1]) // includes both '$' delimiters
		return tag, true
	}
	return "", false
}

// splitStatements splits SQL by semicolon, handling strings, dollar-quoted
// blocks, inline comments, and CREATE TRIGGER bodies properly.
func splitStatements(sql string) []string {
	// First, remove inline comments (-- to end of line)
	sql = removeInlineComments(sql)

	var statements []string
	var current strings.Builder
	inString := false
	var stringChar rune
	inDollarQuote := false
	dollarTag := ""
	inCreateStatement := false
	inTriggerStatement := false
	triggerBodyDepth := 0
	triggerBodyClosed := false
	var i int
	runes := []rune(sql)

	for i < len(runes) {
		char := runes[i]

		// Inside a dollar-quoted string — look for the closing tag.
		if inDollarQuote {
			if char == '$' {
				if tag, ok := scanDollarTag(runes, i); ok && tag == dollarTag {
					current.WriteString(tag)
					i += len([]rune(tag))
					inDollarQuote = false
					dollarTag = ""
					continue
				}
			}
			current.WriteRune(char)
			i++
			continue
		}

		if !inString && char == '$' {
			if tag, ok := scanDollarTag(runes, i); ok {
				inDollarQuote = true
				dollarTag = tag
				current.WriteString(tag)
				i += len([]rune(tag))
				continue
			}
		}

		if !inString && (char == '\'' || char == '"') {
			inString = true
			stringChar = char
			current.WriteRune(char)
		} else if !inString && isSQLIdentifierRune(char) {
			start := i
			for i < len(runes) && isSQLIdentifierRune(runes[i]) {
				i++
			}

			word := strings.ToUpper(string(runes[start:i]))
			current.WriteString(string(runes[start:i]))

			if !inTriggerStatement {
				if word == "CREATE" {
					inCreateStatement = true
				} else if inCreateStatement && word == "TRIGGER" {
					inTriggerStatement = true
					triggerBodyDepth = 0
					triggerBodyClosed = false
					inCreateStatement = false
				} else if inCreateStatement && word != "TEMPORARY" && word != "OR" && word != "REPLACE" {
					inCreateStatement = false
				}
			}

			if inTriggerStatement {
				switch word {
				case "BEGIN":
					triggerBodyDepth++
				case "END":
					if triggerBodyDepth > 0 {
						triggerBodyDepth--
					}
					if triggerBodyDepth == 0 {
						triggerBodyClosed = true
					}
				}
			}

			continue
		} else if inString && char == stringChar {
			if i+1 < len(runes) && runes[i+1] == stringChar {
				// Escaped quote
				current.WriteRune(char)
				current.WriteRune(char)
				i++
			} else {
				inString = false
				current.WriteRune(char)
			}
		} else if !inString && char == ';' {
			if inTriggerStatement && !triggerBodyClosed {
				current.WriteRune(char)
			} else {
				statements = append(statements, current.String())
				current.Reset()
				inCreateStatement = false
				inTriggerStatement = false
				triggerBodyDepth = 0
				triggerBodyClosed = false
			}
		} else {
			current.WriteRune(char)
		}

		i++
	}

	// Add any remaining statement
	if current.Len() > 0 {
		statements = append(statements, current.String())
	}

	return statements
}

func isSQLIdentifierRune(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '_'
}

// removeInlineComments removes SQL inline comments (-- to end of line) while
// preserving strings (single/double-quoted and dollar-quoted).
func removeInlineComments(sql string) string {
	var result strings.Builder
	runes := []rune(sql)
	inString := false
	var stringChar rune
	inDollarQuote := false
	dollarTag := ""

	for i := 0; i < len(runes); i++ {
		char := runes[i]

		// Inside a dollar-quoted string — look for the closing tag.
		if inDollarQuote {
			if char == '$' {
				if tag, ok := scanDollarTag(runes, i); ok && tag == dollarTag {
					result.WriteString(tag)
					i += len(tag) - 1 // -1 because loop increments
					inDollarQuote = false
					dollarTag = ""
					continue
				}
			}
			result.WriteRune(char)
			continue
		}

		// Check for opening dollar-quote.
		if !inString && char == '$' {
			if tag, ok := scanDollarTag(runes, i); ok {
				inDollarQuote = true
				dollarTag = tag
				result.WriteString(tag)
				i += len(tag) - 1 // -1 because loop increments
				continue
			}
		}

		// Handle string delimiters
		if !inString && (char == '\'' || char == '"') {
			inString = true
			stringChar = char
			result.WriteRune(char)
		} else if inString && char == stringChar {
			if i+1 < len(runes) && runes[i+1] == stringChar {
				// Escaped quote
				result.WriteRune(char)
				result.WriteRune(runes[i+1])
				i++
			} else {
				inString = false
				result.WriteRune(char)
			}
		} else if !inString && char == '-' && i+1 < len(runes) && runes[i+1] == '-' {
			// Found inline comment, skip until end of line (but don't skip the newline itself)
			i += 2
			for i < len(runes) && runes[i] != '\n' {
				i++
			}
			// Don't increment i here, so the newline will be written in the next iteration
			i--
		} else {
			result.WriteRune(char)
		}
	}

	return result.String()
}
