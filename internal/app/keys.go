package app

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Quit       key.Binding
	Panel1     key.Binding
	Panel2     key.Binding
	Panel3     key.Binding
	Panel4     key.Binding
	Panel5     key.Binding
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
	Panel5: key.NewBinding(
		key.WithKeys("alt+5"),
		key.WithHelp("alt+5", "decompiler"),
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
