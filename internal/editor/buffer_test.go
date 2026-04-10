package editor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBufferNewAndContent(t *testing.T) {
	buf := NewBuffer("/tmp/test.cs", "hello world", 4)

	if buf.FilePath() != "/tmp/test.cs" {
		t.Errorf("expected path /tmp/test.cs, got %q", buf.FilePath())
	}

	if buf.FileName() != "test.cs" {
		t.Errorf("expected filename test.cs, got %q", buf.FileName())
	}

	if buf.Language() != LangCSharp {
		t.Errorf("expected LangCSharp, got %d", buf.Language())
	}

	if buf.Dirty() {
		t.Error("new buffer should not be dirty")
	}

	if buf.Content() != "hello world" {
		t.Errorf("expected content 'hello world', got %q", buf.Content())
	}
}

func TestBufferDirtyFlag(t *testing.T) {
	buf := NewBuffer("/tmp/test.cs", "content", 4)

	if buf.Dirty() {
		t.Error("new buffer should not be dirty")
	}

	buf.MarkDirty()
	if !buf.Dirty() {
		t.Error("buffer should be dirty after MarkDirty")
	}

	buf.MarkClean()
	if buf.Dirty() {
		t.Error("buffer should be clean after MarkClean")
	}
}

func TestBufferUndo(t *testing.T) {
	buf := NewBuffer("/tmp/test.cs", "original", 4)

	buf.PushUndo("state1")
	buf.PushUndo("state2")

	result := buf.Undo()
	if result != "state2" {
		t.Errorf("expected undo to return 'state2', got %q", result)
	}

	result = buf.Undo()
	if result != "state1" {
		t.Errorf("expected undo to return 'state1', got %q", result)
	}

	// Nothing more to undo
	result = buf.Undo()
	if result != "" {
		t.Errorf("expected empty string when nothing to undo, got %q", result)
	}
}

func TestBufferRedo(t *testing.T) {
	buf := NewBuffer("/tmp/test.cs", "original", 4)

	buf.PushUndo("state1")
	buf.PushUndo("state2")

	// Undo: pops "state2", pushes current ("original") to redo, textarea = "state2"
	result := buf.Undo()
	if result != "state2" {
		t.Errorf("expected undo to return 'state2', got %q", result)
	}
	// Undo: pops "state1", pushes current ("state2") to redo, textarea = "state1"
	result = buf.Undo()
	if result != "state1" {
		t.Errorf("expected undo to return 'state1', got %q", result)
	}

	// Redo: pops "state2" from redo, pushes current ("state1") to undo, textarea = "state2"
	result = buf.Redo()
	if result != "state2" {
		t.Errorf("expected redo to return 'state2', got %q", result)
	}

	// Redo: pops "original" from redo, pushes current ("state2") to undo, textarea = "original"
	result = buf.Redo()
	if result != "original" {
		t.Errorf("expected redo to return 'original', got %q", result)
	}

	// Nothing more to redo
	result = buf.Redo()
	if result != "" {
		t.Errorf("expected empty string when nothing to redo, got %q", result)
	}
}

func TestBufferUndoRedoClearsRedoOnNewEdit(t *testing.T) {
	buf := NewBuffer("/tmp/test.cs", "original", 4)

	buf.PushUndo("state1")
	buf.PushUndo("state2")

	// Undo once
	buf.Undo()

	// Push a new edit - should clear redo stack
	buf.PushUndo("state3")

	// Redo should return nothing since redo was cleared
	result := buf.Redo()
	if result != "" {
		t.Errorf("expected empty redo after new edit, got %q", result)
	}
}

func TestBufferSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.cs")
	os.WriteFile(path, []byte("original content"), 0644)

	buf, err := LoadFile(path, 4)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	if buf.Content() != "original content" {
		t.Errorf("expected 'original content', got %q", buf.Content())
	}
	if buf.Dirty() {
		t.Error("loaded buffer should not be dirty")
	}
}

func TestBufferLoadFileNormalizesLineEndings(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.cs")
	os.WriteFile(path, []byte("line1\r\nline2\r\nline3"), 0644)

	buf, err := LoadFile(path, 4)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	content := buf.Content()
	if content != "line1\nline2\nline3" {
		t.Errorf("expected normalized line endings, got %q", content)
	}
}

func TestBufferUndoCap(t *testing.T) {
	buf := NewBuffer("/tmp/test.cs", "content", 4)
	for i := 0; i < 250; i++ {
		buf.PushUndo("state")
	}
	if len(buf.undoStack) != 200 {
		t.Errorf("expected undo stack capped at 200, got %d", len(buf.undoStack))
	}
}
