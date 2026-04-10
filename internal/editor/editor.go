package editor

import (
	"fmt"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eden/artemis/internal/app"
)

// Editor is a custom text editor widget with syntax highlighting and cursor rendering.
type Editor struct {
	lines     []string
	cursorRow int
	cursorCol int
	scroll    int // vertical scroll offset
	width     int
	height    int
	focused   bool
	language  Language
	tabSize   int
}

// NewEditor creates a new Editor with the given content and language.
func NewEditor(content string, lang Language, tabSize int) *Editor {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	return &Editor{
		lines:    lines,
		language: lang,
		tabSize:  tabSize,
	}
}

// Value returns the full content as a single string.
func (e *Editor) Value() string {
	return strings.Join(e.lines, "\n")
}

// SetValue replaces the editor content and clamps the cursor.
func (e *Editor) SetValue(content string) {
	e.lines = strings.Split(content, "\n")
	if len(e.lines) == 0 {
		e.lines = []string{""}
	}
	if e.cursorRow >= len(e.lines) {
		e.cursorRow = len(e.lines) - 1
	}
	if e.cursorCol > len(e.lines[e.cursorRow]) {
		e.cursorCol = len(e.lines[e.cursorRow])
	}
}

// Line returns the current cursor row (0-indexed).
func (e *Editor) Line() int { return e.cursorRow }

// Focused returns whether the editor has focus.
func (e *Editor) Focused() bool { return e.focused }

// Focus gives focus to the editor.
func (e *Editor) Focus() {
	e.focused = true
}

// Blur removes focus from the editor.
func (e *Editor) Blur() {
	e.focused = false
}

// SetSize sets the visible width and height of the editor.
func (e *Editor) SetSize(width, height int) {
	e.width = width
	e.height = height
}

// Update processes a tea.Msg and returns a command.
func (e *Editor) Update(msg tea.Msg) tea.Cmd {
	if !e.focused {
		return nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		e.handleKey(msg)
	}
	return nil
}

func (e *Editor) handleKey(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyUp:
		e.moveCursorUp()
	case tea.KeyDown:
		e.moveCursorDown()
	case tea.KeyLeft:
		if msg.Alt {
			e.wordLeft()
		} else {
			e.moveCursorLeft()
		}
	case tea.KeyRight:
		if msg.Alt {
			e.wordRight()
		} else {
			e.moveCursorRight()
		}
	case tea.KeyHome:
		e.cursorCol = 0
	case tea.KeyEnd:
		e.cursorCol = len(e.lines[e.cursorRow])
	case tea.KeyPgUp:
		for i := 0; i < e.height-1; i++ {
			e.moveCursorUp()
		}
	case tea.KeyPgDown:
		for i := 0; i < e.height-1; i++ {
			e.moveCursorDown()
		}
	case tea.KeyBackspace:
		e.backspace()
	case tea.KeyDelete:
		e.deleteChar()
	case tea.KeyEnter:
		e.insertNewline()
	case tea.KeySpace:
		e.insertRune(' ')
	case tea.KeyTab:
		e.insertTab()
	case tea.KeyShiftTab:
		e.outdent()
	default:
		// Check for ctrl+arrow combinations
		s := msg.String()
		if s == "ctrl+left" {
			e.wordLeft()
			return
		}
		if s == "ctrl+right" {
			e.wordRight()
			return
		}

		// Insert printable runes
		if msg.Type == tea.KeyRunes {
			for _, r := range msg.Runes {
				e.insertRune(r)
			}
		}
	}
}

func (e *Editor) moveCursorUp() {
	if e.cursorRow > 0 {
		e.cursorRow--
		if e.cursorCol > len(e.lines[e.cursorRow]) {
			e.cursorCol = len(e.lines[e.cursorRow])
		}
	}
}

func (e *Editor) moveCursorDown() {
	if e.cursorRow < len(e.lines)-1 {
		e.cursorRow++
		if e.cursorCol > len(e.lines[e.cursorRow]) {
			e.cursorCol = len(e.lines[e.cursorRow])
		}
	}
}

func (e *Editor) moveCursorLeft() {
	if e.cursorCol > 0 {
		e.cursorCol--
	} else if e.cursorRow > 0 {
		e.cursorRow--
		e.cursorCol = len(e.lines[e.cursorRow])
	}
}

func (e *Editor) moveCursorRight() {
	line := e.lines[e.cursorRow]
	if e.cursorCol < len(line) {
		e.cursorCol++
	} else if e.cursorRow < len(e.lines)-1 {
		e.cursorRow++
		e.cursorCol = 0
	}
}

func (e *Editor) wordLeft() {
	if e.cursorCol == 0 {
		e.moveCursorLeft()
		return
	}
	line := e.lines[e.cursorRow]
	col := e.cursorCol - 1
	// Skip whitespace
	for col > 0 && unicode.IsSpace(rune(line[col])) {
		col--
	}
	// Skip word characters
	for col > 0 && !unicode.IsSpace(rune(line[col-1])) {
		col--
	}
	e.cursorCol = col
}

func (e *Editor) wordRight() {
	line := e.lines[e.cursorRow]
	if e.cursorCol >= len(line) {
		e.moveCursorRight()
		return
	}
	col := e.cursorCol
	// Skip word characters
	for col < len(line) && !unicode.IsSpace(rune(line[col])) {
		col++
	}
	// Skip whitespace
	for col < len(line) && unicode.IsSpace(rune(line[col])) {
		col++
	}
	e.cursorCol = col
}

func (e *Editor) insertRune(r rune) {
	line := e.lines[e.cursorRow]
	e.lines[e.cursorRow] = line[:e.cursorCol] + string(r) + line[e.cursorCol:]
	e.cursorCol++
}

func (e *Editor) insertNewline() {
	line := e.lines[e.cursorRow]
	before := line[:e.cursorCol]
	after := line[e.cursorCol:]

	// Auto-indent: copy leading whitespace from current line
	indent := ""
	for _, ch := range before {
		if ch == ' ' || ch == '\t' {
			indent += string(ch)
		} else {
			break
		}
	}

	e.lines[e.cursorRow] = before
	// Insert new line after current
	newLines := make([]string, len(e.lines)+1)
	copy(newLines, e.lines[:e.cursorRow+1])
	newLines[e.cursorRow+1] = indent + after
	copy(newLines[e.cursorRow+2:], e.lines[e.cursorRow+1:])
	e.lines = newLines

	e.cursorRow++
	e.cursorCol = len(indent)
}

func (e *Editor) insertTab() {
	spaces := strings.Repeat(" ", e.tabSize)
	line := e.lines[e.cursorRow]
	e.lines[e.cursorRow] = line[:e.cursorCol] + spaces + line[e.cursorCol:]
	e.cursorCol += e.tabSize
}

func (e *Editor) outdent() {
	line := e.lines[e.cursorRow]
	removed := 0
	for i := 0; i < e.tabSize && i < len(line) && line[i] == ' '; i++ {
		removed++
	}
	if removed > 0 {
		e.lines[e.cursorRow] = line[removed:]
		e.cursorCol -= removed
		if e.cursorCol < 0 {
			e.cursorCol = 0
		}
	}
}

func (e *Editor) backspace() {
	if e.cursorCol > 0 {
		line := e.lines[e.cursorRow]
		e.lines[e.cursorRow] = line[:e.cursorCol-1] + line[e.cursorCol:]
		e.cursorCol--
	} else if e.cursorRow > 0 {
		// Merge with previous line
		prevLine := e.lines[e.cursorRow-1]
		e.cursorCol = len(prevLine)
		e.lines[e.cursorRow-1] = prevLine + e.lines[e.cursorRow]
		e.lines = append(e.lines[:e.cursorRow], e.lines[e.cursorRow+1:]...)
		e.cursorRow--
	}
}

func (e *Editor) deleteChar() {
	line := e.lines[e.cursorRow]
	if e.cursorCol < len(line) {
		e.lines[e.cursorRow] = line[:e.cursorCol] + line[e.cursorCol+1:]
	} else if e.cursorRow < len(e.lines)-1 {
		// Merge with next line
		e.lines[e.cursorRow] = line + e.lines[e.cursorRow+1]
		e.lines = append(e.lines[:e.cursorRow+1], e.lines[e.cursorRow+2:]...)
	}
}

// View renders the editor content with line numbers, syntax highlighting, and cursor.
func (e *Editor) View() string {
	// Ensure scroll keeps cursor visible
	if e.cursorRow < e.scroll {
		e.scroll = e.cursorRow
	}
	if e.height > 0 && e.cursorRow >= e.scroll+e.height {
		e.scroll = e.cursorRow - e.height + 1
	}

	lineNumWidth := len(fmt.Sprintf("%d", len(e.lines)))
	if lineNumWidth < 2 {
		lineNumWidth = 2
	}

	lineNumStyle := lipgloss.NewStyle().Foreground(app.ColorSurface2)
	cursorLineNumStyle := lipgloss.NewStyle().Foreground(app.ColorMauve).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(app.ColorSurface2)

	contentWidth := e.width - lineNumWidth - 1
	if contentWidth < 1 {
		contentWidth = 1
	}

	viewHeight := e.height
	if viewHeight <= 0 {
		viewHeight = len(e.lines)
	}

	var b strings.Builder
	for i := e.scroll; i < len(e.lines) && i < e.scroll+viewHeight; i++ {
		if i > e.scroll {
			b.WriteString("\n")
		}

		// Line number
		numStr := fmt.Sprintf("%*d", lineNumWidth, i+1)
		if i == e.cursorRow {
			b.WriteString(cursorLineNumStyle.Render(numStr))
		} else {
			b.WriteString(lineNumStyle.Render(numStr))
		}
		b.WriteString(" ")

		// Line content with syntax highlighting
		line := e.lines[i]
		highlighted := HighlightLine(line, e.language)

		if i == e.cursorRow && e.focused {
			// Render cursor line with background and cursor character
			rendered := e.renderCursorLine(line, contentWidth)
			b.WriteString(rendered)
		} else if i == e.cursorRow {
			b.WriteString(highlighted)
		} else {
			b.WriteString(highlighted)
		}
	}

	// Fill empty lines if height is set
	rendered := 0
	for i := e.scroll; i < len(e.lines) && i < e.scroll+viewHeight; i++ {
		rendered++
	}
	for i := rendered; i < viewHeight; i++ {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(strings.Repeat(" ", lineNumWidth) + " ~"))
	}

	return b.String()
}

// renderCursorLine renders a line with syntax highlighting AND a visible cursor.
// The cursor is shown as a reverse-video block character at cursorCol.
func (e *Editor) renderCursorLine(line string, width int) string {
	// Highlight the entire line as one unit so comment/string detection isn't broken
	highlighted := HighlightLine(line, e.language)
	// Inject cursor at the right visual position by walking past ANSI escapes
	return insertCursorAt(highlighted, e.cursorCol)
}

// insertCursorAt walks a styled string (with ANSI escapes), counting visible
// characters, and wraps the character at position col with reverse video.
func insertCursorAt(styled string, col int) string {
	var result strings.Builder
	visiblePos := 0
	i := 0
	cursorInserted := false

	for i < len(styled) {
		if styled[i] == '\x1b' {
			// ANSI escape sequence — copy through without counting as visible
			j := i
			for j < len(styled) && styled[j] != 'm' {
				j++
			}
			if j < len(styled) {
				j++ // include the 'm'
			}
			result.WriteString(styled[i:j])
			i = j
			continue
		}

		if visiblePos == col && !cursorInserted {
			result.WriteString("\x1b[7m")  // reverse video on
			result.WriteByte(styled[i])
			result.WriteString("\x1b[27m") // reverse video off
			cursorInserted = true
		} else {
			result.WriteByte(styled[i])
		}
		visiblePos++
		i++
	}

	// Cursor at end of line
	if !cursorInserted {
		result.WriteString("\x1b[7m \x1b[27m")
	}

	return result.String()
}
