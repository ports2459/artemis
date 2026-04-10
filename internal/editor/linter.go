package editor

import (
	"fmt"
	"strings"
)

// LintResult is a single lint diagnostic.
type LintResult struct {
	Line     int    // 1-indexed
	Col      int    // 1-indexed
	Severity int    // 1=Error, 2=Warning, 3=Info, 4=Hint
	Message  string
}

// LintCSharp runs basic lint checks on C# source code.
func LintCSharp(content string) []LintResult {
	lines := strings.Split(content, "\n")
	var results []LintResult

	braceStack := 0
	parenStack := 0
	inBlockComment := false

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// Track block comments
		if inBlockComment {
			if strings.Contains(line, "*/") {
				inBlockComment = false
			}
			continue
		}
		if strings.Contains(trimmed, "/*") {
			if !strings.Contains(trimmed, "*/") {
				inBlockComment = true
			}
			continue
		}

		// Skip line comments
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Skip preprocessor directives
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// 1. Trailing whitespace
		if len(line) > 0 && (line[len(line)-1] == ' ' || line[len(line)-1] == '\t') {
			results = append(results, LintResult{
				Line: lineNum, Col: len(line), Severity: 4,
				Message: "Trailing whitespace",
			})
		}

		// 2. Mixed tabs and spaces
		leadingWhitespace := ""
		for _, ch := range line {
			if ch == ' ' || ch == '\t' {
				leadingWhitespace += string(ch)
			} else {
				break
			}
		}
		if strings.Contains(leadingWhitespace, "\t") && strings.Contains(leadingWhitespace, " ") {
			results = append(results, LintResult{
				Line: lineNum, Col: 1, Severity: 3,
				Message: "Mixed tabs and spaces in indentation",
			})
		}

		// 3. Track brace matching
		for _, ch := range trimmed {
			if ch == '{' {
				braceStack++
			} else if ch == '}' {
				braceStack--
			} else if ch == '(' {
				parenStack++
			} else if ch == ')' {
				parenStack--
			}
		}
		if braceStack < 0 {
			results = append(results, LintResult{
				Line: lineNum, Col: 1, Severity: 1,
				Message: "Unexpected closing brace '}'",
			})
			braceStack = 0
		}
		if parenStack < 0 {
			results = append(results, LintResult{
				Line: lineNum, Col: 1, Severity: 1,
				Message: "Unexpected closing parenthesis ')'",
			})
			parenStack = 0
		}

		// 4. Missing semicolon on statements (heuristic)
		// Lines that look like statements but don't end with ; { } , or )
		if shouldHaveSemicolon(trimmed) {
			results = append(results, LintResult{
				Line: lineNum, Col: len(trimmed), Severity: 2,
				Message: "Statement may be missing a semicolon",
			})
		}

		// 5. Line too long (120 chars)
		if len(line) > 120 {
			results = append(results, LintResult{
				Line: lineNum, Col: 121, Severity: 4,
				Message: "Line exceeds 120 characters",
			})
		}

		// 6. TODO/FIXME/HACK comments
		upper := strings.ToUpper(trimmed)
		if idx := strings.Index(upper, "TODO"); idx >= 0 {
			results = append(results, LintResult{
				Line: lineNum, Col: idx + 1, Severity: 3,
				Message: "TODO comment found",
			})
		}
		if idx := strings.Index(upper, "FIXME"); idx >= 0 {
			results = append(results, LintResult{
				Line: lineNum, Col: idx + 1, Severity: 2,
				Message: "FIXME comment found",
			})
		}
	}

	// Check unclosed braces at end of file
	if braceStack > 0 {
		results = append(results, LintResult{
			Line: len(lines), Col: 1, Severity: 1,
			Message: fmt.Sprintf("%d unclosed brace(s) '{'", braceStack),
		})
	}
	if parenStack > 0 {
		results = append(results, LintResult{
			Line: len(lines), Col: 1, Severity: 1,
			Message: fmt.Sprintf("%d unclosed parenthesis(es) '('", parenStack),
		})
	}

	return results
}

// shouldHaveSemicolon returns true if a line looks like a C# statement
// that should end with a semicolon but doesn't.
func shouldHaveSemicolon(trimmed string) bool {
	// Lines that definitely don't need semicolons
	if trimmed == "" || trimmed == "{" || trimmed == "}" {
		return false
	}
	last := trimmed[len(trimmed)-1]
	if last == ';' || last == '{' || last == '}' || last == ',' || last == '(' || last == ')' {
		return false
	}

	// Skip attributes, comments, preprocessor, namespace/class/if/else/etc
	if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "//") ||
		strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") ||
		strings.HasPrefix(trimmed, "#") {
		return false
	}

	// Skip declarations that open blocks
	keywords := []string{"namespace ", "class ", "struct ", "interface ", "enum ",
		"if ", "if(", "else", "for ", "for(", "foreach ", "foreach(",
		"while ", "while(", "switch ", "switch(", "try", "catch", "finally",
		"do", "using ", "lock ", "lock("}
	for _, kw := range keywords {
		if strings.HasPrefix(trimmed, kw) {
			return false
		}
	}

	// Skip lines ending with => (lambda)
	if strings.HasSuffix(trimmed, "=>") {
		return false
	}

	// If the line starts with an access modifier or 'static', check whether
	// a block-opening keyword follows (class, struct, interface, enum, etc.).
	// Those are declarations that don't need semicolons.
	blockKeywords := []string{"class ", "struct ", "interface ", "enum ",
		"namespace ", "void ", "async "}
	modifiers := []string{"public ", "private ", "protected ", "internal ",
		"static ", "abstract ", "sealed ", "partial ", "override ", "virtual "}
	rest := trimmed
	for {
		matched := false
		for _, mod := range modifiers {
			if strings.HasPrefix(rest, mod) {
				rest = rest[len(mod):]
				matched = true
				break
			}
		}
		if !matched {
			break
		}
	}
	for _, bk := range blockKeywords {
		if strings.HasPrefix(rest, bk) {
			return false
		}
	}

	// Lines starting with these are likely statements needing semicolons
	starters := []string{"return ", "var ", "int ", "string ", "bool ", "float ",
		"double ", "public ", "private ", "protected ", "internal ",
		"static ", "readonly ", "const ", "throw ", "await ", "yield "}
	for _, s := range starters {
		if strings.HasPrefix(trimmed, s) && last != ')' {
			return true
		}
	}

	// Method calls / assignments (contain = or . and don't end right)
	if (strings.Contains(trimmed, "=") || strings.Contains(trimmed, ".")) &&
		!strings.HasPrefix(trimmed, "public ") && !strings.HasPrefix(trimmed, "private ") &&
		last != ')' && !strings.HasSuffix(trimmed, "=>") {
		// Check it looks like a statement
		if !strings.HasPrefix(trimmed, "case ") && !strings.HasPrefix(trimmed, "default:") {
			return true
		}
	}

	return false
}

// Lint runs the appropriate linter based on language.
func Lint(content string, lang Language) []LintResult {
	switch lang {
	case LangCSharp:
		return LintCSharp(content)
	default:
		return nil
	}
}
