package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eden/artemis/internal/config"
	"github.com/eden/artemis/internal/panel"
	"github.com/eden/artemis/internal/terminal"
)

const (
	topBarHeight    = 1
	tabBarHeight    = 1
	statusBarHeight = 1
	chromeHeight    = topBarHeight + tabBarHeight + statusBarHeight + 2 // +2 for horizontal rules
)

type Model struct {
	width       int
	height      int
	activePanel panel.PanelID
	panels      map[panel.PanelID]panel.Panel
	globalCfg   config.GlobalConfig
	projectCfg  *config.ProjectConfig
	caps        terminal.Capabilities
	statusMsg   string
}

func NewModel(globalCfg config.GlobalConfig, caps terminal.Capabilities, panels ...panel.Panel) Model {
	m := Model{
		activePanel: panel.PanelProject,
		panels:      make(map[panel.PanelID]panel.Panel),
		globalCfg:   globalCfg,
		caps:        caps,
	}
	for _, p := range panels {
		m.panels[p.ID()] = p
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		panelHeight := m.height - chromeHeight
		for id, p := range m.panels {
			p.SetSize(m.width, panelHeight)
			m.panels[id] = p
		}
		return m, nil

	case tea.KeyMsg:
		if cmd := m.handleGlobalKey(msg); cmd != nil {
			return m, cmd
		}

	case SwitchPanelMsg:
		m.activePanel = msg.Panel
		return m, nil

	case OpenProjectMsg:
		cfgPath := filepath.Join(msg.Path, "artemis.toml")
		projectCfg, err := config.LoadProject(cfgPath)
		if err != nil {
			m.statusMsg = fmt.Sprintf("Error loading project: %v", err)
			return m, nil
		}
		m.projectCfg = &projectCfg
		m.activePanel = panel.PanelEditor
		m.statusMsg = fmt.Sprintf("Opened %s", projectCfg.Project.Name)
		// Forward to all panels so they can initialize
		var cmds []tea.Cmd
		for id, p := range m.panels {
			updated, cmd := p.Update(msg)
			m.panels[id] = updated.(panel.Panel)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

	case StatusMsg:
		m.statusMsg = msg.Text
		return m, nil
	}

	// Key messages go to active panel only; everything else goes to all panels
	// (async results like scanDoneMsg may be for a non-active panel).
	switch msg.(type) {
	case tea.KeyMsg:
		if p, ok := m.panels[m.activePanel]; ok {
			updated, cmd := p.Update(msg)
			m.panels[m.activePanel] = updated.(panel.Panel)
			return m, cmd
		}
	default:
		var cmds []tea.Cmd
		for id, p := range m.panels {
			updated, cmd := p.Update(msg)
			m.panels[id] = updated.(panel.Panel)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
	}
	return m, nil
}

func (m Model) handleGlobalKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, globalKeys.Quit):
		return tea.Quit
	case key.Matches(msg, globalKeys.Panel1):
		m.activePanel = panel.PanelProject
		return func() tea.Msg { return SwitchPanelMsg{Panel: panel.PanelProject} }
	case key.Matches(msg, globalKeys.Panel2):
		m.activePanel = panel.PanelEditor
		return func() tea.Msg { return SwitchPanelMsg{Panel: panel.PanelEditor} }
	case key.Matches(msg, globalKeys.Panel3):
		m.activePanel = panel.PanelAssets
		return func() tea.Msg { return SwitchPanelMsg{Panel: panel.PanelAssets} }
	case key.Matches(msg, globalKeys.Panel4):
		m.activePanel = panel.PanelBuild
		return func() tea.Msg { return SwitchPanelMsg{Panel: panel.PanelBuild} }
	}
	return nil
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var sections []string
	sections = append(sections, m.viewTopBar())
	sections = append(sections, m.viewTabs())
	sections = append(sections, HRule(m.width))

	if p, ok := m.panels[m.activePanel]; ok {
		panelView := p.View()
		panelHeight := m.height - chromeHeight
		panelView = lipgloss.NewStyle().
			Width(m.width).
			Height(panelHeight).
			MaxHeight(panelHeight).
			Render(panelView)
		sections = append(sections, panelView)
	} else {
		placeholder := lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
			StyleDim.Render("Panel not available"))
		sections = append(sections, placeholder)
	}

	sections = append(sections, HRule(m.width))
	sections = append(sections, m.viewStatusBar())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) viewTopBar() string {
	left := StyleTitle.Render(" artemis")

	projectName := ""
	framework := ""
	if m.projectCfg != nil {
		projectName = m.projectCfg.Project.Name
		framework = m.projectCfg.Framework.Type + " " + m.projectCfg.Framework.Version
	}

	if projectName != "" {
		left += StyleDim.Render(" › ") + StyleTopBar.Render(projectName)
		if framework != "" {
			left += StyleDim.Render(" › ") + StyleDim.Render(framework)
		}
	}

	right := StyleDim.Render("Ctrl+P: Commands  ?: Help")

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	return StyleTopBar.Width(m.width).Render(
		left + strings.Repeat(" ", gap) + right)
}

func (m Model) viewTabs() string {
	tabs := []struct {
		id   panel.PanelID
		name string
		key  string
	}{
		{panel.PanelProject, "Project", "1"},
		{panel.PanelEditor, "Editor", "2"},
		{panel.PanelAssets, "Assets", "3"},
		{panel.PanelBuild, "Build", "4"},
	}

	var parts []string
	for _, t := range tabs {
		style := StyleTabInactive
		if t.id == m.activePanel {
			style = StyleTabActive
		}
		label := style.Render(t.name)
		parts = append(parts, " "+label+" ")
	}

	tabLine := strings.Join(parts, StyleDim.Render("|"))
	return lipgloss.NewStyle().
		Background(ColorCrust).
		Width(m.width).
		Render("  " + tabLine)
}

func (m Model) viewStatusBar() string {
	left := StyleTitle.Render(" ◆ ")

	if m.projectCfg != nil {
		left += StyleTopBar.Render(m.projectCfg.Project.Name)
	} else {
		left += StyleDim.Render("No project")
	}

	if p, ok := m.panels[m.activePanel]; ok {
		info := p.StatusInfo()
		if info != "" {
			left += StyleDim.Render(" | ") + StyleTopBar.Render(info)
		}
	}

	right := ""
	if m.statusMsg != "" {
		right = StyleTopBar.Render(m.statusMsg)
	}

	capsInfo := StyleSuccess.Render("● ") + StyleTopBar.Render(m.caps.ImageProtocol.String())
	right = capsInfo + "  " + right

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	return StyleStatusBar.Width(m.width).Render(
		left + strings.Repeat(" ", gap) + right)
}
