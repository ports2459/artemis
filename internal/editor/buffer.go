package editor

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Buffer represents an open file editor buffer backed by a custom Editor.
type Buffer struct {
	filePath    string
	language    Language
	editor      *Editor
	dirty       bool
	undoStack   []string
	redoStack   []string
	lastContent string
	tabSize     int
	width       int
	height      int
}

// NewBuffer creates a new Buffer for the given path and content.
func NewBuffer(path, content string, tabSize int) *Buffer {
	lang := DetectLanguage(path)
	ed := NewEditor(content, lang, tabSize)

	return &Buffer{
		filePath:    path,
		language:    lang,
		editor:      ed,
		dirty:       false,
		lastContent: content,
		tabSize:     tabSize,
	}
}

// FilePath returns the file path of the buffer.
func (b *Buffer) FilePath() string {
	return b.filePath
}

// FileName returns just the filename component.
func (b *Buffer) FileName() string {
	return filepath.Base(b.filePath)
}

// Language returns the detected language.
func (b *Buffer) Language() Language {
	return b.language
}

// Dirty returns whether the buffer has unsaved changes.
func (b *Buffer) Dirty() bool {
	return b.dirty
}

// MarkDirty marks the buffer as having unsaved changes.
func (b *Buffer) MarkDirty() {
	b.dirty = true
}

// MarkClean marks the buffer as saved/clean.
func (b *Buffer) MarkClean() {
	b.dirty = false
}

// Content returns the current editor content.
func (b *Buffer) Content() string {
	return b.editor.Value()
}

// SetContent replaces the buffer content (used by formatter).
func (b *Buffer) SetContent(content string) {
	b.editor.SetValue(content)
}

// SetSize sets the editor dimensions.
func (b *Buffer) SetSize(w, h int) {
	b.width = w
	b.height = h
	b.editor.SetSize(w, h)
}

// Focus gives focus to the editor and returns nil (no blink command needed).
func (b *Buffer) Focus() tea.Cmd {
	b.editor.Focus()
	return nil
}

// Blur removes focus from the editor.
func (b *Buffer) Blur() {
	b.editor.Blur()
}

// PushUndo saves a content snapshot to the undo stack.
// Clears the redo stack. Caps the undo stack at 200 entries.
func (b *Buffer) PushUndo(content string) {
	b.undoStack = append(b.undoStack, content)
	if len(b.undoStack) > 200 {
		b.undoStack = b.undoStack[len(b.undoStack)-200:]
	}
	b.redoStack = nil
}

// Undo pops the last undo state, pushes the current state to redo, and returns the restored content.
// Returns empty string if nothing to undo.
func (b *Buffer) Undo() string {
	if len(b.undoStack) == 0 {
		return ""
	}
	current := b.editor.Value()
	b.redoStack = append(b.redoStack, current)

	prev := b.undoStack[len(b.undoStack)-1]
	b.undoStack = b.undoStack[:len(b.undoStack)-1]
	b.editor.SetValue(prev)
	return prev
}

// Redo pops the last redo state, pushes the current state to undo, and returns the restored content.
// Returns empty string if nothing to redo.
func (b *Buffer) Redo() string {
	if len(b.redoStack) == 0 {
		return ""
	}
	current := b.editor.Value()
	b.undoStack = append(b.undoStack, current)

	next := b.redoStack[len(b.redoStack)-1]
	b.redoStack = b.redoStack[:len(b.redoStack)-1]
	b.editor.SetValue(next)
	return next
}

// CheckDirty compares the current editor value to the lastContent and updates the dirty flag.
func (b *Buffer) CheckDirty() {
	b.dirty = b.editor.Value() != b.lastContent
}

// SnapshotForUndo captures the current content as an undo snapshot if it has changed since the last snapshot.
func (b *Buffer) SnapshotForUndo() {
	current := b.editor.Value()
	if len(b.undoStack) == 0 || b.undoStack[len(b.undoStack)-1] != current {
		b.PushUndo(current)
	}
}

// Update forwards key messages to the editor and checks dirty state.
func (b *Buffer) Update(msg tea.Msg) tea.Cmd {
	cmd := b.editor.Update(msg)
	b.CheckDirty()
	return cmd
}

// View returns the rendered buffer with syntax highlighting.
func (b *Buffer) View() string {
	return b.editor.View()
}

// Save writes the current content to disk and marks the buffer clean.
func (b *Buffer) Save() error {
	content := b.editor.Value()
	err := os.WriteFile(b.filePath, []byte(content), 0644)
	if err != nil {
		return err
	}
	b.lastContent = content
	b.MarkClean()
	return nil
}

// LoadFile reads a file from disk and creates a new Buffer for it.
// Normalizes line endings (CRLF -> LF).
func LoadFile(path string, tabSize int) (*Buffer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	return NewBuffer(path, content, tabSize), nil
}
