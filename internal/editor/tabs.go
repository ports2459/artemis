package editor

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/eden/artemis/internal/app"
)

// Tab represents a single open editor tab.
type Tab struct {
	Path     string
	Name     string
	Modified bool
}

// TabBar manages a set of open tabs.
type TabBar struct {
	tabs   []Tab
	active int
}

// NewTabBar creates a new empty TabBar with no active tab.
func NewTabBar() *TabBar {
	return &TabBar{
		active: -1,
	}
}

// Count returns the number of open tabs.
func (tb *TabBar) Count() int {
	return len(tb.tabs)
}

// ActiveIndex returns the index of the active tab (-1 if none).
func (tb *TabBar) ActiveIndex() int {
	return tb.active
}

// ActivePath returns the path of the active tab, or "" if none.
func (tb *TabBar) ActivePath() string {
	if tb.active >= 0 && tb.active < len(tb.tabs) {
		return tb.tabs[tb.active].Path
	}
	return ""
}

// Open opens a file in the tab bar. If a tab with the same path already exists,
// it switches to it. Otherwise, a new tab is added and activated.
// Returns true if a new tab was created.
func (tb *TabBar) Open(path, name string) bool {
	// Check if already open
	for i, t := range tb.tabs {
		if t.Path == path {
			tb.active = i
			return false
		}
	}
	// Add new tab
	tb.tabs = append(tb.tabs, Tab{Path: path, Name: name})
	tb.active = len(tb.tabs) - 1
	return true
}

// SetActive sets the active tab by index.
func (tb *TabBar) SetActive(index int) {
	if index >= 0 && index < len(tb.tabs) {
		tb.active = index
	}
}

// Next switches to the next tab, wrapping around.
func (tb *TabBar) Next() {
	if len(tb.tabs) == 0 {
		return
	}
	tb.active = (tb.active + 1) % len(tb.tabs)
}

// Prev switches to the previous tab, wrapping around.
func (tb *TabBar) Prev() {
	if len(tb.tabs) == 0 {
		return
	}
	tb.active = (tb.active - 1 + len(tb.tabs)) % len(tb.tabs)
}

// CloseCurrent closes the currently active tab and returns its path.
// Adjusts the active index appropriately.
func (tb *TabBar) CloseCurrent() string {
	if tb.active < 0 || tb.active >= len(tb.tabs) {
		return ""
	}
	closedPath := tb.tabs[tb.active].Path
	tb.tabs = append(tb.tabs[:tb.active], tb.tabs[tb.active+1:]...)

	if len(tb.tabs) == 0 {
		tb.active = -1
	} else if tb.active >= len(tb.tabs) {
		tb.active = len(tb.tabs) - 1
	}
	return closedPath
}

// SetModified sets the modified flag for a tab identified by path.
func (tb *TabBar) SetModified(path string, modified bool) {
	for i, t := range tb.tabs {
		if t.Path == path {
			tb.tabs[i].Modified = modified
			return
		}
	}
}

// IsModified returns whether the tab with the given path is marked as modified.
func (tb *TabBar) IsModified(path string) bool {
	for _, t := range tb.tabs {
		if t.Path == path {
			return t.Modified
		}
	}
	return false
}

// Tabs returns a copy of the current tabs.
func (tb *TabBar) Tabs() []Tab {
	result := make([]Tab, len(tb.tabs))
	copy(result, tb.tabs)
	return result
}

// View renders the tab bar as a string within the given width.
func (tb *TabBar) View(width int) string {
	if len(tb.tabs) == 0 {
		return app.StyleDim.Render(strings.Repeat(" ", width))
	}

	activeStyle := lipgloss.NewStyle().
		Foreground(app.ColorMauve).
		Bold(true)
	inactiveStyle := lipgloss.NewStyle().
		Foreground(app.ColorSurface2)
	modifiedStyle := lipgloss.NewStyle().
		Foreground(app.ColorPeach)
	separatorStyle := lipgloss.NewStyle().
		Foreground(app.ColorSurface1)

	separator := separatorStyle.Render(" | ")

	var parts []string
	for i, tab := range tb.tabs {
		name := tab.Name
		if tab.Modified {
			name = modifiedStyle.Render("●") + " " + name
		}
		if i == tb.active {
			parts = append(parts, activeStyle.Render(name))
		} else {
			parts = append(parts, inactiveStyle.Render(name))
		}
	}

	result := strings.Join(parts, separator)

	// Pad to width
	rendered := lipgloss.Width(result)
	if rendered < width {
		result += strings.Repeat(" ", width-rendered)
	}

	return result
}
