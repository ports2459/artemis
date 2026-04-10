package app

import "github.com/charmbracelet/lipgloss"

// Artemis palette — blue theme matching i3/desktop
var (
	ColorRosewater = lipgloss.Color("#d4d9e0")
	ColorFlamingo  = lipgloss.Color("#c0c8d4")
	ColorPink      = lipgloss.Color("#a3b8d0")
	ColorMauve     = lipgloss.Color("#6b9ccf") // primary accent (from i3)
	ColorRed       = lipgloss.Color("#cf6b6b")
	ColorMaroon    = lipgloss.Color("#d48a8a")
	ColorPeach     = lipgloss.Color("#d4a373")
	ColorYellow    = lipgloss.Color("#d4c97a")
	ColorGreen     = lipgloss.Color("#7abf7a")
	ColorTeal      = lipgloss.Color("#5ca3a3")
	ColorSky       = lipgloss.Color("#7ab5ff") // link color (from i3)
	ColorSapphire  = lipgloss.Color("#5a9fd4")
	ColorBlue      = lipgloss.Color("#7ab5ff") // link/bright blue (from i3)
	ColorLavender  = lipgloss.Color("#8aaed4")
	ColorText      = lipgloss.Color("#d4d9e0") // text (from i3)
	ColorSubtext1  = lipgloss.Color("#b0b8c4")
	ColorSubtext0  = lipgloss.Color("#8a94a4")
	ColorOverlay2  = lipgloss.Color("#6a7486")
	ColorOverlay1  = lipgloss.Color("#4a6f9e") // muted (from i3)
	ColorOverlay0  = lipgloss.Color("#3d5578")
	ColorSurface2  = lipgloss.Color("#2c2c3a") // border (from i3)
	ColorSurface1  = lipgloss.Color("#242434")
	ColorSurface0  = lipgloss.Color("#1a1a24") // panel (from i3)
	ColorBase      = lipgloss.Color("#14141c")
	ColorMantle    = lipgloss.Color("#101018")
	ColorCrust     = lipgloss.Color("#0f0f14") // bg (from i3)
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
