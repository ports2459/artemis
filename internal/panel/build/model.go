package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eden/artemis/internal/app"
	buildlib "github.com/eden/artemis/internal/build"
	"github.com/eden/artemis/internal/config"
	"github.com/eden/artemis/internal/panel"
)

// Model is the build & deploy panel.
type Model struct {
	width         int
	height        int
	runner        *buildlib.Runner
	projectPath   string
	projectConfig *config.ProjectConfig

	output       []buildlib.OutputLine
	scrollOffset int
	building     bool
	deployAfter  bool
	lastResult   *buildlib.Result
}

// Compile-time check: *Model satisfies panel.Panel.
var _ panel.Panel = (*Model)(nil)

// NewModel creates a new build panel.
func NewModel() Model {
	return Model{
		runner: buildlib.NewRunner(),
	}
}

// ---------------------------------------------------------------------------
// panel.Panel interface
// ---------------------------------------------------------------------------

func (m *Model) ID() panel.PanelID { return panel.PanelBuild }

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) StatusInfo() string {
	if m.building {
		return "building..."
	}
	if m.lastResult != nil {
		return m.lastResult.Status.String()
	}
	return "idle"
}

// ---------------------------------------------------------------------------
// tea.Model interface
// ---------------------------------------------------------------------------

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case app.OpenProjectMsg:
		return m.handleOpenProject(msg)

	case buildlib.BuildOutputMsg:
		m.output = append(m.output, msg.Line)
		m.autoScroll()
		return m, nil

	case buildlib.BuildCompleteMsg:
		m.building = false
		m.lastResult = &msg.Result
		m.output = m.runner.Output()

		if m.deployAfter && msg.Result.Status == buildlib.StatusSuccess {
			m.deployAfter = false
			return m, m.doDeploy()
		}
		m.deployAfter = false
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) handleOpenProject(msg app.OpenProjectMsg) (tea.Model, tea.Cmd) {
	m.projectPath = msg.Path
	cfgPath := filepath.Join(msg.Path, "artemis.toml")
	cfg, err := config.LoadProject(cfgPath)
	if err == nil {
		m.projectConfig = &cfg
		configuration := cfg.Build.Configuration
		if configuration == "" {
			configuration = "Debug"
		}
		m.runner.SetProject(msg.Path, configuration)
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.building {
		// Only allow cancel while building.
		if key.Matches(msg, key.NewBinding(key.WithKeys("esc"))) {
			m.runner.Cancel()
			m.building = false
			return m, nil
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("b"))):
		if m.projectConfig == nil {
			return m, nil
		}
		m.building = true
		m.deployAfter = false
		m.output = nil
		m.scrollOffset = 0
		return m, m.runner.Build()

	case key.Matches(msg, key.NewBinding(key.WithKeys("d"))):
		if m.projectConfig == nil {
			return m, nil
		}
		m.building = true
		m.deployAfter = true
		m.output = nil
		m.scrollOffset = 0
		return m, m.runner.Build()

	case key.Matches(msg, key.NewBinding(key.WithKeys("c"))):
		if m.projectPath == "" {
			return m, nil
		}
		return m, m.doClean()

	case key.Matches(msg, key.NewBinding(key.WithKeys("up"))):
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("down"))):
		maxScroll := m.maxScroll()
		if m.scrollOffset < maxScroll {
			m.scrollOffset++
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("pgup"))):
		m.scrollOffset -= m.outputViewHeight()
		if m.scrollOffset < 0 {
			m.scrollOffset = 0
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("pgdown"))):
		maxScroll := m.maxScroll()
		m.scrollOffset += m.outputViewHeight()
		if m.scrollOffset > maxScroll {
			m.scrollOffset = maxScroll
		}
	}

	return m, nil
}

// ---------------------------------------------------------------------------
// Actions
// ---------------------------------------------------------------------------

func (m *Model) doDeploy() tea.Cmd {
	if m.projectConfig == nil {
		return nil
	}
	cfg := m.projectConfig
	dllName := cfg.Project.Name + ".dll"
	configuration := cfg.Build.Configuration
	if configuration == "" {
		configuration = "Debug"
	}
	dllPath := filepath.Join(m.projectPath, "bin", configuration, dllName)
	deployDir := cfg.Game.DeployDir

	return func() tea.Msg {
		runner := buildlib.NewRunner()
		if err := runner.Deploy(dllPath, deployDir); err != nil {
			return app.StatusMsg{Text: fmt.Sprintf("Deploy failed: %v", err)}
		}
		return app.StatusMsg{Text: fmt.Sprintf("Deployed %s to %s", dllName, deployDir)}
	}
}

func (m *Model) doClean() tea.Cmd {
	projectPath := m.projectPath
	return func() tea.Msg {
		for _, dir := range []string{"bin", "obj"} {
			target := filepath.Join(projectPath, dir)
			_ = os.RemoveAll(target)
		}
		return app.StatusMsg{Text: "Clean complete"}
	}
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var sections []string
	sections = append(sections, m.viewConfig())
	sections = append(sections, m.viewActions())
	sections = append(sections, m.viewOutput())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m *Model) viewConfig() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + app.StyleTitle.Render("BUILD CONFIGURATION") + "\n")
	b.WriteString("  " + app.HRule(m.width-4) + "\n")

	labelStyle := app.StyleDim
	valueStyle := lipgloss.NewStyle().Foreground(app.ColorText)

	if m.projectConfig != nil {
		cfg := m.projectConfig
		framework := cfg.Framework.Type
		if cfg.Framework.Version != "" {
			framework += " " + cfg.Framework.Version
		}
		if framework == "" {
			framework = "unknown"
		}

		gamePath := cfg.Game.Path
		if gamePath == "" {
			gamePath = "not set"
		}

		configuration := cfg.Build.Configuration
		if configuration == "" {
			configuration = "Debug"
		}
		outputPath := filepath.Join("bin", configuration, cfg.Project.Name+".dll")

		deployDir := cfg.Game.DeployDir
		if deployDir == "" {
			deployDir = "not set"
		}

		b.WriteString(fmt.Sprintf("    %s  %s\n",
			labelStyle.Render("Framework:"), valueStyle.Render(framework)))
		b.WriteString(fmt.Sprintf("    %s  %s\n",
			labelStyle.Render("Game Path:"), valueStyle.Render(gamePath)))
		b.WriteString(fmt.Sprintf("    %s     %s\n",
			labelStyle.Render("Output:"), valueStyle.Render(outputPath)))
		b.WriteString(fmt.Sprintf("    %s  %s\n",
			labelStyle.Render("Deploy To:"), valueStyle.Render(deployDir)))
	} else {
		b.WriteString("    " + labelStyle.Render("No project loaded") + "\n")
	}

	// Status indicator.
	statusText := m.statusIndicator()
	b.WriteString(fmt.Sprintf("    %s     %s\n",
		labelStyle.Render("Status:"), statusText))

	return b.String()
}

func (m *Model) statusIndicator() string {
	if m.building {
		return app.StyleWarning.Render("● Building...")
	}
	if m.lastResult != nil {
		switch m.lastResult.Status {
		case buildlib.StatusSuccess:
			return app.StyleSuccess.Render("● Build succeeded")
		case buildlib.StatusFailed:
			return app.StyleError.Render("● Build failed")
		}
	}
	return app.StyleDim.Render("● Idle")
}

func (m *Model) viewActions() string {
	var b strings.Builder
	b.WriteString("\n")

	if m.building {
		b.WriteString("    " + app.StyleWarning.Render("[esc] Cancel") + "\n")
	} else {
		buildKey := app.StyleInfo.Render("[b]") + " Build"
		deployKey := app.StyleInfo.Render("[d]") + " Build & Deploy"
		cleanKey := app.StyleInfo.Render("[c]") + " Clean"
		b.WriteString(fmt.Sprintf("    %s   %s   %s\n", buildKey, deployKey, cleanKey))
	}

	return b.String()
}

func (m *Model) viewOutput() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + app.StyleTitle.Render("OUTPUT") + "\n")
	b.WriteString("  " + app.HRule(m.width-4) + "\n")

	if len(m.output) == 0 && !m.building {
		if m.lastResult == nil {
			b.WriteString("    " + app.StyleDim.Render("No build output yet. Press [b] to build.") + "\n")
		}
		return b.String()
	}

	viewHeight := m.outputViewHeight()
	if viewHeight < 1 {
		viewHeight = 1
	}

	end := m.scrollOffset + viewHeight
	if end > len(m.output) {
		end = len(m.output)
	}
	start := m.scrollOffset
	if start > len(m.output) {
		start = len(m.output)
	}

	for i := start; i < end; i++ {
		line := m.output[i]
		styled := m.styleLine(line)
		b.WriteString("    " + styled + "\n")
	}

	// Build summary after completion.
	if !m.building && m.lastResult != nil {
		b.WriteString("\n")
		res := m.lastResult
		duration := fmt.Sprintf("%.1fs", res.Duration.Seconds())
		if res.Status == buildlib.StatusSuccess {
			b.WriteString("    " + app.StyleSuccess.Render(fmt.Sprintf("Done in %s", duration)) + "\n")
		} else {
			b.WriteString("    " + app.StyleError.Render(fmt.Sprintf("Failed in %s", duration)) + "\n")
		}
		if res.Errors > 0 || res.Warnings > 0 {
			summary := fmt.Sprintf("    %d error(s), %d warning(s)", res.Errors, res.Warnings)
			b.WriteString("    " + app.StyleDim.Render(summary) + "\n")
		}
	}

	// Scroll indicator.
	if len(m.output) > viewHeight {
		scrollInfo := fmt.Sprintf("[%d-%d of %d lines]", start+1, end, len(m.output))
		b.WriteString("    " + app.StyleDim.Render(scrollInfo) + "\n")
	}

	return b.String()
}

func (m *Model) styleLine(line buildlib.OutputLine) string {
	if line.IsError {
		return app.StyleError.Render(line.Text)
	}
	if line.IsInfo {
		return app.StyleInfo.Render(line.Text)
	}
	return app.StyleDim.Render(line.Text)
}

// outputViewHeight returns how many output lines fit in the panel.
// Config section takes about 10 lines, actions 3 lines, output header 3 lines,
// summary ~4 lines. Reserve roughly 20 lines for chrome.
func (m *Model) outputViewHeight() int {
	h := m.height - 20
	if h < 3 {
		h = 3
	}
	return h
}

func (m *Model) maxScroll() int {
	max := len(m.output) - m.outputViewHeight()
	if max < 0 {
		return 0
	}
	return max
}

func (m *Model) autoScroll() {
	maxScroll := m.maxScroll()
	if m.scrollOffset >= maxScroll-1 {
		m.scrollOffset = maxScroll
	}
}
