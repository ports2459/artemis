package editor

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eden/artemis/internal/app"
	editorlib "github.com/eden/artemis/internal/editor"
	"github.com/eden/artemis/internal/panel"
)

type focus int

const (
	focusTree focus = iota
	focusEditor
	focusFind
)

const (
	treeWidthDefault = 28
	editorTabBarH    = 1
)

type Model struct {
	width       int
	height      int
	tabSize     int
	projectPath string

	tree    *editorlib.FileTree
	tabBar  *editorlib.TabBar
	buffers map[string]*editorlib.Buffer

	focus       focus
	sidebarOpen bool
	treeWidth   int

	findInput  textinput.Model
	findActive bool

	diagnostics map[string][]app.DiagnosticItem
}

var _ panel.Panel = (*Model)(nil)

func NewModel(tabSize int) Model {
	fi := textinput.New()
	fi.Placeholder = "Search..."
	fi.CharLimit = 256

	return Model{
		tabSize:     tabSize,
		tree:        editorlib.NewFileTree(),
		tabBar:      editorlib.NewTabBar(),
		buffers:     make(map[string]*editorlib.Buffer),
		focus:       focusTree,
		sidebarOpen: true,
		treeWidth:   treeWidthDefault,
		findInput:   fi,
		diagnostics: make(map[string][]app.DiagnosticItem),
	}
}

func (m *Model) ID() panel.PanelID { return panel.PanelEditor }

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.layoutChildren()
}

func (m *Model) StatusInfo() string {
	if path := m.tabBar.ActivePath(); path != "" {
		buf := m.buffers[path]
		parts := []string{filepath.Base(path)}
		if buf != nil && buf.Dirty() {
			parts = append(parts, "modified")
		}
		lang := editorlib.DetectLanguage(path)
		switch lang {
		case editorlib.LangCSharp:
			parts = append(parts, "C#")
		case editorlib.LangJSON:
			parts = append(parts, "JSON")
		case editorlib.LangTOML:
			parts = append(parts, "TOML")
		case editorlib.LangLua:
			parts = append(parts, "Lua")
		}
		if diags, ok := m.diagnostics[path]; ok && len(diags) > 0 {
			errors, warnings := 0, 0
			for _, d := range diags {
				switch d.Severity {
				case 1:
					errors++
				case 2:
					warnings++
				}
			}
			if errors > 0 || warnings > 0 {
				parts = append(parts, fmt.Sprintf("E:%d W:%d", errors, warnings))
			}
		}
		return strings.Join(parts, " | ")
	}
	if m.projectPath != "" {
		return filepath.Base(m.projectPath)
	}
	return "No file open"
}

func (m *Model) layoutChildren() {
	editorWidth := m.width
	editorHeight := m.height - editorTabBarH

	if m.sidebarOpen {
		editorWidth -= m.treeWidth + 1
		m.tree.SetSize(m.treeWidth, m.height)
	}

	if m.findActive {
		editorHeight--
	}

	m.tree.SetFocused(m.focus == focusTree)

	for _, buf := range m.buffers {
		buf.SetSize(editorWidth, editorHeight)
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case app.OpenProjectMsg:
		m.projectPath = msg.Path
		m.tree.SetRoot(msg.Path)
		m.focus = focusTree
		m.tree.SetFocused(true)
		m.layoutChildren()
		return m, nil

	case app.FileOpenMsg:
		return m, m.openFile(msg.Path)

	case app.FileSavedMsg:
		return m, nil

	case app.DiagnosticsMsg:
		m.diagnostics[msg.Path] = msg.Diagnostics
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	if m.focus == focusEditor {
		if buf := m.activeBuffer(); buf != nil {
			cmd := buf.Update(msg)
			m.tabBar.SetModified(m.tabBar.ActivePath(), buf.Dirty())
			return m, cmd
		}
	}

	if m.focus == focusFind {
		var cmd tea.Cmd
		m.findInput, cmd = m.findInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, editorKeys.ToggleSidebar):
		m.sidebarOpen = !m.sidebarOpen
		m.layoutChildren()
		return m, nil

	case key.Matches(msg, editorKeys.Save):
		return m, m.saveActiveFile()

	case key.Matches(msg, editorKeys.CloseTab):
		return m, m.closeActiveTab()

	case key.Matches(msg, editorKeys.NextTab):
		m.tabBar.Next()
		if buf := m.activeBuffer(); buf != nil {
			m.focus = focusEditor
			m.tree.SetFocused(false)
			m.layoutChildren()
			return m, buf.Focus()
		}
		return m, nil

	case key.Matches(msg, editorKeys.PrevTab):
		m.tabBar.Prev()
		if buf := m.activeBuffer(); buf != nil {
			m.focus = focusEditor
			m.tree.SetFocused(false)
			m.layoutChildren()
			return m, buf.Focus()
		}
		return m, nil

	case key.Matches(msg, editorKeys.Find):
		m.findActive = !m.findActive
		if m.findActive {
			m.focus = focusFind
			m.findInput.Focus()
			m.layoutChildren()
			return m, m.findInput.Cursor.BlinkCmd()
		}
		m.focus = focusEditor
		m.findInput.Blur()
		m.layoutChildren()
		return m, nil

	case key.Matches(msg, editorKeys.Undo):
		if m.focus == focusEditor {
			if buf := m.activeBuffer(); buf != nil {
				buf.Undo()
			}
		}
		return m, nil

	case key.Matches(msg, editorKeys.Redo):
		if m.focus == focusEditor {
			if buf := m.activeBuffer(); buf != nil {
				buf.Redo()
			}
		}
		return m, nil
	}

	switch m.focus {
	case focusTree:
		if msg.String() == "tab" && m.tabBar.Count() > 0 {
			m.focus = focusEditor
			m.tree.SetFocused(false)
			if buf := m.activeBuffer(); buf != nil {
				return m, buf.Focus()
			}
			return m, nil
		}
		cmd := m.tree.Update(msg)
		return m, cmd

	case focusEditor:
		if msg.String() == "esc" {
			m.focus = focusTree
			m.tree.SetFocused(true)
			if buf := m.activeBuffer(); buf != nil {
				buf.Blur()
			}
			return m, nil
		}
		if buf := m.activeBuffer(); buf != nil {
			buf.SnapshotForUndo()
			cmd := buf.Update(msg)
			m.tabBar.SetModified(m.tabBar.ActivePath(), buf.Dirty())
			return m, cmd
		}

	case focusFind:
		if msg.String() == "esc" {
			m.findActive = false
			m.focus = focusEditor
			m.findInput.Blur()
			m.layoutChildren()
			return m, nil
		}
		var cmd tea.Cmd
		m.findInput, cmd = m.findInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) activeBuffer() *editorlib.Buffer {
	path := m.tabBar.ActivePath()
	if path == "" {
		return nil
	}
	return m.buffers[path]
}

func (m *Model) openFile(path string) tea.Cmd {
	isNew := m.tabBar.Open(path, filepath.Base(path))
	if !isNew {
		m.focus = focusEditor
		m.tree.SetFocused(false)
		m.layoutChildren()
		if buf := m.buffers[path]; buf != nil {
			return buf.Focus()
		}
		return nil
	}

	buf, err := editorlib.LoadFile(path, m.tabSize)
	if err != nil {
		return func() tea.Msg {
			return app.StatusMsg{Text: fmt.Sprintf("Error: %v", err)}
		}
	}
	m.buffers[path] = buf
	m.focus = focusEditor
	m.tree.SetFocused(false)
	m.layoutChildren()

	// Run linter on file open
	m.runLinter(buf, path)

	return buf.Focus()
}

func (m *Model) saveActiveFile() tea.Cmd {
	buf := m.activeBuffer()
	if buf == nil {
		return nil
	}
	// Auto-format on save
	formatted := editorlib.Format(buf.Content(), buf.Language())
	if formatted != buf.Content() {
		buf.SetContent(formatted)
	}

	if err := buf.Save(); err != nil {
		return func() tea.Msg {
			return app.StatusMsg{Text: fmt.Sprintf("Save error: %v", err)}
		}
	}
	path := buf.FilePath()
	m.tabBar.SetModified(path, false)

	// Run linter
	m.runLinter(buf, path)

	return func() tea.Msg {
		return app.FileSavedMsg{Path: path}
	}
}

// runLinter runs the built-in linter on a buffer and stores results in diagnostics.
func (m *Model) runLinter(buf *editorlib.Buffer, path string) {
	lintResults := editorlib.Lint(buf.Content(), buf.Language())
	if len(lintResults) > 0 {
		diags := make([]app.DiagnosticItem, len(lintResults))
		for i, r := range lintResults {
			diags[i] = app.DiagnosticItem{
				Line:     r.Line,
				Col:      r.Col,
				Severity: r.Severity,
				Message:  r.Message,
			}
		}
		m.diagnostics[path] = diags
	} else {
		delete(m.diagnostics, path)
	}
}

func (m *Model) closeActiveTab() tea.Cmd {
	closedPath := m.tabBar.CloseCurrent()
	if closedPath != "" {
		delete(m.buffers, closedPath)
	}
	if m.tabBar.Count() == 0 {
		m.focus = focusTree
		m.tree.SetFocused(true)
	} else {
		if buf := m.activeBuffer(); buf != nil {
			m.layoutChildren()
			return buf.Focus()
		}
	}
	m.layoutChildren()
	return nil
}

func (m *Model) View() string {
	if m.projectPath == "" {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(app.ColorOverlay1).Render(
				"Open a project to start editing\n\n"+
					app.StyleDim.Render("Alt+1 to switch to Project panel")))
	}

	editorWidth := m.width
	if m.sidebarOpen {
		editorWidth -= m.treeWidth + 1
	}
	tabView := m.tabBar.View(editorWidth)
	editorContent := m.viewEditorContent(editorWidth)
	editorColumn := lipgloss.JoinVertical(lipgloss.Left, tabView, editorContent)

	if m.sidebarOpen {
		treeView := m.tree.View()
		borderStyle := lipgloss.NewStyle().Foreground(app.ColorSurface1)
		var borderLines []string
		for i := 0; i < m.height; i++ {
			borderLines = append(borderLines, "\u2502")
		}
		border := borderStyle.Render(strings.Join(borderLines, "\n"))
		return lipgloss.JoinHorizontal(lipgloss.Top, treeView, border, editorColumn)
	}

	return editorColumn
}

func (m *Model) viewEditorContent(width int) string {
	contentHeight := m.height - editorTabBarH
	if m.findActive {
		contentHeight--
	}

	buf := m.activeBuffer()
	path := m.tabBar.ActivePath()
	diags := m.diagnostics[path]

	// Reserve space for diagnostics panel if there are diagnostics
	diagHeight := 0
	if len(diags) > 0 {
		diagHeight = len(diags) + 1 // +1 for header
		if diagHeight > 6 {
			diagHeight = 6 // cap at 6 lines
		}
		contentHeight -= diagHeight
		if contentHeight < 3 {
			contentHeight = 3
		}
	}

	var content string
	if buf == nil {
		content = lipgloss.Place(width, contentHeight, lipgloss.Center, lipgloss.Center,
			app.StyleDim.Render("Select a file to edit"))
	} else {
		content = buf.View()
	}

	// Append diagnostics panel
	if len(diags) > 0 {
		diagView := m.viewDiagnostics(width, diags, diagHeight)
		content = lipgloss.JoinVertical(lipgloss.Left, content, diagView)
	}

	if m.findActive {
		findBar := m.viewFindBar(width)
		return lipgloss.JoinVertical(lipgloss.Left, content, findBar)
	}

	return content
}

func (m *Model) viewDiagnostics(width int, diags []app.DiagnosticItem, maxLines int) string {
	headerStyle := lipgloss.NewStyle().
		Foreground(app.ColorMauve).
		Bold(true)
	errStyle := lipgloss.NewStyle().Foreground(app.ColorRed).Bold(true)
	warnStyle := lipgloss.NewStyle().Foreground(app.ColorPeach).Bold(true)
	infoStyle := lipgloss.NewStyle().Foreground(app.ColorBlue)
	hintStyle := lipgloss.NewStyle().Foreground(app.ColorOverlay1)
	lineStyle := lipgloss.NewStyle().Foreground(app.ColorSubtext0)

	var b strings.Builder
	b.WriteString(headerStyle.Render(app.HRule(width)))

	shown := maxLines - 1 // subtract header
	if shown > len(diags) {
		shown = len(diags)
	}

	for i := 0; i < shown; i++ {
		d := diags[i]
		b.WriteString("\n")

		var tag string
		switch d.Severity {
		case 1:
			tag = errStyle.Render("ERR")
		case 2:
			tag = warnStyle.Render("WRN")
		case 3:
			tag = infoStyle.Render("INF")
		default:
			tag = hintStyle.Render("HNT")
		}

		loc := lineStyle.Render(fmt.Sprintf("Line %d:", d.Line))
		msg := d.Message
		maxMsg := width - 16
		if maxMsg > 0 && len(msg) > maxMsg {
			msg = msg[:maxMsg-1] + "…"
		}
		b.WriteString(fmt.Sprintf(" %s  %s %s", tag, loc, msg))
	}

	return b.String()
}

func (m *Model) viewFindBar(width int) string {
	barStyle := lipgloss.NewStyle().
		Background(app.ColorSurface0).
		Foreground(app.ColorText).
		Width(width)
	label := lipgloss.NewStyle().
		Foreground(app.ColorMauve).
		Bold(true).
		Render("Find: ")
	return barStyle.Render(label + m.findInput.View())
}
