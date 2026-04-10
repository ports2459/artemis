package editor

import (
	"testing"
)

func TestTabBarOpen(t *testing.T) {
	tb := NewTabBar()

	if tb.Count() != 0 {
		t.Errorf("expected 0 tabs, got %d", tb.Count())
	}
	if tb.ActiveIndex() != -1 {
		t.Errorf("expected active -1, got %d", tb.ActiveIndex())
	}

	isNew := tb.Open("/path/a.cs", "a.cs")
	if !isNew {
		t.Error("expected Open to return true for new tab")
	}
	if tb.Count() != 1 {
		t.Errorf("expected 1 tab, got %d", tb.Count())
	}
	if tb.ActiveIndex() != 0 {
		t.Errorf("expected active 0, got %d", tb.ActiveIndex())
	}
	if tb.ActivePath() != "/path/a.cs" {
		t.Errorf("expected active path /path/a.cs, got %q", tb.ActivePath())
	}
}

func TestTabBarOpenDuplicate(t *testing.T) {
	tb := NewTabBar()
	tb.Open("/path/a.cs", "a.cs")
	tb.Open("/path/b.cs", "b.cs")

	if tb.ActiveIndex() != 1 {
		t.Errorf("expected active 1, got %d", tb.ActiveIndex())
	}

	// Open duplicate of first tab
	isNew := tb.Open("/path/a.cs", "a.cs")
	if isNew {
		t.Error("expected Open to return false for duplicate")
	}
	if tb.Count() != 2 {
		t.Errorf("expected still 2 tabs, got %d", tb.Count())
	}
	if tb.ActiveIndex() != 0 {
		t.Errorf("expected active switched to 0, got %d", tb.ActiveIndex())
	}
}

func TestTabBarSwitch(t *testing.T) {
	tb := NewTabBar()
	tb.Open("/path/a.cs", "a.cs")
	tb.Open("/path/b.cs", "b.cs")
	tb.Open("/path/c.cs", "c.cs")

	// Active is 2 (c.cs)
	if tb.ActiveIndex() != 2 {
		t.Errorf("expected active 2, got %d", tb.ActiveIndex())
	}

	// Next wraps to 0
	tb.Next()
	if tb.ActiveIndex() != 0 {
		t.Errorf("expected Next to wrap to 0, got %d", tb.ActiveIndex())
	}

	// Prev wraps to 2
	tb.Prev()
	if tb.ActiveIndex() != 2 {
		t.Errorf("expected Prev to wrap to 2, got %d", tb.ActiveIndex())
	}

	// Prev to 1
	tb.Prev()
	if tb.ActiveIndex() != 1 {
		t.Errorf("expected Prev to go to 1, got %d", tb.ActiveIndex())
	}
}

func TestTabBarClose(t *testing.T) {
	tb := NewTabBar()
	tb.Open("/path/a.cs", "a.cs")
	tb.Open("/path/b.cs", "b.cs")
	tb.Open("/path/c.cs", "c.cs")

	// Active is 2 (c.cs)
	tb.SetActive(1) // switch to b.cs

	closed := tb.CloseCurrent()
	if closed != "/path/b.cs" {
		t.Errorf("expected closed path /path/b.cs, got %q", closed)
	}
	if tb.Count() != 2 {
		t.Errorf("expected 2 tabs, got %d", tb.Count())
	}
	// Active should stay at 1 (now c.cs)
	if tb.ActivePath() != "/path/c.cs" {
		t.Errorf("expected active path /path/c.cs, got %q", tb.ActivePath())
	}
}

func TestTabBarCloseLastTab(t *testing.T) {
	tb := NewTabBar()
	tb.Open("/path/a.cs", "a.cs")

	closed := tb.CloseCurrent()
	if closed != "/path/a.cs" {
		t.Errorf("expected closed path /path/a.cs, got %q", closed)
	}
	if tb.Count() != 0 {
		t.Errorf("expected 0 tabs, got %d", tb.Count())
	}
	if tb.ActiveIndex() != -1 {
		t.Errorf("expected active -1, got %d", tb.ActiveIndex())
	}
}

func TestTabBarModified(t *testing.T) {
	tb := NewTabBar()
	tb.Open("/path/a.cs", "a.cs")

	if tb.IsModified("/path/a.cs") {
		t.Error("tab should not be modified initially")
	}

	tb.SetModified("/path/a.cs", true)
	if !tb.IsModified("/path/a.cs") {
		t.Error("tab should be modified after SetModified(true)")
	}

	tb.SetModified("/path/a.cs", false)
	if tb.IsModified("/path/a.cs") {
		t.Error("tab should not be modified after SetModified(false)")
	}
}

func TestTabBarTabs(t *testing.T) {
	tb := NewTabBar()
	tb.Open("/path/a.cs", "a.cs")
	tb.Open("/path/b.cs", "b.cs")

	tabs := tb.Tabs()
	if len(tabs) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(tabs))
	}

	// Verify it's a copy
	tabs[0].Name = "modified"
	original := tb.Tabs()
	if original[0].Name != "a.cs" {
		t.Error("Tabs() should return a copy, not a reference")
	}
}
