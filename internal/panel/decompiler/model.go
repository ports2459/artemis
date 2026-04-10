package decompiler

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eden/artemis/internal/app"
	"github.com/eden/artemis/internal/config"
	"github.com/eden/artemis/internal/decompile"
	"github.com/eden/artemis/internal/editor"
	"github.com/eden/artemis/internal/panel"
)

// focus tracks which column has keyboard focus.
type focus int

const (
	focusAssemblies focus = iota
	focusTypes
	focusSource
	focusSearch
)

const (
	assemblyWidthDefault = 32
	typeWidthDefault     = 36
	minSourceWidth       = 30
)

// Model is the decompiler panel. It presents a three-column layout:
// assembly list | type list | decompiled source.
type Model struct {
	width, height int
	decompiler    *decompile.Decompiler
	projectPath   string
	managedDir    string

	// Assembly list (left column)
	assemblies     []decompile.Assembly
	assemblyCursor int
	assemblyScroll int

	// Type list (center column)
	types      []decompile.TypeEntry
	typeCursor int
	typeScroll int

	// Source view (right column)
	source       []string // lines of decompiled source
	sourceScroll int
	sourceType   string // currently decompiled type name

	focus            focus
	assemblyWidth    int
	typeWidth        int
	scanError        string
	scanning         bool
	decompiling      bool
	buildingHelper   bool
	selectedAssembly string

	// Search state
	searchActive  bool
	searchInput   textinput.Model
	searchResults []searchResult
	searchCursor  int
	searchScroll  int
	searching     bool

	// When navigating to a search result, remember the target line
	targetScrollLine int
}

// Compile-time check: *Model satisfies panel.Panel.
var _ panel.Panel = (*Model)(nil)

// NewModel creates a new decompiler panel.
func NewModel() Model {
	si := textinput.New()
	si.Placeholder = "Search all assemblies..."
	si.CharLimit = 256

	return Model{
		decompiler:    decompile.NewDecompiler(),
		assemblyWidth: assemblyWidthDefault,
		typeWidth:     typeWidthDefault,
		focus:         focusAssemblies,
		searchInput:   si,
	}
}

// ---------------------------------------------------------------------------
// panel.Panel interface
// ---------------------------------------------------------------------------

func (m *Model) ID() panel.PanelID { return panel.PanelDecompiler }

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Model) StatusInfo() string {
	if !m.decompiler.Available() {
		return "Decompiler (dotnet not found)"
	}
	if m.buildingHelper {
		return "Building decompiler..."
	}
	if m.searching {
		return "Searching..."
	}
	if m.searchActive && m.searchResults != nil {
		return fmt.Sprintf("Search: %d results", len(m.searchResults))
	}
	if m.sourceType != "" {
		return m.sourceType
	}
	return fmt.Sprintf("%d assemblies", len(m.assemblies))
}

// ---------------------------------------------------------------------------
// tea.Model interface
// ---------------------------------------------------------------------------

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Forward cursor blink and other messages to the search input when active
	if m.searchActive && m.focus == focusSearch {
		if _, ok := msg.(tea.KeyMsg); !ok {
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			if cmd != nil {
				return m, cmd
			}
		}
	}

	switch msg := msg.(type) {
	case app.OpenProjectMsg:
		m.projectPath = msg.Path
		m.scanError = ""
		m.assemblies = nil
		m.types = nil
		m.source = nil
		m.sourceType = ""
		m.assemblyCursor = 0
		m.assemblyScroll = 0
		m.typeCursor = 0
		m.typeScroll = 0
		m.sourceScroll = 0
		m.selectedAssembly = ""

		cfgPath := filepath.Join(msg.Path, "artemis.toml")
		cfg, err := config.LoadProject(cfgPath)
		if err != nil {
			m.scanError = fmt.Sprintf("Config error: %v", err)
			return m, nil
		}
		if cfg.Game.Path == "" {
			m.scanError = "No game path set. Edit artemis.toml to add [game] path."
			return m, nil
		}

		managed := decompile.FindManagedDir(cfg.Game.Path)
		if managed == "" {
			m.scanError = "No Managed/ directory found in game path."
			return m, nil
		}
		m.managedDir = managed
		m.scanning = true
		// Scan assemblies and build the helper tool in parallel
		m.buildingHelper = true
		return m, tea.Batch(m.doScanAssemblies(), m.doBuildHelper())

	case helperBuiltMsg:
		m.buildingHelper = false
		if msg.err != nil {
			m.scanError = fmt.Sprintf("Helper build failed: %v", msg.err)
		}
		return m, nil

	case scanAssembliesMsg:
		m.scanning = false
		if msg.err != nil {
			m.scanError = fmt.Sprintf("Scan error: %v", msg.err)
			return m, nil
		}
		m.assemblies = msg.assemblies
		return m, nil

	case listTypesMsg:
		m.decompiling = false
		if msg.err != nil {
			m.scanError = fmt.Sprintf("Error listing types: %v", msg.err)
			return m, nil
		}
		m.scanError = ""
		m.types = msg.types
		m.typeCursor = 0
		m.typeScroll = 0
		return m, nil

	case decompileMsg:
		m.decompiling = false
		if msg.err != nil {
			m.source = []string{fmt.Sprintf("Decompile error: %v", msg.err)}
			return m, nil
		}
		m.sourceType = msg.typeName
		// Expand tabs to spaces to prevent terminal line wrapping
		raw := strings.ReplaceAll(msg.source, "\t", "    ")
		m.source = strings.Split(raw, "\n")
		m.sourceScroll = 0
		// If we navigated here from a search result, scroll to the target line
		if m.targetScrollLine > 0 {
			m.sourceScroll = m.targetScrollLine - 1
			maxScroll := len(m.source) - m.sourceViewHeight()
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.sourceScroll > maxScroll {
				m.sourceScroll = maxScroll
			}
			m.targetScrollLine = 0
		}
		return m, nil

	case searchResultsMsg:
		m.searching = false
		if msg.err != nil {
			m.scanError = fmt.Sprintf("Search error: %v", msg.err)
			return m, nil
		}
		m.searchResults = msg.results
		m.searchCursor = 0
		m.searchScroll = 0
		return m, nil

	case tea.KeyMsg:
		if m.searchActive {
			return m.handleSearchKey(msg)
		}
		return m.handleKey(msg)
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// Internal messages
// ---------------------------------------------------------------------------

type helperBuiltMsg struct {
	err error
}

type scanAssembliesMsg struct {
	assemblies []decompile.Assembly
	err        error
}

type listTypesMsg struct {
	types []decompile.TypeEntry
	err   error
}

type decompileMsg struct {
	typeName string
	source   string
	err      error
}

type searchResult struct {
	assembly string // display name
	asmPath  string // full path to assembly
	line     int    // 1-based line number
	content  string // trimmed matching line
}

type searchResultsMsg struct {
	results []searchResult
	err     error
}

func (m *Model) doBuildHelper() tea.Cmd {
	dc := m.decompiler
	return func() tea.Msg {
		err := dc.EnsureHelper()
		return helperBuiltMsg{err: err}
	}
}

func (m *Model) doScanAssemblies() tea.Cmd {
	dir := m.managedDir
	return func() tea.Msg {
		assemblies, err := decompile.ScanAssemblies(dir)
		return scanAssembliesMsg{assemblies: assemblies, err: err}
	}
}

func (m *Model) doListTypes(assemblyPath string) tea.Cmd {
	dc := m.decompiler
	return func() tea.Msg {
		types, err := dc.ListTypes(assemblyPath)
		return listTypesMsg{types: types, err: err}
	}
}

func (m *Model) doDecompileType(assemblyPath, typeName string) tea.Cmd {
	dc := m.decompiler
	return func() tea.Msg {
		source, err := dc.DecompileType(assemblyPath, typeName)
		return decompileMsg{typeName: typeName, source: source, err: err}
	}
}

func (m *Model) doDecompileAssembly(assemblyPath string) tea.Cmd {
	dc := m.decompiler
	name := filepath.Base(assemblyPath)
	return func() tea.Msg {
		source, err := dc.DecompileAssembly(assemblyPath)
		return decompileMsg{typeName: name, source: source, err: err}
	}
}

func (m *Model) doSearchAll(query string) tea.Cmd {
	dc := m.decompiler
	assemblies := make([]decompile.Assembly, len(m.assemblies))
	copy(assemblies, m.assemblies)
	return func() tea.Msg {
		var results []searchResult
		queryLower := strings.ToLower(query)
		for _, asm := range assemblies {
			source, err := dc.DecompileAssembly(asm.Path)
			if err != nil {
				continue
			}
			// Expand tabs for consistent display
			source = strings.ReplaceAll(source, "\t", "    ")
			lines := strings.Split(source, "\n")
			for i, line := range lines {
				if strings.Contains(strings.ToLower(line), queryLower) {
					results = append(results, searchResult{
						assembly: asm.Name,
						asmPath:  asm.Path,
						line:     i + 1,
						content:  strings.TrimSpace(line),
					})
				}
			}
		}
		return searchResultsMsg{results: results}
	}
}

// ---------------------------------------------------------------------------
// Key handling
// ---------------------------------------------------------------------------

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.cycleFocusForward()
		return m, nil
	case "shift+tab":
		m.cycleFocusBackward()
		return m, nil
	case "/", "ctrl+f":
		if len(m.assemblies) > 0 && m.decompiler.Ready() {
			m.searchActive = true
			m.focus = focusSearch
			m.searchResults = nil
			m.searchCursor = 0
			m.searchScroll = 0
			m.searching = false
			m.searchInput.SetValue("")
			m.searchInput.Focus()
			return m, m.searchInput.Cursor.BlinkCmd()
		}
		return m, nil
	}

	switch m.focus {
	case focusAssemblies:
		return m.handleAssemblyKey(msg)
	case focusTypes:
		return m.handleTypeKey(msg)
	case focusSource:
		return m.handleSourceKey(msg)
	}
	return m, nil
}

func (m *Model) handleAssemblyKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.assemblyCursor > 0 {
			m.assemblyCursor--
			m.ensureAssemblyVisible()
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if m.assemblyCursor < len(m.assemblies)-1 {
			m.assemblyCursor++
			m.ensureAssemblyVisible()
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if len(m.assemblies) > 0 && m.decompiler.Ready() {
			asm := m.assemblies[m.assemblyCursor]
			m.selectedAssembly = asm.Path
			m.types = nil
			m.source = nil
			m.sourceType = ""
			m.decompiling = true
			m.scanError = ""
			m.focus = focusTypes
			return m, m.doListTypes(asm.Path)
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("d"))):
		if len(m.assemblies) > 0 && m.decompiler.Ready() {
			asm := m.assemblies[m.assemblyCursor]
			m.selectedAssembly = asm.Path
			m.decompiling = true
			m.scanError = ""
			m.focus = focusSource
			return m, m.doDecompileAssembly(asm.Path)
		}
	}
	return m, nil
}

func (m *Model) handleTypeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.typeCursor > 0 {
			m.typeCursor--
			m.ensureTypeVisible()
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if m.typeCursor < len(m.types)-1 {
			m.typeCursor++
			m.ensureTypeVisible()
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if len(m.types) > 0 && m.selectedAssembly != "" {
			t := m.types[m.typeCursor]
			m.decompiling = true
			m.focus = focusSource
			return m, m.doDecompileType(m.selectedAssembly, t.FullName)
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		m.focus = focusAssemblies
	}
	return m, nil
}

func (m *Model) handleSourceKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	viewH := m.sourceViewHeight()
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.sourceScroll > 0 {
			m.sourceScroll--
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		maxScroll := len(m.source) - viewH
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.sourceScroll < maxScroll {
			m.sourceScroll++
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("pgup"))):
		m.sourceScroll -= viewH
		if m.sourceScroll < 0 {
			m.sourceScroll = 0
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("pgdown"))):
		maxScroll := len(m.source) - viewH
		if maxScroll < 0 {
			maxScroll = 0
		}
		m.sourceScroll += viewH
		if m.sourceScroll > maxScroll {
			m.sourceScroll = maxScroll
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		if len(m.types) > 0 {
			m.focus = focusTypes
		} else {
			m.focus = focusAssemblies
		}
	}
	return m, nil
}

func (m *Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchActive = false
		m.searching = false
		m.searchInput.Blur()
		m.focus = focusAssemblies
		return m, nil
	case "enter":
		if m.focus == focusSearch {
			// Submit search query
			query := strings.TrimSpace(m.searchInput.Value())
			if query != "" && !m.searching {
				m.searching = true
				m.searchResults = nil
				return m, m.doSearchAll(query)
			}
			return m, nil
		}
		// Enter on a result — navigate to it
		if len(m.searchResults) > 0 {
			r := m.searchResults[m.searchCursor]
			m.searchActive = false
			m.searchInput.Blur()
			m.selectedAssembly = r.asmPath
			m.targetScrollLine = r.line
			m.decompiling = true
			m.focus = focusSource
			return m, m.doDecompileAssembly(r.asmPath)
		}
		return m, nil
	case "up", "k":
		if m.focus != focusSearch && m.searchCursor > 0 {
			m.searchCursor--
			m.ensureSearchVisible()
		}
		return m, nil
	case "down", "j":
		if m.focus != focusSearch && m.searchCursor < len(m.searchResults)-1 {
			m.searchCursor++
			m.ensureSearchVisible()
		}
		return m, nil
	case "tab":
		if m.focus == focusSearch && len(m.searchResults) > 0 {
			m.focus = focusAssemblies // reuse as "results navigation" focus
			m.searchInput.Blur()
		} else {
			m.focus = focusSearch
			m.searchInput.Focus()
			return m, m.searchInput.Cursor.BlinkCmd()
		}
		return m, nil
	case "pgup":
		if m.focus != focusSearch {
			ch := m.contentHeight()
			m.searchCursor -= ch
			if m.searchCursor < 0 {
				m.searchCursor = 0
			}
			m.ensureSearchVisible()
		}
		return m, nil
	case "pgdown":
		if m.focus != focusSearch {
			ch := m.contentHeight()
			m.searchCursor += ch
			if m.searchCursor >= len(m.searchResults) {
				m.searchCursor = len(m.searchResults) - 1
			}
			if m.searchCursor < 0 {
				m.searchCursor = 0
			}
			m.ensureSearchVisible()
		}
		return m, nil
	}

	// Forward other keys to the text input when it's focused
	if m.focus == focusSearch {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) ensureSearchVisible() {
	ch := m.contentHeight()
	if m.searchCursor < m.searchScroll {
		m.searchScroll = m.searchCursor
	}
	if m.searchCursor >= m.searchScroll+ch {
		m.searchScroll = m.searchCursor - ch + 1
	}
}

// ---------------------------------------------------------------------------
// Focus helpers
// ---------------------------------------------------------------------------

func (m *Model) cycleFocusForward() {
	switch m.focus {
	case focusAssemblies:
		if len(m.types) > 0 {
			m.focus = focusTypes
		} else if len(m.source) > 0 {
			m.focus = focusSource
		}
	case focusTypes:
		if len(m.source) > 0 {
			m.focus = focusSource
		} else {
			m.focus = focusAssemblies
		}
	case focusSource:
		m.focus = focusAssemblies
	}
}

func (m *Model) cycleFocusBackward() {
	switch m.focus {
	case focusAssemblies:
		if len(m.source) > 0 {
			m.focus = focusSource
		} else if len(m.types) > 0 {
			m.focus = focusTypes
		}
	case focusTypes:
		m.focus = focusAssemblies
	case focusSource:
		if len(m.types) > 0 {
			m.focus = focusTypes
		} else {
			m.focus = focusAssemblies
		}
	}
}

// ---------------------------------------------------------------------------
// Scroll helpers
// ---------------------------------------------------------------------------

func (m *Model) contentHeight() int {
	h := m.height - 3
	if h < 1 {
		h = 1
	}
	return h
}

func (m *Model) sourceViewHeight() int {
	h := m.height - 4
	if h < 1 {
		h = 1
	}
	return h
}

func (m *Model) ensureAssemblyVisible() {
	ch := m.contentHeight()
	if m.assemblyCursor < m.assemblyScroll {
		m.assemblyScroll = m.assemblyCursor
	}
	if m.assemblyCursor >= m.assemblyScroll+ch {
		m.assemblyScroll = m.assemblyCursor - ch + 1
	}
}

func (m *Model) ensureTypeVisible() {
	ch := m.contentHeight()
	if m.typeCursor < m.typeScroll {
		m.typeScroll = m.typeCursor
	}
	if m.typeCursor >= m.typeScroll+ch {
		m.typeScroll = m.typeCursor - ch + 1
	}
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m *Model) View() string {
	if m.projectPath == "" {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(app.ColorOverlay1).Render(
				"Open a project to browse decompiled game code\n\n"+
					app.StyleDim.Render("Alt+1 to switch to Project panel")))
	}

	if !m.decompiler.Available() {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Center,
				app.StyleWarning.Render("dotnet CLI not found"),
				"",
				lipgloss.NewStyle().Foreground(app.ColorText).Render("Install the .NET SDK:"),
				app.StyleInfo.Render("https://dot.net/download"),
				"",
				app.StyleDim.Render("The decompiler requires the .NET SDK to build"),
				app.StyleDim.Render("a helper tool using ICSharpCode.Decompiler."),
			))
	}

	if m.buildingHelper {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Center,
				lipgloss.NewStyle().Foreground(app.ColorMauve).Render("Building decompiler helper..."),
				"",
				app.StyleDim.Render("First run: downloading ICSharpCode.Decompiler from NuGet."),
				app.StyleDim.Render("This only happens once."),
			))
	}

	if m.scanError != "" {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			app.StyleError.Render(m.scanError))
	}

	if m.searchActive {
		return m.viewSearch()
	}

	// Three-column layout
	sourceWidth := m.width - m.assemblyWidth - m.typeWidth - 2
	if sourceWidth < minSourceWidth {
		sourceWidth = minSourceWidth
	}

	asmCol := m.viewAssemblyColumn()
	typeCol := m.viewTypeColumn()
	srcCol := m.viewSourceColumn(sourceWidth)

	border := m.viewVerticalBorder()

	return lipgloss.JoinHorizontal(lipgloss.Top,
		asmCol,
		border,
		typeCol,
		border,
		srcCol,
	)
}

func (m *Model) viewVerticalBorder() string {
	borderStyle := lipgloss.NewStyle().Foreground(app.ColorSurface1)
	lines := make([]string, m.height)
	for i := range lines {
		lines[i] = "\u2502"
	}
	return borderStyle.Render(strings.Join(lines, "\n"))
}

func (m *Model) viewAssemblyColumn() string {
	w := m.assemblyWidth
	titleStyle := app.StyleTitle.Width(w)
	if m.focus == focusAssemblies {
		titleStyle = titleStyle.Underline(true)
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Assemblies"))
	b.WriteString("\n")
	b.WriteString(app.HRule(w))
	b.WriteString("\n")

	ch := m.contentHeight()
	normalStyle := lipgloss.NewStyle().Foreground(app.ColorText).Width(w)
	selectedStyle := lipgloss.NewStyle().Foreground(app.ColorText).Background(app.ColorSurface0).Width(w)
	dimStyle := lipgloss.NewStyle().Foreground(app.ColorSurface2).Width(w)

	if m.scanning {
		b.WriteString(lipgloss.NewStyle().Foreground(app.ColorMauve).Width(w).Render("Scanning..."))
		b.WriteString("\n")
	} else if len(m.assemblies) == 0 {
		b.WriteString(dimStyle.Render("No assemblies found"))
		b.WriteString("\n")
	}

	end := m.assemblyScroll + ch
	if end > len(m.assemblies) {
		end = len(m.assemblies)
	}

	lines := 0
	for i := m.assemblyScroll; i < end && lines < ch; i++ {
		asm := m.assemblies[i]
		sizeStr := formatSize(asm.Size)
		nameMaxW := w - lipgloss.Width(sizeStr) - 4
		name := truncate(asm.Name, nameMaxW)

		style := normalStyle
		prefix := "  "
		if i == m.assemblyCursor {
			style = selectedStyle
			prefix = "> "
		}
		label := prefix + name
		gap := w - lipgloss.Width(label) - len(sizeStr)
		if gap < 1 {
			gap = 1
		}
		label += strings.Repeat(" ", gap) + sizeStr
		b.WriteString(style.Render(label))
		b.WriteString("\n")
		lines++
	}

	for lines < ch {
		b.WriteString(strings.Repeat(" ", w))
		b.WriteString("\n")
		lines++
	}

	return b.String()
}

func (m *Model) viewTypeColumn() string {
	w := m.typeWidth
	titleStyle := app.StyleTitle.Width(w)
	if m.focus == focusTypes {
		titleStyle = titleStyle.Underline(true)
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Types"))
	b.WriteString("\n")
	b.WriteString(app.HRule(w))
	b.WriteString("\n")

	ch := m.contentHeight()
	normalStyle := lipgloss.NewStyle().Foreground(app.ColorText).Width(w)
	selectedStyle := lipgloss.NewStyle().Foreground(app.ColorText).Background(app.ColorSurface0).Width(w)
	dimStyle := lipgloss.NewStyle().Foreground(app.ColorSurface2).Width(w)
	nsStyle := lipgloss.NewStyle().Foreground(app.ColorOverlay1)

	if m.decompiling && len(m.types) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(app.ColorMauve).Width(w).Render("Loading types..."))
		b.WriteString("\n")
	} else if len(m.types) == 0 {
		b.WriteString(dimStyle.Render("Select an assembly"))
		b.WriteString("\n")
	}

	end := m.typeScroll + ch
	if end > len(m.types) {
		end = len(m.types)
	}

	lines := 0
	for i := m.typeScroll; i < end && lines < ch; i++ {
		t := m.types[i]

		style := normalStyle
		prefix := "  "
		if i == m.typeCursor {
			style = selectedStyle
			prefix = "> "
		}

		display := prefix
		if t.Namespace != "" {
			nsStr := truncate(t.Namespace+".", w-len(prefix)-len(t.Name)-1)
			display += nsStyle.Render(nsStr) + t.Name
		} else {
			display += t.Name
		}
		b.WriteString(style.Render(truncate(display, w)))
		b.WriteString("\n")
		lines++
	}

	for lines < ch {
		b.WriteString(strings.Repeat(" ", w))
		b.WriteString("\n")
		lines++
	}

	return b.String()
}

func (m *Model) viewSourceColumn(w int) string {
	titleStyle := app.StyleTitle.Width(w)
	if m.focus == focusSource {
		titleStyle = titleStyle.Underline(true)
	}

	var b strings.Builder

	title := "Source"
	if m.sourceType != "" {
		title = "Source: " + m.sourceType
	}
	b.WriteString(titleStyle.Render(truncate(title, w)))
	b.WriteString("\n")
	b.WriteString(app.HRule(w))
	b.WriteString("\n")

	ch := m.sourceViewHeight()
	dimStyle := lipgloss.NewStyle().Foreground(app.ColorSurface2)
	lineNumStyle := lipgloss.NewStyle().Foreground(app.ColorOverlay1)

	if m.decompiling && len(m.source) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(app.ColorMauve).Width(w).Render("Decompiling..."))
		b.WriteString("\n")
	} else if len(m.source) == 0 {
		b.WriteString(dimStyle.Width(w).Render("Select a type to decompile"))
		b.WriteString("\n")
		b.WriteString("\n")
		b.WriteString(dimStyle.Width(w).Render("  [enter] List types  [d] Decompile full assembly"))
		b.WriteString("\n")
	}

	end := m.sourceScroll + ch
	if end > len(m.source) {
		end = len(m.source)
	}

	gutterWidth := len(fmt.Sprintf("%d", len(m.source))) + 1
	if gutterWidth < 4 {
		gutterWidth = 4
	}
	codeWidth := w - gutterWidth - 1
	if codeWidth < 10 {
		codeWidth = 10
	}

	lines := 0
	for i := m.sourceScroll; i < end && lines < ch; i++ {
		lineNum := fmt.Sprintf("%*d", gutterWidth, i+1)
		code := m.source[i]

		highlighted := editor.HighlightLine(code, editor.LangCSharp)

		if lipgloss.Width(code) > codeWidth {
			code = truncateRaw(code, codeWidth)
			highlighted = editor.HighlightLine(code, editor.LangCSharp)
		}

		line := lineNumStyle.Render(lineNum) + " " + highlighted
		b.WriteString(line)
		b.WriteString("\n")
		lines++
	}

	for lines < ch {
		b.WriteString(strings.Repeat(" ", w))
		b.WriteString("\n")
		lines++
	}

	if len(m.source) > ch {
		scrollInfo := fmt.Sprintf("[%d-%d of %d lines]", m.sourceScroll+1, end, len(m.source))
		b.WriteString(dimStyle.Render(scrollInfo))
	}

	return b.String()
}

func (m *Model) viewSearch() string {
	w := m.width

	var b strings.Builder

	// Search bar
	labelStyle := lipgloss.NewStyle().Foreground(app.ColorMauve).Bold(true)
	label := labelStyle.Render("Search: ")
	inputView := m.searchInput.View()

	countStr := ""
	if m.searching {
		countStr = lipgloss.NewStyle().Foreground(app.ColorMauve).Render("  Searching...")
	} else if m.searchResults != nil {
		countStr = lipgloss.NewStyle().Foreground(app.ColorOverlay1).
			Render(fmt.Sprintf("  %d results", len(m.searchResults)))
	}

	b.WriteString(label + inputView + countStr)
	b.WriteString("\n")
	b.WriteString(app.HRule(w))
	b.WriteString("\n")

	ch := m.contentHeight()
	dimStyle := lipgloss.NewStyle().Foreground(app.ColorSurface2)
	normalStyle := lipgloss.NewStyle().Foreground(app.ColorText)
	selectedStyle := lipgloss.NewStyle().Foreground(app.ColorText).Background(app.ColorSurface0)
	asmStyle := lipgloss.NewStyle().Foreground(app.ColorOverlay1)
	lineNumStyle := lipgloss.NewStyle().Foreground(app.ColorYellow)

	if m.searching {
		b.WriteString(lipgloss.NewStyle().Foreground(app.ColorMauve).Width(w).
			Render("Decompiling and searching all assemblies..."))
		b.WriteString("\n")
		b.WriteString(dimStyle.Width(w).Render("This may take a moment."))
		b.WriteString("\n")
	} else if m.searchResults != nil && len(m.searchResults) == 0 {
		b.WriteString(dimStyle.Width(w).Render("No results found."))
		b.WriteString("\n")
	} else if m.searchResults == nil {
		b.WriteString(dimStyle.Width(w).Render("Type a query and press Enter to search."))
		b.WriteString("\n")
		b.WriteString("\n")
		b.WriteString(dimStyle.Width(w).Render("  Searches decompiled C# source across all assemblies."))
		b.WriteString("\n")
	}

	if len(m.searchResults) > 0 {
		end := m.searchScroll + ch
		if end > len(m.searchResults) {
			end = len(m.searchResults)
		}

		// Compute column widths: assembly name and line number
		maxAsmLen := 0
		for _, r := range m.searchResults {
			if len(r.assembly) > maxAsmLen {
				maxAsmLen = len(r.assembly)
			}
		}
		if maxAsmLen > 30 {
			maxAsmLen = 30
		}
		lines := 0
		for i := m.searchScroll; i < end && lines < ch; i++ {
			r := m.searchResults[i]

			prefix := "  "
			style := normalStyle
			if i == m.searchCursor {
				prefix = "> "
				style = selectedStyle
			}

			asmName := truncate(r.assembly, maxAsmLen)
			lineStr := fmt.Sprintf(":%d", r.line)

			contentW := w - len(prefix) - maxAsmLen - len(lineStr) - 2
			if contentW < 10 {
				contentW = 10
			}
			content := truncate(r.content, contentW)

			entry := prefix +
				asmStyle.Render(fmt.Sprintf("%-*s", maxAsmLen, asmName)) +
				lineNumStyle.Render(lineStr) + " " +
				style.Render(content)

			b.WriteString(truncate(entry, w))
			b.WriteString("\n")
			lines++
		}

		for lines < ch {
			b.WriteString("\n")
			lines++
		}
	}

	return b.String()
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	visWidth := lipgloss.Width(s)
	if visWidth <= maxLen {
		return s
	}
	if maxLen <= 3 {
		runes := []rune(s)
		if len(runes) > maxLen {
			return string(runes[:maxLen])
		}
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > maxLen-3 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "..."
}

func truncateRaw(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024*1024:
		return fmt.Sprintf("%.1fG", float64(bytes)/(1024*1024*1024))
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1fM", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.0fK", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
