package editor

import "github.com/charmbracelet/bubbles/key"

type editorKeyMap struct {
	ToggleSidebar key.Binding
	Save          key.Binding
	CloseTab      key.Binding
	NextTab       key.Binding
	PrevTab       key.Binding
	Undo          key.Binding
	Redo          key.Binding
	Find          key.Binding
}

var editorKeys = editorKeyMap{
	ToggleSidebar: key.NewBinding(key.WithKeys("ctrl+b"), key.WithHelp("ctrl+b", "toggle sidebar")),
	Save:          key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save")),
	CloseTab:      key.NewBinding(key.WithKeys("ctrl+w"), key.WithHelp("ctrl+w", "close tab")),
	NextTab:       key.NewBinding(key.WithKeys("ctrl+tab"), key.WithHelp("ctrl+tab", "next tab")),
	PrevTab:       key.NewBinding(key.WithKeys("ctrl+shift+tab"), key.WithHelp("ctrl+shift+tab", "prev tab")),
	Undo:          key.NewBinding(key.WithKeys("ctrl+z"), key.WithHelp("ctrl+z", "undo")),
	Redo:          key.NewBinding(key.WithKeys("ctrl+y"), key.WithHelp("ctrl+y", "redo")),
	Find:          key.NewBinding(key.WithKeys("ctrl+f"), key.WithHelp("ctrl+f", "find")),
}
