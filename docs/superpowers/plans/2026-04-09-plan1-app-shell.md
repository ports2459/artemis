# Plan 1: App Shell & Project Manager

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the foundational Artemis binary — the app shell that manages panels, keybindings, config, and the project manager panel for creating/opening mod projects.

**Architecture:** Single Bubbletea program with a root model that owns child panel models. The root handles global keybindings (panel switching, command palette, quit) and routes input to the focused panel. Config is two-tier TOML (global + per-project).

**Tech Stack:** Go 1.22+, Bubbletea v1, Lipgloss v1, Bubbles v1, BurntSushi/toml

**Terminal note:** Ctrl+number doesn't work in terminals. Panel switching uses Alt+1/2/3/4.

---

## File Structure

```
artemis/
├── main.go                          -- entry point, arg parsing, program init
├── go.mod
├── go.sum
├── internal/
│   ├── app/
│   │   ├── model.go                 -- root model (panel switching, layout, global keys)
│   │   ├── keys.go                  -- global keybinding definitions
│   │   ├── messages.go              -- shared message types between panels
│   │   └── theme.go                 -- Catppuccin Mocha color palette, shared styles
│   ├── config/
│   │   ├── config.go                -- global config struct + load/save
│   │   ├── project.go               -- per-project config (artemis.toml) struct + load/save
│   │   └── config_test.go           -- tests for config load/save
│   ├── panel/
│   │   ├── panel.go                 -- Panel interface definition
│   │   └── project/
│   │       ├── model.go             -- project manager panel model
│   │       ├── model_test.go        -- tests for project creation logic
│   │       └── templates.go         -- BepInEx/MelonLoader/blank scaffolding
│   └── terminal/
│       ├── caps.go                  -- terminal capability detection
│       └── caps_test.go             -- tests for capability parsing
```

---

### Task 1: Go Module & Dependencies

**Files:**
- Create: `go.mod`
- Create: `main.go`

- [ ] **Step 1: Initialize Go module**

Run:
```bash
cd /home/eden/artemis
go mod init github.com/eden/artemis
```

Expected: `go.mod` created with module path.

- [ ] **Step 2: Add dependencies**

Run:
```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/BurntSushi/toml@latest
```

Expected: `go.mod` and `go.sum` updated with dependencies.

- [ ] **Step 3: Create minimal main.go**

Create `main.go`:

```go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eden/artemis/internal/app"
)

func main() {
	m := app.NewModel()
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Create stub app model so it compiles**

Create `internal/app/model.go`:

```go
package app

import tea "github.com/charmbracelet/bubbletea"

type Model struct {
	width  int
	height int
}

func NewModel() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if msg.String() == "ctrl+q" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	return "Artemis — press Ctrl+Q to quit"
}
```

- [ ] **Step 5: Verify it builds and runs**

Run:
```bash
go mod tidy && go build -o artemis . && echo "BUILD OK"
```

Expected: Binary compiles, `BUILD OK` printed.

Run the binary briefly to verify it shows the alt screen and responds to Ctrl+Q:
```bash
echo "q" | timeout 2 ./artemis || true
```

- [ ] **Step 6: Commit**

```bash
git add main.go go.mod go.sum internal/app/model.go
git commit -m "feat: initialize Go module with Bubbletea skeleton"
```

---

### Task 2: Theme & Shared Styles

**Files:**
- Create: `internal/app/theme.go`

- [ ] **Step 1: Define Catppuccin Mocha palette and shared styles**

Create `internal/app/theme.go`:

```go
package app

import "github.com/charmbracelet/lipgloss"

// Catppuccin Mocha palette
var (
	ColorRosewater = lipgloss.Color("#f5e0dc")
	ColorFlamingo  = lipgloss.Color("#f2cdcd")
	ColorPink      = lipgloss.Color("#f5c2e7")
	ColorMauve     = lipgloss.Color("#cba6f7")
	ColorRed       = lipgloss.Color("#f38ba8")
	ColorMaroon    = lipgloss.Color("#eba0ac")
	ColorPeach     = lipgloss.Color("#fab387")
	ColorYellow    = lipgloss.Color("#f9e2af")
	ColorGreen     = lipgloss.Color("#a6e3a1")
	ColorTeal      = lipgloss.Color("#94e2d5")
	ColorSky       = lipgloss.Color("#89dceb")
	ColorSapphire  = lipgloss.Color("#74c7ec")
	ColorBlue      = lipgloss.Color("#89b4fa")
	ColorLavender  = lipgloss.Color("#b4befe")
	ColorText      = lipgloss.Color("#cdd6f4")
	ColorSubtext1  = lipgloss.Color("#bac2de")
	ColorSubtext0  = lipgloss.Color("#a6adc8")
	ColorOverlay2  = lipgloss.Color("#9399b2")
	ColorOverlay1  = lipgloss.Color("#7f849c")
	ColorOverlay0  = lipgloss.Color("#6c7086")
	ColorSurface2  = lipgloss.Color("#585b70")
	ColorSurface1  = lipgloss.Color("#45475a")
	ColorSurface0  = lipgloss.Color("#313244")
	ColorBase      = lipgloss.Color("#1e1e2e")
	ColorMantle    = lipgloss.Color("#181825")
	ColorCrust     = lipgloss.Color("#11111b")
)

// Shared styles
var (
	StyleTopBar = lipgloss.NewStyle().
			Background(ColorMantle).
			Foreground(ColorSubtext0)

	StyleStatusBar = lipgloss.NewStyle().
			Background(ColorMantle).
			Foreground(ColorSurface2)

	StyleTabActive = lipgloss.NewStyle().
			Foreground(ColorMauve).
			Bold(true)

	StyleTabInactive = lipgloss.NewStyle().
			Foreground(ColorSurface2)

	StyleBorder = lipgloss.NewStyle().
			Foreground(ColorSurface1)

	StyleTitle = lipgloss.NewStyle().
			Foreground(ColorMauve).
			Bold(true)

	StyleDim = lipgloss.NewStyle().
			Foreground(ColorSurface2)

	StyleError = lipgloss.NewStyle().
			Foreground(ColorRed)

	StyleWarning = lipgloss.NewStyle().
			Foreground(ColorPeach)

	StyleSuccess = lipgloss.NewStyle().
			Foreground(ColorGreen)

	StyleInfo = lipgloss.NewStyle().
			Foreground(ColorBlue)
)

// HRule returns a horizontal rule of the given width.
func HRule(width int) string {
	s := ""
	for i := 0; i < width; i++ {
		s += "─"
	}
	return StyleBorder.Render(s)
}
```

- [ ] **Step 2: Verify it compiles**

Run:
```bash
go build ./... && echo "BUILD OK"
```

Expected: `BUILD OK`

- [ ] **Step 3: Commit**

```bash
git add internal/app/theme.go
git commit -m "feat: add Catppuccin Mocha theme and shared styles"
```

---

### Task 3: Panel Interface & Messages

**Files:**
- Create: `internal/panel/panel.go`
- Create: `internal/app/messages.go`
- Create: `internal/app/keys.go`

- [ ] **Step 1: Define the Panel interface**

Create `internal/panel/panel.go`:

```go
package panel

import tea "github.com/charmbracelet/bubbletea"

// PanelID identifies which panel is active.
type PanelID int

const (
	PanelProject PanelID = iota
	PanelEditor
	PanelAssets
	PanelBuild
)

func (p PanelID) String() string {
	switch p {
	case PanelProject:
		return "Project"
	case PanelEditor:
		return "Editor"
	case PanelAssets:
		return "Assets"
	case PanelBuild:
		return "Build"
	default:
		return "Unknown"
	}
}

// Panel is the interface all panels implement.
type Panel interface {
	tea.Model
	// SetSize is called when the terminal resizes.
	// width and height are the available space for this panel (excluding top bar, tabs, status bar).
	SetSize(width, height int)
	// ID returns the panel's identifier.
	ID() PanelID
	// StatusInfo returns text displayed in the status bar from this panel.
	StatusInfo() string
}
```

- [ ] **Step 2: Define shared messages**

Create `internal/app/messages.go`:

```go
package app

import "github.com/eden/artemis/internal/panel"

// SwitchPanelMsg requests the app shell to switch to a different panel.
type SwitchPanelMsg struct {
	Panel panel.PanelID
}

// OpenProjectMsg requests opening a project at the given path.
type OpenProjectMsg struct {
	Path string
}

// StatusMsg updates the status bar with a temporary message.
type StatusMsg struct {
	Text string
}
```

- [ ] **Step 3: Define global keybindings**

Create `internal/app/keys.go`:

```go
package app

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Quit       key.Binding
	Panel1     key.Binding
	Panel2     key.Binding
	Panel3     key.Binding
	Panel4     key.Binding
	CmdPalette key.Binding
	Save       key.Binding
	CloseTab   key.Binding
}

var globalKeys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("ctrl+q"),
		key.WithHelp("ctrl+q", "quit"),
	),
	Panel1: key.NewBinding(
		key.WithKeys("alt+1"),
		key.WithHelp("alt+1", "project"),
	),
	Panel2: key.NewBinding(
		key.WithKeys("alt+2"),
		key.WithHelp("alt+2", "editor"),
	),
	Panel3: key.NewBinding(
		key.WithKeys("alt+3"),
		key.WithHelp("alt+3", "assets"),
	),
	Panel4: key.NewBinding(
		key.WithKeys("alt+4"),
		key.WithHelp("alt+4", "build"),
	),
	CmdPalette: key.NewBinding(
		key.WithKeys("ctrl+p"),
		key.WithHelp("ctrl+p", "commands"),
	),
	Save: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "save"),
	),
	CloseTab: key.NewBinding(
		key.WithKeys("ctrl+w"),
		key.WithHelp("ctrl+w", "close tab"),
	),
}
```

- [ ] **Step 4: Verify it compiles**

Run:
```bash
go build ./... && echo "BUILD OK"
```

Expected: `BUILD OK`

- [ ] **Step 5: Commit**

```bash
git add internal/panel/panel.go internal/app/messages.go internal/app/keys.go
git commit -m "feat: add Panel interface, shared messages, global keybindings"
```

---

### Task 4: Config Store

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/project.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write tests for global config**

Create `internal/config/config_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGlobalConfig_Defaults(t *testing.T) {
	// Loading from a nonexistent path should return defaults
	cfg, err := LoadGlobal("/tmp/artemis-test-nonexistent/config.toml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Editor.TabSize != 4 {
		t.Errorf("expected TabSize 4, got %d", cfg.Editor.TabSize)
	}
	if cfg.Terminal.ImageProtocol != "auto" {
		t.Errorf("expected ImageProtocol 'auto', got %s", cfg.Terminal.ImageProtocol)
	}
}

func TestSaveAndLoadGlobalConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := DefaultGlobal()
	cfg.Editor.TabSize = 2
	cfg.Terminal.ImageProtocol = "kitty"

	if err := cfg.Save(path); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := LoadGlobal(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.Editor.TabSize != 2 {
		t.Errorf("expected TabSize 2, got %d", loaded.Editor.TabSize)
	}
	if loaded.Terminal.ImageProtocol != "kitty" {
		t.Errorf("expected 'kitty', got %s", loaded.Terminal.ImageProtocol)
	}
}

func TestLoadProjectConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "artemis.toml")

	content := []byte(`[project]
name = "TestMod"
version = "1.0.0"
author = "tester"

[framework]
type = "bepinex"
version = "5.4.22"

[game]
path = "/home/user/games/TestGame"
deploy_dir = "BepInEx/plugins"

[build]
configuration = "Debug"
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadProject(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.Project.Name != "TestMod" {
		t.Errorf("expected 'TestMod', got %s", cfg.Project.Name)
	}
	if cfg.Framework.Type != "bepinex" {
		t.Errorf("expected 'bepinex', got %s", cfg.Framework.Type)
	}
	if cfg.Game.Path != "/home/user/games/TestGame" {
		t.Errorf("expected game path, got %s", cfg.Game.Path)
	}
}

func TestSaveProjectConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "artemis.toml")

	cfg := ProjectConfig{
		Project: ProjectInfo{
			Name:    "NewMod",
			Version: "0.1.0",
		},
		Framework: FrameworkConfig{
			Type:    "melonloader",
			Version: "0.6.1",
		},
		Game: GameConfig{
			Path:      "/home/user/games/AnotherGame",
			DeployDir: "Mods",
		},
		Build: BuildConfig{
			Configuration: "Release",
		},
	}

	if err := cfg.Save(path); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := LoadProject(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.Project.Name != "NewMod" {
		t.Errorf("expected 'NewMod', got %s", loaded.Project.Name)
	}
	if loaded.Framework.Type != "melonloader" {
		t.Errorf("expected 'melonloader', got %s", loaded.Framework.Type)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
go test ./internal/config/ -v
```

Expected: Compilation errors — types and functions don't exist yet.

- [ ] **Step 3: Implement global config**

Create `internal/config/config.go`:

```go
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// GlobalConfig holds application-wide settings.
type GlobalConfig struct {
	Editor   EditorConfig   `toml:"editor"`
	Terminal TerminalConfig `toml:"terminal"`
}

type EditorConfig struct {
	TabSize int `toml:"tab_size"`
}

type TerminalConfig struct {
	ImageProtocol string `toml:"image_protocol"` // "auto", "kitty", "sixel", "ascii"
}

func DefaultGlobal() GlobalConfig {
	return GlobalConfig{
		Editor: EditorConfig{
			TabSize: 4,
		},
		Terminal: TerminalConfig{
			ImageProtocol: "auto",
		},
	}
}

// LoadGlobal loads the global config from the given path.
// Returns defaults if the file doesn't exist.
func LoadGlobal(path string) (GlobalConfig, error) {
	cfg := DefaultGlobal()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Save writes the global config to the given path, creating directories as needed.
func (c GlobalConfig) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(c)
}

// DefaultGlobalPath returns the default path for the global config file.
func DefaultGlobalPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "artemis", "config.toml")
}
```

- [ ] **Step 4: Implement project config**

Create `internal/config/project.go`:

```go
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ProjectConfig holds per-project settings (artemis.toml).
type ProjectConfig struct {
	Project   ProjectInfo     `toml:"project"`
	Framework FrameworkConfig `toml:"framework"`
	Game      GameConfig      `toml:"game"`
	Build     BuildConfig     `toml:"build"`
}

type ProjectInfo struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
	Author  string `toml:"author,omitempty"`
}

type FrameworkConfig struct {
	Type    string `toml:"type"`    // "bepinex" or "melonloader"
	Version string `toml:"version"`
}

type GameConfig struct {
	Path      string `toml:"path"`
	DeployDir string `toml:"deploy_dir"`
}

type BuildConfig struct {
	Configuration string `toml:"configuration"` // "Debug" or "Release"
}

// LoadProject loads a project config from the given artemis.toml path.
func LoadProject(path string) (ProjectConfig, error) {
	var cfg ProjectConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Save writes the project config to the given path.
func (c ProjectConfig) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(c)
}

// FindProjectConfig walks up from dir looking for artemis.toml.
// Returns the path if found, empty string if not.
func FindProjectConfig(dir string) string {
	for {
		path := filepath.Join(dir, "artemis.toml")
		if _, err := os.Stat(path); err == nil {
			return path
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run:
```bash
go test ./internal/config/ -v
```

Expected: All 4 tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/config/
git commit -m "feat: add global and project config store with TOML"
```

---

### Task 5: Terminal Capability Detection

**Files:**
- Create: `internal/terminal/caps.go`
- Create: `internal/terminal/caps_test.go`

- [ ] **Step 1: Write tests for capability detection**

Create `internal/terminal/caps_test.go`:

```go
package terminal

import "testing"

func TestDetectImageProtocol_Override(t *testing.T) {
	caps := Detect("kitty")
	if caps.ImageProtocol != ProtocolKitty {
		t.Errorf("expected kitty, got %v", caps.ImageProtocol)
	}
}

func TestDetectImageProtocol_AutoWithEnv(t *testing.T) {
	// TERM_PROGRAM=WezTerm should detect sixel
	caps := detectFromEnv("WezTerm", "")
	if caps.ImageProtocol != ProtocolSixel {
		t.Errorf("expected sixel for WezTerm, got %v", caps.ImageProtocol)
	}

	// TERM=xterm-kitty should detect kitty
	caps = detectFromEnv("", "xterm-kitty")
	if caps.ImageProtocol != ProtocolKitty {
		t.Errorf("expected kitty for xterm-kitty, got %v", caps.ImageProtocol)
	}
}

func TestDetectImageProtocol_Fallback(t *testing.T) {
	caps := detectFromEnv("", "xterm-256color")
	if caps.ImageProtocol != ProtocolASCII {
		t.Errorf("expected ASCII fallback, got %v", caps.ImageProtocol)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
go test ./internal/terminal/ -v
```

Expected: Compilation errors.

- [ ] **Step 3: Implement capability detection**

Create `internal/terminal/caps.go`:

```go
package terminal

import (
	"os"
	"strings"
)

// ImageProtocol represents the terminal's image display capability.
type ImageProtocol int

const (
	ProtocolASCII  ImageProtocol = iota // block characters only
	ProtocolSixel                        // sixel graphics
	ProtocolKitty                        // kitty graphics protocol
)

func (p ImageProtocol) String() string {
	switch p {
	case ProtocolKitty:
		return "kitty"
	case ProtocolSixel:
		return "sixel"
	default:
		return "ascii"
	}
}

// Capabilities holds detected terminal features.
type Capabilities struct {
	ImageProtocol ImageProtocol
}

// Detect returns terminal capabilities.
// override can be "kitty", "sixel", or "ascii" to skip auto-detection.
func Detect(override string) Capabilities {
	switch strings.ToLower(override) {
	case "kitty":
		return Capabilities{ImageProtocol: ProtocolKitty}
	case "sixel":
		return Capabilities{ImageProtocol: ProtocolSixel}
	case "ascii":
		return Capabilities{ImageProtocol: ProtocolASCII}
	default:
		return detectFromEnv(os.Getenv("TERM_PROGRAM"), os.Getenv("TERM"))
	}
}

// detectFromEnv detects capabilities from environment variables.
func detectFromEnv(termProgram, term string) Capabilities {
	tp := strings.ToLower(termProgram)
	t := strings.ToLower(term)

	// Kitty detection
	if strings.Contains(t, "kitty") || tp == "kitty" {
		return Capabilities{ImageProtocol: ProtocolKitty}
	}

	// Sixel-capable terminals
	sixelTerminals := []string{"wezterm", "foot", "contour", "mintty", "mlterm"}
	for _, st := range sixelTerminals {
		if strings.Contains(tp, st) {
			return Capabilities{ImageProtocol: ProtocolSixel}
		}
	}

	// XTerm with sixel support
	if tp == "xterm" {
		return Capabilities{ImageProtocol: ProtocolSixel}
	}

	return Capabilities{ImageProtocol: ProtocolASCII}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:
```bash
go test ./internal/terminal/ -v
```

Expected: All 3 tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/terminal/
git commit -m "feat: add terminal capability detection for image protocols"
```

---

### Task 6: Project Manager Panel

**Files:**
- Create: `internal/panel/project/model.go`
- Create: `internal/panel/project/templates.go`
- Create: `internal/panel/project/model_test.go`

- [ ] **Step 1: Write tests for project scaffolding**

Create `internal/panel/project/model_test.go`:

```go
package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScaffoldBlank(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "TestMod")

	err := Scaffold(projectDir, "TestMod", TemplateBlank)
	if err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	// Check expected files exist
	expectedFiles := []string{
		"src/Plugin.cs",
		"artemis.toml",
		"TestMod.csproj",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(projectDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}
}

func TestScaffoldBepInEx(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "MyBepMod")

	err := Scaffold(projectDir, "MyBepMod", TemplateBepInEx)
	if err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	expectedFiles := []string{
		"src/Plugin.cs",
		"src/Patches.cs",
		"artemis.toml",
		"MyBepMod.csproj",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(projectDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	// Verify Plugin.cs contains BepInPlugin attribute
	data, err := os.ReadFile(filepath.Join(projectDir, "src/Plugin.cs"))
	if err != nil {
		t.Fatal(err)
	}
	if !contains(string(data), "BepInPlugin") {
		t.Error("Plugin.cs should contain BepInPlugin attribute")
	}
}

func TestScaffoldMelonLoader(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "MyMelonMod")

	err := Scaffold(projectDir, "MyMelonMod", TemplateMelonLoader)
	if err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	expectedFiles := []string{
		"src/Mod.cs",
		"src/Patches.cs",
		"artemis.toml",
		"MyMelonMod.csproj",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(projectDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	data, err := os.ReadFile(filepath.Join(projectDir, "src/Mod.cs"))
	if err != nil {
		t.Fatal(err)
	}
	if !contains(string(data), "MelonMod") {
		t.Error("Mod.cs should contain MelonMod")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
go test ./internal/panel/project/ -v
```

Expected: Compilation errors.

- [ ] **Step 3: Implement project templates**

Create `internal/panel/project/templates.go`:

```go
package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/eden/artemis/internal/config"
)

// TemplateType identifies a project template.
type TemplateType int

const (
	TemplateBlank TemplateType = iota
	TemplateBepInEx
	TemplateMelonLoader
)

func (t TemplateType) String() string {
	switch t {
	case TemplateBepInEx:
		return "BepInEx"
	case TemplateMelonLoader:
		return "MelonLoader"
	default:
		return "Blank"
	}
}

// Scaffold creates a new mod project at the given directory.
func Scaffold(dir, name string, tmpl TemplateType) error {
	// Create directory structure
	dirs := []string{"src"}
	if tmpl != TemplateBlank {
		dirs = append(dirs, "libs", "assets")
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(dir, d), 0755); err != nil {
			return err
		}
	}

	// Write source files
	switch tmpl {
	case TemplateBlank:
		if err := writeFile(filepath.Join(dir, "src/Plugin.cs"), blankPlugin(name)); err != nil {
			return err
		}
	case TemplateBepInEx:
		if err := writeFile(filepath.Join(dir, "src/Plugin.cs"), bepinexPlugin(name)); err != nil {
			return err
		}
		if err := writeFile(filepath.Join(dir, "src/Patches.cs"), bepinexPatches(name)); err != nil {
			return err
		}
	case TemplateMelonLoader:
		if err := writeFile(filepath.Join(dir, "src/Mod.cs"), melonMod(name)); err != nil {
			return err
		}
		if err := writeFile(filepath.Join(dir, "src/Patches.cs"), melonPatches(name)); err != nil {
			return err
		}
	}

	// Write .csproj
	if err := writeFile(filepath.Join(dir, name+".csproj"), csproj(name, tmpl)); err != nil {
		return err
	}

	// Write artemis.toml
	cfg := config.ProjectConfig{
		Project: config.ProjectInfo{
			Name:    name,
			Version: "1.0.0",
		},
		Framework: frameworkConfig(tmpl),
		Build: config.BuildConfig{
			Configuration: "Debug",
		},
	}
	return cfg.Save(filepath.Join(dir, "artemis.toml"))
}

func frameworkConfig(tmpl TemplateType) config.FrameworkConfig {
	switch tmpl {
	case TemplateBepInEx:
		return config.FrameworkConfig{Type: "bepinex", Version: "5.4.22"}
	case TemplateMelonLoader:
		return config.FrameworkConfig{Type: "melonloader", Version: "0.6.1"}
	default:
		return config.FrameworkConfig{}
	}
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func blankPlugin(name string) string {
	return fmt.Sprintf(`using UnityEngine;

namespace %s
{
    public class Plugin
    {
    }
}
`, name)
}

func bepinexPlugin(name string) string {
	return fmt.Sprintf(`using BepInEx;
using BepInEx.Logging;
using HarmonyLib;
using UnityEngine;

namespace %s
{
    [BepInPlugin("com.author.%s", "%s", "1.0.0")]
    public class Plugin : BaseUnityPlugin
    {
        private Harmony _harmony;

        private void Awake()
        {
            Logger.LogInfo("%s loaded!");
            _harmony = new Harmony("com.author.%s");
            _harmony.PatchAll();
        }
    }
}
`, name, name, name, name, name)
}

func bepinexPatches(name string) string {
	return fmt.Sprintf(`using HarmonyLib;

namespace %s
{
    // Example patch — replace TargetType and TargetMethod with real targets.
    // [HarmonyPatch(typeof(TargetType), "TargetMethod")]
    // public class ExamplePatch
    // {
    //     static void Postfix()
    //     {
    //     }
    // }
}
`, name)
}

func melonMod(name string) string {
	return fmt.Sprintf(`using MelonLoader;
using HarmonyLib;
using UnityEngine;

[assembly: MelonInfo(typeof(%s.Mod), "%s", "1.0.0", "Author")]
[assembly: MelonGame(null, null)]

namespace %s
{
    public class Mod : MelonMod
    {
        public override void OnInitializeMelon()
        {
            LoggerInstance.Msg("%s loaded!");
        }
    }
}
`, name, name, name, name)
}

func melonPatches(name string) string {
	return fmt.Sprintf(`using HarmonyLib;

namespace %s
{
    // Example patch — replace TargetType and TargetMethod with real targets.
    // [HarmonyPatch(typeof(TargetType), "TargetMethod")]
    // public class ExamplePatch
    // {
    //     static void Postfix()
    //     {
    //     }
    // }
}
`, name)
}

func csproj(name string, tmpl TemplateType) string {
	refs := ""
	switch tmpl {
	case TemplateBepInEx:
		refs = `
    <!-- Update these paths to point to your game's managed DLLs -->
    <Reference Include="BepInEx">
      <HintPath>libs/BepInEx.dll</HintPath>
    </Reference>
    <Reference Include="0Harmony">
      <HintPath>libs/0Harmony.dll</HintPath>
    </Reference>
    <Reference Include="UnityEngine">
      <HintPath>libs/UnityEngine.dll</HintPath>
    </Reference>
    <Reference Include="UnityEngine.CoreModule">
      <HintPath>libs/UnityEngine.CoreModule.dll</HintPath>
    </Reference>`
	case TemplateMelonLoader:
		refs = `
    <!-- Update these paths to point to your game's managed DLLs -->
    <Reference Include="MelonLoader">
      <HintPath>libs/MelonLoader.dll</HintPath>
    </Reference>
    <Reference Include="0Harmony">
      <HintPath>libs/0Harmony.dll</HintPath>
    </Reference>
    <Reference Include="UnityEngine">
      <HintPath>libs/UnityEngine.dll</HintPath>
    </Reference>
    <Reference Include="UnityEngine.CoreModule">
      <HintPath>libs/UnityEngine.CoreModule.dll</HintPath>
    </Reference>`
	}

	return fmt.Sprintf(`<Project Sdk="Microsoft.NET.Sdk">

  <PropertyGroup>
    <TargetFramework>net472</TargetFramework>
    <AssemblyName>%s</AssemblyName>
    <RootNamespace>%s</RootNamespace>
    <LangVersion>latest</LangVersion>
  </PropertyGroup>

  <ItemGroup>%s
  </ItemGroup>

</Project>
`, name, name, refs)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:
```bash
go test ./internal/panel/project/ -v
```

Expected: All 3 tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/panel/project/templates.go internal/panel/project/model_test.go
git commit -m "feat: add project scaffolding with BepInEx/MelonLoader/blank templates"
```

---

### Task 7: Project Manager Panel Model

**Files:**
- Create: `internal/panel/project/model.go`
- Modify: `internal/app/model.go`

- [ ] **Step 1: Implement the project manager panel**

Create `internal/panel/project/model.go`:

```go
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

// State tracks the current view state of the project manager.
type state int

const (
	stateList state = iota
	stateNewName
	stateNewTemplate
)

// ProjectEntry represents a discovered project.
type ProjectEntry struct {
	Name      string
	Path      string
	Framework string
	Modified  time.Time
}

type Model struct {
	width        int
	height       int
	projects     []ProjectEntry
	cursor       int
	state        state
	nameInput    textinput.Model
	templateIdx  int
	workspaceDir string
}

var templates = []TemplateType{TemplateBlank, TemplateBepInEx, TemplateMelonLoader}

func NewModel(workspaceDir string) Model {
	ti := textinput.New()
	ti.Placeholder = "MyMod"
	ti.CharLimit = 64

	m := Model{
		workspaceDir: workspaceDir,
		nameInput:    ti,
		state:        stateList,
	}
	m.scanProjects()
	return m
}

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
			Modified:  modified,
		})
	}
	sort.Slice(m.projects, func(i, j int) bool {
		return m.projects[i].Modified.After(m.projects[j].Modified)
	})
}

func (m Model) ID() panel.PanelID { return panel.PanelProject }

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m Model) StatusInfo() string {
	return fmt.Sprintf("%d projects", len(m.projects))
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	// Forward to text input when active
	if m.state == stateNewName {
		var cmd tea.Cmd
		m.nameInput, cmd = m.nameInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateList:
		return m.handleListKey(msg)
	case stateNewName:
		return m.handleNameKey(msg)
	case stateNewTemplate:
		return m.handleTemplateKey(msg)
	}
	return m, nil
}

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	totalItems := len(m.projects) + 1 // +1 for "New Project"
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
		// Open existing project
		return m, func() tea.Msg {
			return app.OpenProjectMsg{Path: m.projects[m.cursor].Path}
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
		m.state = stateNewName
		m.nameInput.SetValue("")
		m.nameInput.Focus()
		return m, m.nameInput.Cursor.BlinkCmd()
	}
	return m, nil
}

func (m Model) handleNameKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (m Model) handleTemplateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		name := strings.TrimSpace(m.nameInput.Value())
		dir := filepath.Join(m.workspaceDir, name)
		tmpl := templates[m.templateIdx]
		if err := Scaffold(dir, name, tmpl); err != nil {
			m.state = stateList
			return m, func() tea.Msg {
				return app.StatusMsg{Text: fmt.Sprintf("Error: %v", err)}
			}
		}
		m.scanProjects()
		m.state = stateList
		m.cursor = 0
		return m, func() tea.Msg {
			return app.OpenProjectMsg{Path: dir}
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		m.state = stateNewName
		return m, nil
	}
	return m, nil
}

func (m Model) View() string {
	switch m.state {
	case stateNewName:
		return m.viewNewName()
	case stateNewTemplate:
		return m.viewNewTemplate()
	default:
		return m.viewList()
	}
}

func (m Model) viewList() string {
	titleStyle := lipgloss.NewStyle().Foreground(app.ColorMauve).Bold(true)
	subtitleStyle := lipgloss.NewStyle().Foreground(app.ColorSurface2)
	selectedStyle := lipgloss.NewStyle().Foreground(app.ColorText).Background(app.ColorSurface0)
	normalStyle := lipgloss.NewStyle().Foreground(app.ColorText)
	dimStyle := lipgloss.NewStyle().Foreground(app.ColorSurface2)
	greenStyle := lipgloss.NewStyle().Foreground(app.ColorGreen)
	borderStyle := lipgloss.NewStyle().Foreground(app.ColorSurface1)

	var b strings.Builder

	// Center the content
	title := titleStyle.Render("Welcome to Artemis")
	subtitle := subtitleStyle.Render("Select or create a project")

	b.WriteString("\n\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, title))
	b.WriteString("\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, subtitle))
	b.WriteString("\n\n")

	// Project list box
	boxWidth := 48
	top := borderStyle.Render("┌" + strings.Repeat("─", boxWidth) + "┐")
	bot := borderStyle.Render("└" + strings.Repeat("─", boxWidth) + "┘")
	sep := borderStyle.Render("├" + strings.Repeat("─", boxWidth) + "┤")
	side := borderStyle.Render("│")

	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, top))
	b.WriteString("\n")

	for i, p := range m.projects {
		style := normalStyle
		if i == m.cursor {
			style = selectedStyle
		}

		nameLine := style.Render(padRight(p.Name, boxWidth))
		descLine := dimStyle.Render(padRight(p.Framework, boxWidth))

		b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
			side+nameLine+side))
		b.WriteString("\n")
		b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
			side+descLine+side))
		b.WriteString("\n")
		b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, sep))
		b.WriteString("\n")
	}

	// "New Project" entry
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

func (m Model) viewNewName() string {
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

func (m Model) viewNewTemplate() string {
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

func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}
```

- [ ] **Step 2: Verify it compiles**

Run:
```bash
go build ./... && echo "BUILD OK"
```

Expected: `BUILD OK`

- [ ] **Step 3: Commit**

```bash
git add internal/panel/project/model.go
git commit -m "feat: add project manager panel with list and new project flow"
```

---

### Task 8: Wire Up the App Shell

**Files:**
- Modify: `internal/app/model.go`
- Modify: `main.go`

- [ ] **Step 1: Rewrite the app shell to manage panels**

Replace the contents of `internal/app/model.go`:

```go
package app

import (
	"fmt"
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

// Model is the root application model.
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

// NewModel creates the root model with the given panels.
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
		// Global keybindings take priority
		if cmd := m.handleGlobalKey(msg); cmd != nil {
			return m, cmd
		}

	case SwitchPanelMsg:
		m.activePanel = msg.Panel
		return m, nil

	case OpenProjectMsg:
		m.activePanel = panel.PanelEditor
		// TODO: load project and switch to editor (implemented in Plan 2)
		m.statusMsg = fmt.Sprintf("Opened %s", msg.Path)
		return m, nil

	case StatusMsg:
		m.statusMsg = msg.Text
		return m, nil
	}

	// Forward to active panel
	if p, ok := m.panels[m.activePanel]; ok {
		updated, cmd := p.Update(msg)
		m.panels[m.activePanel] = updated.(panel.Panel)
		return m, cmd
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

	// Panel content
	if p, ok := m.panels[m.activePanel]; ok {
		panelView := p.View()
		panelHeight := m.height - chromeHeight
		panelView = lipgloss.NewStyle().
			Width(m.width).
			Height(panelHeight).
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

	tabLine := strings.Join(parts, StyleDim.Render("│"))
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

	// Active panel status
	if p, ok := m.panels[m.activePanel]; ok {
		info := p.StatusInfo()
		if info != "" {
			left += StyleDim.Render(" │ ") + StyleTopBar.Render(info)
		}
	}

	right := ""
	if m.statusMsg != "" {
		right = StyleTopBar.Render(m.statusMsg)
	}

	// Terminal image protocol indicator
	capsInfo := StyleSuccess.Render("● ") + StyleTopBar.Render(m.caps.ImageProtocol.String())
	right = capsInfo + "  " + right

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	return StyleStatusBar.Width(m.width).Render(
		left + strings.Repeat(" ", gap) + right)
}
```

- [ ] **Step 2: Update main.go to wire everything together**

Replace the contents of `main.go`:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eden/artemis/internal/app"
	"github.com/eden/artemis/internal/config"
	"github.com/eden/artemis/internal/panel/project"
	"github.com/eden/artemis/internal/terminal"
)

func main() {
	// Load global config
	globalCfg, err := config.LoadGlobal(config.DefaultGlobalPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		globalCfg = config.DefaultGlobal()
	}

	// Detect terminal capabilities
	caps := terminal.Detect(globalCfg.Terminal.ImageProtocol)

	// Workspace directory: use arg if provided, else current dir
	workspaceDir, _ := os.Getwd()
	if len(os.Args) > 1 {
		workspaceDir = os.Args[1]
		if !filepath.IsAbs(workspaceDir) {
			cwd, _ := os.Getwd()
			workspaceDir = filepath.Join(cwd, workspaceDir)
		}
	}

	// Create panels
	projectPanel := project.NewModel(workspaceDir)

	// Create app
	m := app.NewModel(globalCfg, caps, &projectPanel)

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Fix the project panel to implement Panel via pointer receiver**

The project panel `Model` uses pointer receivers for `SetSize`. Ensure it's passed as a pointer in `main.go` (already done with `&projectPanel`). The `Update` method returns `tea.Model`, so we need to handle the type assertion in the app shell. This is already handled in `model.go` with `updated.(panel.Panel)`.

Verify it compiles:
```bash
go build ./... && echo "BUILD OK"
```

Expected: `BUILD OK`

- [ ] **Step 4: Run the app and verify manually**

Run:
```bash
./artemis
```

Expected:
- Alt screen opens
- Top bar shows "artemis" with "Ctrl+P: Commands  ?: Help"
- Tab bar shows Project (active), Editor, Assets, Build
- Project manager shows "Welcome to Artemis" with "New Project" entry
- Status bar shows terminal image protocol
- Ctrl+Q quits

- [ ] **Step 5: Commit**

```bash
git add main.go internal/app/model.go
git commit -m "feat: wire up app shell with panel switching and project manager"
```

---

### Task 9: End-to-End Test — Create a Project

This is a manual smoke test to verify the full flow works.

- [ ] **Step 1: Create a temp workspace and launch**

Run:
```bash
mkdir -p /tmp/artemis-test-workspace
go build -o artemis . && ./artemis /tmp/artemis-test-workspace
```

- [ ] **Step 2: Test the flow**

In the TUI:
1. Arrow down to "+ New Project", press Enter
2. Type "TestMod", press Enter
3. Arrow to "BepInEx", press Enter
4. Verify it creates the project and attempts to open it

- [ ] **Step 3: Verify scaffolded files**

After quitting (Ctrl+Q), check:
```bash
ls -la /tmp/artemis-test-workspace/TestMod/
cat /tmp/artemis-test-workspace/TestMod/artemis.toml
cat /tmp/artemis-test-workspace/TestMod/src/Plugin.cs
```

Expected: Project files exist with BepInEx boilerplate.

- [ ] **Step 4: Clean up**

```bash
rm -rf /tmp/artemis-test-workspace
```

- [ ] **Step 5: Final commit (if any fixes needed)**

```bash
git add -A
git commit -m "fix: address issues found in smoke testing"
```

---

## Summary

After completing this plan, Artemis will:
- Launch as a full-screen TUI with Catppuccin Mocha theme
- Show a project manager panel with project discovery
- Support creating new projects from blank/BepInEx/MelonLoader templates
- Switch between panel tabs with Alt+1-4
- Load/save TOML config (global and per-project)
- Detect terminal image capabilities
- Quit cleanly with Ctrl+Q

**Next plan:** Plan 2 — Code Editor (file explorer, tabbed editor, syntax highlighting, VS Code keybindings)
