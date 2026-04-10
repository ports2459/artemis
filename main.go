package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eden/artemis/internal/app"
	"github.com/eden/artemis/internal/config"
	"github.com/eden/artemis/internal/lua"
	assetspanel "github.com/eden/artemis/internal/panel/assets"
	buildpanel "github.com/eden/artemis/internal/panel/build"
	decompilerpanel "github.com/eden/artemis/internal/panel/decompiler"
	editorpanel "github.com/eden/artemis/internal/panel/editor"
	"github.com/eden/artemis/internal/panel/project"
	"github.com/eden/artemis/internal/terminal"
)

func main() {
	globalCfg, err := config.LoadGlobal(config.DefaultGlobalPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		globalCfg = config.DefaultGlobal()
	}

	caps := terminal.Detect(globalCfg.Terminal.ImageProtocol)

	workspaceDir, _ := os.Getwd()
	if len(os.Args) > 1 {
		workspaceDir = os.Args[1]
		if !filepath.IsAbs(workspaceDir) {
			cwd, _ := os.Getwd()
			workspaceDir = filepath.Join(cwd, workspaceDir)
		}
	}

	// Initialize Lua plugin runtime
	luaRuntime := lua.NewRuntime()
	defer luaRuntime.Close()

	// Load user init.lua and plugins
	initPath := filepath.Join(os.Getenv("HOME"), ".config", "artemis", "init.lua")
	_ = luaRuntime.LoadInitFile(initPath)
	globalPluginDir := filepath.Join(os.Getenv("HOME"), ".config", "artemis", "plugins")
	_ = luaRuntime.LoadPlugins(globalPluginDir)

	// Create panels
	projectPanel := project.NewModel(workspaceDir)
	editorPanel := editorpanel.NewModel(globalCfg.Editor.TabSize)
	assetsPanel := assetspanel.NewModel(caps)
	buildPanel := buildpanel.NewModel()
	decompilerPanel := decompilerpanel.NewModel()

	m := app.NewModel(globalCfg, caps, &projectPanel, &editorPanel, &assetsPanel, &buildPanel, &decompilerPanel)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
