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
	SetSize(width, height int)
	ID() PanelID
	StatusInfo() string
}
