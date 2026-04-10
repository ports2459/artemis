package project

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eden/artemis/internal/app"
	"github.com/eden/artemis/internal/config"
	"github.com/eden/artemis/internal/panel"
)

type state int

const (
	stateList state = iota
	stateNewName
	stateNewTemplate
	stateNewGamePath
)

// ProjectEntry represents a discovered project on disk.
type ProjectEntry struct {
	Name      string
	Path      string
	Framework string
	GamePath  string
	Modified  time.Time
}

// Model is the project manager panel. It lets the user browse existing projects
// and create new ones from templates.
type Model struct {
	width        int
	height       int
	projects     []ProjectEntry
	cursor       int
	state        state
	nameInput    textinput.Model
	gameInput    textinput.Model
	templateIdx  int
	workspaceDir string
}

// Compile-time check: *Model satisfies panel.Panel.
var _ panel.Panel = (*Model)(nil)

var templates = []TemplateType{TemplateBlank, TemplateBepInEx, TemplateMelonLoader}

// NewModel creates a new project manager panel that scans workspaceDir for
// existing projects.
func NewModel(workspaceDir string) Model {
	ti := textinput.New()
	ti.Placeholder = "MyMod"
	ti.CharLimit = 64

	gi := textinput.New()
	gi.Placeholder = "/path/to/game"
	gi.CharLimit = 256

	m := Model{
		workspaceDir: workspaceDir,
		nameInput:    ti,
		gameInput:    gi,
		state:        stateList,
	}
	m.scanProjects()
	return m
}

// scanProjects reads the workspace directory and populates m.projects with
// every subdirectory that contains an artemis.toml.
func (m *Model) scanProjects() {
	m.projects = nil
	entries, err := os.ReadDir(m.workspaceDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		cfgPath := filepath.Join(m.workspaceDir, e.Name(), "artemis.toml")
		cfg, err := config.LoadProject(cfgPath)
		if err != nil {
			continue
		}
		info, _ := e.Info()
		modified := time.Time{}
		if info != nil {
			modified = info.ModTime()
		}
		m.projects = append(m.projects, ProjectEntry{
			Name:      cfg.Project.Name,
			Path:      filepath.Join(m.workspaceDir, e.Name()),
			Framework: cfg.Framework.Type + " " + cfg.Framework.Version,
			GamePath:  cfg.Game.Path,
			Modified:  modified,
		})
	}
	sort.Slice(m.projects, func(i, j int) bool {
		return m.projects[i].Modified.After(m.projects[j].Modified)
	})
}

// ---------------------------------------------------------------------------
// panel.Panel interface
// ---------------------------------------------------------------------------

func (m *Model) ID() panel.PanelID { return panel.PanelProject }

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Model) StatusInfo() string {
	return fmt.Sprintf("%d projects", len(m.projects))
}

// ---------------------------------------------------------------------------
// tea.Model interface
// ---------------------------------------------------------------------------

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	if m.state == stateNewName {
		var cmd tea.Cmd
		m.nameInput, cmd = m.nameInput.Update(msg)
		return m, cmd
	}
	if m.state == stateNewGamePath {
		var cmd tea.Cmd
		m.gameInput, cmd = m.gameInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateList:
		return m.handleListKey(msg)
	case stateNewName:
		return m.handleNameKey(msg)
	case stateNewTemplate:
		return m.handleTemplateKey(msg)
	case stateNewGamePath:
		return m.handleGamePathKey(msg)
	}
	return m, nil
}

func (m *Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	totalItems := len(m.projects) + 1 // projects + "New Project" entry
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up"))):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("down"))):
		if m.cursor < totalItems-1 {
			m.cursor++
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if m.cursor == len(m.projects) {
			// "New Project" selected
			m.state = stateNewName
			m.nameInput.SetValue("")
			m.nameInput.Focus()
			return m, m.nameInput.Cursor.BlinkCmd()
		}
		path := m.projects[m.cursor].Path
		return m, func() tea.Msg {
			return app.OpenProjectMsg{Path: path}
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
		m.state = stateNewName
		m.nameInput.SetValue("")
		m.nameInput.Focus()
		return m, m.nameInput.Cursor.BlinkCmd()
	case key.Matches(msg, key.NewBinding(key.WithKeys("q"))):
		if m.cursor < len(m.projects) {
			p := m.projects[m.cursor]
			if err := os.RemoveAll(p.Path); err != nil {
				return m, func() tea.Msg {
					return app.StatusMsg{Text: fmt.Sprintf("Delete failed: %v", err)}
				}
			}
			m.scanProjects()
			if m.cursor >= len(m.projects) && m.cursor > 0 {
				m.cursor--
			}
			return m, func() tea.Msg {
				return app.StatusMsg{Text: fmt.Sprintf("Deleted %s", p.Name)}
			}
		}
	}
	return m, nil
}

func (m *Model) handleNameKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.nameInput.Value())
		if name == "" {
			return m, nil
		}
		m.state = stateNewTemplate
		m.templateIdx = 0
		return m, nil
	case "esc":
		m.state = stateList
		m.nameInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m *Model) handleTemplateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up"))):
		if m.templateIdx > 0 {
			m.templateIdx--
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("down"))):
		if m.templateIdx < len(templates)-1 {
			m.templateIdx++
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		m.state = stateNewGamePath
		m.gameInput.SetValue("")
		m.gameInput.Focus()
		return m, m.gameInput.Cursor.BlinkCmd()
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		m.state = stateNewName
		return m, nil
	}
	return m, nil
}

func (m *Model) handleGamePathKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		gamePath := strings.TrimSpace(m.gameInput.Value())
		name := strings.TrimSpace(m.nameInput.Value())
		dir := filepath.Join(m.workspaceDir, name)
		tmpl := templates[m.templateIdx]

		if err := Scaffold(dir, name, tmpl); err != nil {
			m.state = stateList
			return m, func() tea.Msg {
				return app.StatusMsg{Text: fmt.Sprintf("Error: %v", err)}
			}
		}

		// Update the project config with game path
		if gamePath != "" {
			cfgPath := filepath.Join(dir, "artemis.toml")
			cfg, err := config.LoadProject(cfgPath)
			if err == nil {
				cfg.Game.Path = gamePath
				cfg.Game.DeployDir = defaultDeployDir(tmpl)
				_ = cfg.Save(cfgPath)
			}
		}

		m.scanProjects()
		m.state = stateList
		m.gameInput.Blur()
		m.cursor = 0
		return m, func() tea.Msg {
			return app.OpenProjectMsg{Path: dir}
		}
	case "esc":
		m.state = stateNewTemplate
		m.gameInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.gameInput, cmd = m.gameInput.Update(msg)
	return m, cmd
}

func defaultDeployDir(tmpl TemplateType) string {
	switch tmpl {
	case TemplateBepInEx:
		return "BepInEx/plugins"
	case TemplateMelonLoader:
		return "Mods"
	default:
		return ""
	}
}

// ---------------------------------------------------------------------------
// Views
// ---------------------------------------------------------------------------

func (m *Model) View() string {
	switch m.state {
	case stateNewName:
		return m.viewNewName()
	case stateNewTemplate:
		return m.viewNewTemplate()
	case stateNewGamePath:
		return m.viewNewGamePath()
	default:
		return m.viewList()
	}
}

func (m *Model) viewList() string {
	titleStyle := lipgloss.NewStyle().Foreground(app.ColorMauve).Bold(true)
	subtitleStyle := lipgloss.NewStyle().Foreground(app.ColorSurface2)
	selectedStyle := lipgloss.NewStyle().Foreground(app.ColorText).Background(app.ColorSurface0)
	normalStyle := lipgloss.NewStyle().Foreground(app.ColorText)
	dimStyle := lipgloss.NewStyle().Foreground(app.ColorSurface2)
	greenStyle := lipgloss.NewStyle().Foreground(app.ColorGreen)
	borderStyle := lipgloss.NewStyle().Foreground(app.ColorSurface1)

	var b strings.Builder

	title := titleStyle.Render("Welcome to Artemis")
	subtitle := subtitleStyle.Render("Select or create a project")

	b.WriteString("\n\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, title))
	b.WriteString("\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, subtitle))
	b.WriteString("\n\n")

	boxWidth := 48
	top := borderStyle.Render("\u250c" + strings.Repeat("\u2500", boxWidth) + "\u2510")
	bot := borderStyle.Render("\u2514" + strings.Repeat("\u2500", boxWidth) + "\u2518")
	sep := borderStyle.Render("\u251c" + strings.Repeat("\u2500", boxWidth) + "\u2524")
	side := borderStyle.Render("\u2502")

	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, top))
	b.WriteString("\n")

	for i, p := range m.projects {
		style := normalStyle
		if i == m.cursor {
			style = selectedStyle
		}

		nameLine := style.Render(padRight(p.Name, boxWidth))
		desc := p.Framework
		if p.GamePath != "" {
			desc += " │ " + filepath.Base(p.GamePath)
		}
		descLine := dimStyle.Render(padRight(desc, boxWidth))

		b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
			side+nameLine+side))
		b.WriteString("\n")
		b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
			side+descLine+side))
		b.WriteString("\n")
		b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, sep))
		b.WriteString("\n")
	}

	newStyle := greenStyle
	if m.cursor == len(m.projects) {
		newStyle = lipgloss.NewStyle().Foreground(app.ColorGreen).Background(app.ColorSurface0)
	}
	newLine := newStyle.Render(padRight("+ New Project", boxWidth))
	newDesc := dimStyle.Render(padRight("  Start from blank or template", boxWidth))
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
		side+newLine+side))
	b.WriteString("\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
		side+newDesc+side))
	b.WriteString("\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, bot))
	b.WriteString("\n")

	return b.String()
}

func (m *Model) viewNewName() string {
	titleStyle := lipgloss.NewStyle().Foreground(app.ColorMauve).Bold(true)
	subtitleStyle := lipgloss.NewStyle().Foreground(app.ColorSurface2)

	var b strings.Builder
	b.WriteString("\n\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
		titleStyle.Render("New Project")))
	b.WriteString("\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
		subtitleStyle.Render("Enter a name for your mod")))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
		m.nameInput.View()))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
		subtitleStyle.Render("Enter: confirm  Esc: cancel")))

	return b.String()
}

func (m *Model) viewNewTemplate() string {
	titleStyle := lipgloss.NewStyle().Foreground(app.ColorMauve).Bold(true)
	subtitleStyle := lipgloss.NewStyle().Foreground(app.ColorSurface2)
	selectedStyle := lipgloss.NewStyle().Foreground(app.ColorText).Background(app.ColorSurface0)
	normalStyle := lipgloss.NewStyle().Foreground(app.ColorSubtext0)

	var b strings.Builder
	b.WriteString("\n\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
		titleStyle.Render("Choose a Template")))
	b.WriteString("\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
		subtitleStyle.Render("Project: "+m.nameInput.Value())))
	b.WriteString("\n\n")

	for i, tmpl := range templates {
		style := normalStyle
		prefix := "  "
		if i == m.templateIdx {
			style = selectedStyle
			prefix = "> "
		}
		line := style.Render(prefix + tmpl.String())
		b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, line))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
		subtitleStyle.Render("Enter: create  Esc: back")))

	return b.String()
}

func (m *Model) viewNewGamePath() string {
	titleStyle := lipgloss.NewStyle().Foreground(app.ColorMauve).Bold(true)
	subtitleStyle := lipgloss.NewStyle().Foreground(app.ColorSurface2)
	dimStyle := lipgloss.NewStyle().Foreground(app.ColorOverlay1)

	var b strings.Builder
	b.WriteString("\n\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
		titleStyle.Render("Game Path")))
	b.WriteString("\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
		subtitleStyle.Render("Project: "+m.nameInput.Value()+" │ "+templates[m.templateIdx].String())))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
		dimStyle.Render("Path to the game's install directory")))
	b.WriteString("\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
		dimStyle.Render("Used for asset browsing and mod deployment")))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
		m.gameInput.View()))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
		subtitleStyle.Render("Enter: create project  Esc: back  (leave empty to skip)")))

	return b.String()
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		// Truncate by runes to avoid cutting multi-byte chars
		runes := []rune(s)
		for len(runes) > 0 && lipgloss.Width(string(runes)) > width {
			runes = runes[:len(runes)-1]
		}
		return string(runes)
	}
	return s + strings.Repeat(" ", width-w)
}
