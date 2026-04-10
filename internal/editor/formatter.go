package editor

import "strings"

const indentUnit = "    "

// FormatCSharp reformats C# source with consistent 4-space indentation.
func FormatCSharp(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	depth := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			result = append(result, "")
			continue
		}

		// Count leading close braces to dedent this line
		leadingClose := 0
		for _, ch := range trimmed {
			if ch == '}' {
				leadingClose++
			} else {
				break
			}
		}

		// This line's indent = current depth minus leading close braces
		lineDepth := depth - leadingClose
		if lineDepth < 0 {
			lineDepth = 0
		}

		result = append(result, strings.Repeat(indentUnit, lineDepth)+trimmed)

		// Update depth for next line based on ALL braces in this line (outside strings/comments)
		depth += netBracesInLine(trimmed)
		if depth < 0 {
			depth = 0
		}
	}

	return strings.Join(result, "\n")
}

// netBracesInLine counts { minus } in a line, skipping braces inside
// string literals and comments.
func netBracesInLine(line string) int {
	net := 0
	inString := false
	inChar := false
	var stringChar rune
	escaped := false

	for i, ch := range line {
		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' && (inString || inChar) {
			escaped = true
			continue
		}

		// Line comment — skip rest
		if !inString && !inChar && ch == '/' && i+1 < len(line) && line[i+1] == '/' {
			break
		}

		// String/char literal tracking
		if !inString && !inChar {
			if ch == '"' {
				inString = true
				stringChar = '"'
				continue
			}
			if ch == '\'' {
				inChar = true
				stringChar = '\''
				continue
			}
		} else if inString && ch == stringChar {
			inString = false
			continue
		} else if inChar && ch == stringChar {
			inChar = false
			continue
		}

		if !inString && !inChar {
			if ch == '{' {
				net++
			} else if ch == '}' {
				net--
			}
		}
	}
	return net
}

// Format runs the appropriate formatter based on language.
func Format(content string, lang Language) string {
	switch lang {
	case LangCSharp:
		return FormatCSharp(content)
	default:
		return content
	}
}
