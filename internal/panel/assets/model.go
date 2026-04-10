package assets

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eden/artemis/internal/app"
	assetslib "github.com/eden/artemis/internal/assets"
	"github.com/eden/artemis/internal/config"
	"github.com/eden/artemis/internal/panel"
	"github.com/eden/artemis/internal/terminal"
)

// focus tracks which column has keyboard focus.
type focus int

const (
	focusBundles focus = iota
	focusAssets
	focusInspector
)

const (
	bundleWidthDefault    = 30
	inspectorWidthDefault = 44
	minCenterWidth        = 20
)

// Model is the asset browser panel. It presents a three-column layout:
// bundle list | asset list | inspector.
type Model struct {
	width, height int
	engine        *assetslib.Engine
	projectPath   string

	// Bundle list (left column)
	bundles      []assetslib.BundleInfo
	bundleCursor int
	bundleScroll int

	// Asset list (center column)
	assets      []assetslib.AssetInfo
	assetCursor int
	assetScroll int
	assetFilter assetslib.AssetType // -1 for "all"

	// Inspector (right column)
	selectedAsset *assetslib.AssetInfo
	properties    map[string]interface{}

	// Image preview (Kitty graphics protocol)
	caps         terminal.Capabilities
	previewImage string // rendered Kitty escape sequence for current texture
	previewPath  string // path of currently previewed asset

	focus          focus
	bundleWidth    int
	inspectorWidth int
	scanError      string
	scanning       bool
	scanStatus     string
	scanFiles      []string
	scanIndex      int
}

// Compile-time check: *Model satisfies panel.Panel.
var _ panel.Panel = (*Model)(nil)

// NewModel creates a new asset browser panel.
func NewModel(caps terminal.Capabilities) Model {
	return Model{
		engine:         assetslib.NewEngine(),
		caps:           caps,
		assetFilter:    assetslib.AssetType(-1), // all
		bundleWidth:    bundleWidthDefault,
		inspectorWidth: inspectorWidthDefault,
		focus:          focusBundles,
	}
}

// ---------------------------------------------------------------------------
// panel.Panel interface
// ---------------------------------------------------------------------------

func (m *Model) ID() panel.PanelID { return panel.PanelAssets }

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Model) StatusInfo() string {
	if !m.engine.Available() {
		return "Assets (UnityPy not found)"
	}
	n := 0
	for _, b := range m.bundles {
		n += len(b.Assets)
	}
	return fmt.Sprintf("%d bundles, %d assets", len(m.bundles), n)
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
		m.projectPath = msg.Path
		m.scanError = ""
		m.scanning = false
		m.scanStatus = ""
		m.engine.ClearBundles()
		// Load project config to get game path
		cfgPath := filepath.Join(msg.Path, "artemis.toml")
		cfg, err := config.LoadProject(cfgPath)
		if err != nil {
			m.scanError = fmt.Sprintf("Config error: %v", err)
		} else if cfg.Game.Path == "" {
			m.scanError = "No game path set. Edit artemis.toml to add [game] path."
		} else {
			gamePath := cfg.Game.Path
			dataPath := findUnityDataDir(gamePath)
			m.engine.SetGameDataPath(dataPath)
			m.scanning = true
			m.scanStatus = "Discovering asset files..."
		}
		m.bundles = nil
		m.assets = nil
		m.selectedAsset = nil
		m.properties = nil
		m.previewImage = ""
		m.previewPath = ""
		m.bundleCursor = 0
		m.assetCursor = 0
		if m.scanning {
			return m, m.doScan()
		}
		return m, nil

	case scanDoneMsg:
		m.scanning = false
		m.scanStatus = ""
		if msg.err != nil {
			m.scanError = fmt.Sprintf("Scan error: %v", msg.err)
			return m, nil
		}
		m.scanError = ""
		m.bundles = msg.bundles
		// Don't auto-index first bundle — just show the file list
		return m, nil

	case indexBundleMsg:
		m.scanning = false
		m.scanStatus = ""
		if msg.err != nil {
			m.scanError = fmt.Sprintf("Index error: %v", msg.err)
			return m, nil
		}
		if msg.index < len(m.bundles) {
			m.bundles[msg.index].Assets = msg.assets
			m.applyFilter()
		}
		return m, nil

	case propertiesMsg:
		m.properties = msg.props
		return m, nil

	case texturePreviewMsg:
		if msg.err == nil {
			// Only apply if the preview is still for the currently selected asset
			currentKey := ""
			if m.selectedAsset != nil {
				currentKey = fmt.Sprintf("%s:%d", m.selectedAsset.Bundle, m.selectedAsset.PathID)
			}
			if currentKey == msg.assetKey {
				m.previewImage = msg.rendered
				m.previewPath = msg.assetKey
			}
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// Internal messages
// ---------------------------------------------------------------------------

type scanDoneMsg struct {
	bundles []assetslib.BundleInfo
	err     error
}

type propertiesMsg struct {
	props map[string]interface{}
}

type texturePreviewMsg struct {
	rendered string
	assetKey string // bundle:pathID to match against current selection
	err      error
}

func (m *Model) doScan() tea.Cmd {
	engine := m.engine
	return func() tea.Msg {
		files, err := engine.ListFiles()
		if err != nil {
			return scanDoneMsg{err: err}
		}
		if len(files) == 0 {
			return scanDoneMsg{err: fmt.Errorf("no .assets or .bundle files found")}
		}

		// Deduplicate files (symlinks or overlapping walks can cause repeats)
		seen := make(map[string]bool, len(files))
		var bundles []assetslib.BundleInfo
		for _, f := range files {
			if seen[f] {
				continue
			}
			seen[f] = true
			info, _ := os.Stat(f)
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			bundles = append(bundles, assetslib.BundleInfo{
				Path: f,
				Name: filepath.Base(f),
				Size: size,
			})
		}
		return scanDoneMsg{bundles: bundles}
	}
}

// indexBundleMsg carries the result of indexing a single bundle's contents.
type indexBundleMsg struct {
	index  int
	assets []assetslib.AssetInfo
	err    error
}

func (m *Model) doIndexBundle(bundleIdx int) tea.Cmd {
	if bundleIdx >= len(m.bundles) {
		return nil
	}
	engine := m.engine
	bundle := m.bundles[bundleIdx]
	return func() tea.Msg {
		assets, err := engine.IndexBundle(bundle.Path)
		return indexBundleMsg{index: bundleIdx, assets: assets, err: err}
	}
}

func (m *Model) doGetProperties(bundle string, pathID int64) tea.Cmd {
	engine := m.engine
	return func() tea.Msg {
		props, _ := engine.GetProperties(bundle, pathID)
		return propertiesMsg{props: props}
	}
}

func (m *Model) doPreviewTexture(bundle string, pathID int64) tea.Cmd {
	engine := m.engine
	caps := m.caps
	assetKey := fmt.Sprintf("%s:%d", bundle, pathID)
	return func() tea.Msg {
		if caps.ImageProtocol != terminal.ProtocolKitty {
			return texturePreviewMsg{assetKey: assetKey, err: fmt.Errorf("no image protocol")}
		}

		tmpFile, err := os.CreateTemp("", "artemis-preview-*.png")
		if err != nil {
			return texturePreviewMsg{assetKey: assetKey, err: err}
		}
		tmpPath := tmpFile.Name()
		tmpFile.Close()
		defer os.Remove(tmpPath)

		if err := engine.ExportTexture(bundle, pathID, tmpPath); err != nil {
			return texturePreviewMsg{assetKey: assetKey, err: err}
		}

		rendered, err := terminal.RenderImageKitty(tmpPath, 28, 14)
		if err != nil {
			return texturePreviewMsg{assetKey: assetKey, err: err}
		}
		return texturePreviewMsg{assetKey: assetKey, rendered: rendered}
	}
}

// ---------------------------------------------------------------------------
// Key handling
// ---------------------------------------------------------------------------

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys
	switch msg.String() {
	case "tab":
		m.cycleFocusForward()
		return m, nil
	case "shift+tab":
		m.cycleFocusBackward()
		return m, nil
	case "1":
		m.assetFilter = assetslib.AssetType(-1) // All
		m.applyFilter()
		return m, nil
	case "2":
		m.assetFilter = assetslib.TypeTexture2D
		m.applyFilter()
		return m, nil
	case "3":
		m.assetFilter = assetslib.TypeMaterial
		m.applyFilter()
		return m, nil
	case "4":
		m.assetFilter = assetslib.TypeGameObject
		m.applyFilter()
		return m, nil
	case "5":
		m.assetFilter = assetslib.TypeAudioClip
		m.applyFilter()
		return m, nil
	}

	switch m.focus {
	case focusBundles:
		return m.handleBundleKey(msg)
	case focusAssets:
		return m.handleAssetKey(msg)
	case focusInspector:
		return m.handleInspectorKey(msg)
	}
	return m, nil
}

func (m *Model) handleBundleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.bundleCursor > 0 {
			m.bundleCursor--
			m.ensureBundleVisible()
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if m.bundleCursor < len(m.bundles)-1 {
			m.bundleCursor++
			m.ensureBundleVisible()
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if len(m.bundles) > 0 {
			cmd := m.selectBundle(m.bundleCursor)
			m.focus = focusAssets
			return m, cmd
		}
	}
	return m, nil
}

func (m *Model) handleAssetKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.assetCursor > 0 {
			m.assetCursor--
			m.ensureAssetVisible()
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if m.assetCursor < len(m.assets)-1 {
			m.assetCursor++
			m.ensureAssetVisible()
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if len(m.assets) > 0 {
			asset := m.assets[m.assetCursor]
			m.selectedAsset = &asset
			m.properties = nil
			m.previewImage = ""
			m.previewPath = ""
			m.focus = focusInspector
			if m.engine.Available() {
				var cmds []tea.Cmd
				cmds = append(cmds, m.doGetProperties(asset.Bundle, asset.PathID))
				if asset.Type == assetslib.TypeTexture2D && m.caps.ImageProtocol == terminal.ProtocolKitty {
					cmds = append(cmds, m.doPreviewTexture(asset.Bundle, asset.PathID))
				}
				return m, tea.Batch(cmds...)
			}
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("e"))):
		if m.selectedAsset != nil && m.engine.Available() {
			return m, m.doExport()
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		m.focus = focusBundles
	}
	return m, nil
}

func (m *Model) handleInspectorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		m.focus = focusAssets
	}
	return m, nil
}

func (m *Model) doExport() tea.Cmd {
	if m.selectedAsset == nil {
		return nil
	}
	asset := *m.selectedAsset
	engine := m.engine
	return func() tea.Msg {
		outDir := filepath.Join(filepath.Dir(asset.Bundle), "exports")
		outPath := filepath.Join(outDir, fmt.Sprintf("%s_%d.png", asset.Name, asset.PathID))
		err := engine.ExportTexture(asset.Bundle, asset.PathID, outPath)
		if err != nil {
			return app.StatusMsg{Text: fmt.Sprintf("Export error: %v", err)}
		}
		return app.StatusMsg{Text: fmt.Sprintf("Exported to %s", outPath)}
	}
}

// ---------------------------------------------------------------------------
// Focus helpers
// ---------------------------------------------------------------------------

func (m *Model) cycleFocusForward() {
	switch m.focus {
	case focusBundles:
		m.focus = focusAssets
	case focusAssets:
		m.focus = focusInspector
	case focusInspector:
		m.focus = focusBundles
	}
}

func (m *Model) cycleFocusBackward() {
	switch m.focus {
	case focusBundles:
		m.focus = focusInspector
	case focusAssets:
		m.focus = focusBundles
	case focusInspector:
		m.focus = focusAssets
	}
}

// ---------------------------------------------------------------------------
// Data helpers
// ---------------------------------------------------------------------------

func (m *Model) selectBundle(idx int) tea.Cmd {
	if idx < 0 || idx >= len(m.bundles) {
		return nil
	}
	m.bundleCursor = idx
	m.assetCursor = 0
	m.assetScroll = 0
	m.selectedAsset = nil
	m.properties = nil
	m.previewImage = ""
	m.previewPath = ""

	// If already indexed, just filter
	if len(m.bundles[idx].Assets) > 0 {
		m.applyFilter()
		m.scanning = false
		m.scanStatus = ""
		return nil
	}

	// Need to index — show loading state
	m.assets = nil
	m.scanning = true
	m.scanStatus = fmt.Sprintf("Indexing %s...", m.bundles[idx].Name)
	return m.doIndexBundle(idx)
}

func (m *Model) applyFilter() {
	if m.bundleCursor < 0 || m.bundleCursor >= len(m.bundles) {
		m.assets = nil
		return
	}
	all := m.bundles[m.bundleCursor].Assets
	if m.assetFilter < 0 {
		m.assets = all
		return
	}
	m.assets = nil
	for _, a := range all {
		if a.Type == m.assetFilter {
			m.assets = append(m.assets, a)
		}
	}
	if m.assetCursor >= len(m.assets) {
		m.assetCursor = max(0, len(m.assets)-1)
	}
}

func (m *Model) ensureBundleVisible() {
	visibleLines := m.contentHeight()
	if m.bundleCursor < m.bundleScroll {
		m.bundleScroll = m.bundleCursor
	}
	if m.bundleCursor >= m.bundleScroll+visibleLines {
		m.bundleScroll = m.bundleCursor - visibleLines + 1
	}
}

func (m *Model) ensureAssetVisible() {
	visibleLines := m.contentHeight()
	if m.assetCursor < m.assetScroll {
		m.assetScroll = m.assetCursor
	}
	if m.assetCursor >= m.assetScroll+visibleLines {
		m.assetScroll = m.assetCursor - visibleLines + 1
	}
}

func (m *Model) contentHeight() int {
	// header(1) + filter bar(1) + hrule(1) = 3 lines of overhead
	h := m.height - 3
	if h < 1 {
		h = 1
	}
	return h
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m *Model) View() string {
	if m.projectPath == "" {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(app.ColorOverlay1).Render(
				"Open a project to browse assets\n\n"+
					app.StyleDim.Render("Alt+1 to switch to Project panel")))
	}

	if !m.engine.Available() {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Center,
				app.StyleWarning.Render("UnityPy not found"),
				"",
				lipgloss.NewStyle().Foreground(app.ColorText).Render("Install with:"),
				app.StyleInfo.Render("pip install UnityPy"),
				"",
				app.StyleDim.Render("Asset browsing requires Python 3 and UnityPy."),
				app.StyleDim.Render("Bundles are listed but contents cannot be indexed."),
			))
	}

	// Three-column layout
	centerWidth := m.width - m.bundleWidth - m.inspectorWidth - 2 // 2 for borders
	if centerWidth < minCenterWidth {
		centerWidth = minCenterWidth
	}

	bundleCol := m.viewBundleColumn()
	assetCol := m.viewAssetColumn(centerWidth)
	inspCol := m.viewInspectorColumn()

	border := m.viewVerticalBorder()

	return lipgloss.JoinHorizontal(lipgloss.Top,
		bundleCol,
		border,
		assetCol,
		border,
		inspCol,
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

func (m *Model) viewBundleColumn() string {
	w := m.bundleWidth
	titleStyle := app.StyleTitle.Width(w)
	if m.focus == focusBundles {
		titleStyle = titleStyle.Underline(true)
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Bundles"))
	b.WriteString("\n")
	b.WriteString(app.HRule(w))
	b.WriteString("\n")

	ch := m.contentHeight()
	normalStyle := lipgloss.NewStyle().Foreground(app.ColorText).Width(w)
	selectedStyle := lipgloss.NewStyle().Foreground(app.ColorText).Background(app.ColorSurface0).Width(w)
	dimStyle := lipgloss.NewStyle().Foreground(app.ColorSurface2).Width(w)

	if m.scanning {
		b.WriteString(lipgloss.NewStyle().Foreground(app.ColorMauve).Width(w).Render(m.scanStatus))
		b.WriteString("\n")
	} else if len(m.bundles) == 0 {
		if m.scanError != "" {
			b.WriteString(lipgloss.NewStyle().Foreground(app.ColorRed).Width(w).Render(m.scanError))
		} else {
			b.WriteString(dimStyle.Render("No bundles found"))
		}
		b.WriteString("\n")
	}

	end := m.bundleScroll + ch
	if end > len(m.bundles) {
		end = len(m.bundles)
	}

	lines := 0
	for i := m.bundleScroll; i < end && lines < ch; i++ {
		bundle := m.bundles[i]
		sizeStr := formatSize(bundle.Size)
		nameMaxW := w - lipgloss.Width(sizeStr) - 4 // 2 prefix + 2 gap
		name := truncate(bundle.Name, nameMaxW)

		style := normalStyle
		if i == m.bundleCursor {
			style = selectedStyle
			label := "> " + name
			gap := w - lipgloss.Width(label) - len(sizeStr)
			if gap < 1 {
				gap = 1
			}
			label += strings.Repeat(" ", gap) + sizeStr
			b.WriteString(style.Render(label))
		} else {
			label := "  " + name
			gap := w - lipgloss.Width(label) - len(sizeStr)
			if gap < 1 {
				gap = 1
			}
			label += strings.Repeat(" ", gap) + sizeStr
			b.WriteString(style.Render(label))
		}
		b.WriteString("\n")
		lines++
	}

	// Pad remaining lines
	for lines < ch {
		b.WriteString(strings.Repeat(" ", w))
		b.WriteString("\n")
		lines++
	}

	return b.String()
}

func (m *Model) viewAssetColumn(w int) string {
	titleStyle := app.StyleTitle.Width(w)
	if m.focus == focusAssets {
		titleStyle = titleStyle.Underline(true)
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Assets"))
	b.WriteString("\n")

	// Filter bar
	b.WriteString(m.viewFilterBar(w))
	b.WriteString("\n")

	ch := m.contentHeight()
	normalStyle := lipgloss.NewStyle().Foreground(app.ColorText).Width(w)
	selectedStyle := lipgloss.NewStyle().Foreground(app.ColorText).Background(app.ColorSurface0).Width(w)
	dimStyle := lipgloss.NewStyle().Foreground(app.ColorSurface2).Width(w)
	typeStyle := lipgloss.NewStyle().Foreground(app.ColorOverlay1)

	if len(m.assets) == 0 {
		b.WriteString(dimStyle.Render("No assets"))
		b.WriteString("\n")
	}

	end := m.assetScroll + ch
	if end > len(m.assets) {
		end = len(m.assets)
	}

	lines := 0
	for i := m.assetScroll; i < end && lines < ch; i++ {
		a := m.assets[i]
		icon := assetIcon(a.Type)
		typeTag := "[" + a.TypeName + "]"
		// Reserve space: icon(1) + space(1) + space(1) + typeTag visual width
		reserved := 1 + 1 + 1 + lipgloss.Width(typeTag)
		nameMaxW := w - reserved
		if nameMaxW < 4 {
			nameMaxW = 4
		}
		cleanName := stripAnsi(a.Name)
		nameStr := truncate(cleanName, nameMaxW)
		entry := icon + " " + nameStr + " " + typeStyle.Render(typeTag)

		style := normalStyle
		if i == m.assetCursor {
			style = selectedStyle
		}
		b.WriteString(style.Render(truncate(entry, w)))
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

func (m *Model) viewFilterBar(w int) string {
	active := lipgloss.NewStyle().Foreground(app.ColorMauve).Bold(true)
	inactive := lipgloss.NewStyle().Foreground(app.ColorSurface2)

	filters := []struct {
		key   string
		label string
		ft    assetslib.AssetType
	}{
		{"1", "All", assetslib.AssetType(-1)},
		{"2", "Tex", assetslib.TypeTexture2D},
		{"3", "Mat", assetslib.TypeMaterial},
		{"4", "GO", assetslib.TypeGameObject},
		{"5", "Aud", assetslib.TypeAudioClip},
	}

	var parts []string
	for _, f := range filters {
		style := inactive
		if m.assetFilter == f.ft {
			style = active
		}
		parts = append(parts, style.Render(f.key+":"+f.label))
	}

	bar := strings.Join(parts, " ")
	return lipgloss.NewStyle().Width(w).Render(bar)
}

func (m *Model) viewInspectorColumn() string {
	w := m.inspectorWidth
	titleStyle := app.StyleTitle.Width(w)
	if m.focus == focusInspector {
		titleStyle = titleStyle.Underline(true)
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Inspector"))
	b.WriteString("\n")
	b.WriteString(app.HRule(w))
	b.WriteString("\n")

	ch := m.contentHeight()
	dimStyle := lipgloss.NewStyle().Foreground(app.ColorSurface2).Width(w)
	labelStyle := lipgloss.NewStyle().Foreground(app.ColorMauve)
	valStyle := lipgloss.NewStyle().Foreground(app.ColorText)

	lines := 0

	if m.selectedAsset == nil {
		b.WriteString(dimStyle.Render("Select an asset"))
		b.WriteString("\n")
		lines++
	} else {
		a := m.selectedAsset

		// Texture preview image (Kitty graphics protocol)
		if m.previewImage != "" && lines < ch {
			b.WriteString(m.previewImage)
			b.WriteString("\n")
			// The image occupies up to 14 rows in the terminal
			lines += 14
			if lines > ch {
				lines = ch
			}
		}

		// Asset info
		fields := []struct {
			label string
			value string
		}{
			{"Name", a.Name},
			{"Type", a.TypeName},
			{"Size", formatSize(a.Size)},
			{"PathID", fmt.Sprintf("%d", a.PathID)},
			{"Bundle", filepath.Base(a.Bundle)},
		}

		for _, f := range fields {
			if lines >= ch {
				break
			}
			cleanVal := stripAnsi(f.value)
			labelPart := f.label + ": "
			labelRendered := labelStyle.Render(labelPart)
			labelVisWidth := lipgloss.Width(labelRendered)
			valMaxWidth := w - labelVisWidth
			if valMaxWidth < 4 {
				valMaxWidth = 4
			}
			line := labelRendered + valStyle.Render(truncate(cleanVal, valMaxWidth))
			b.WriteString(lipgloss.NewStyle().Width(w).Render(line))
			b.WriteString("\n")
			lines++
		}

		if lines < ch {
			b.WriteString(app.HRule(w))
			b.WriteString("\n")
			lines++
		}

		// Properties
		if m.properties != nil && lines < ch {
			b.WriteString(labelStyle.Render("Properties:"))
			b.WriteString("\n")
			lines++

			for k, v := range m.properties {
				if lines >= ch {
					break
				}
				vs := stripAnsi(fmt.Sprintf("%v", v))
				keyPart := "  " + k + ": "
				valMaxWidth := w - lipgloss.Width(keyPart)
				if valMaxWidth < 4 {
					valMaxWidth = 4
				}
				line := keyPart + truncate(vs, valMaxWidth)
				b.WriteString(valStyle.Render(lipgloss.NewStyle().Width(w).Render(line)))
				b.WriteString("\n")
				lines++
			}
		} else if m.properties == nil && m.engine.Available() && lines < ch {
			b.WriteString(dimStyle.Render("Loading..."))
			b.WriteString("\n")
			lines++
		}

		// Action hints
		if lines < ch-1 {
			b.WriteString("\n")
			lines++
			hintStyle := lipgloss.NewStyle().Foreground(app.ColorOverlay1)
			hints := []string{"[e] Export", "[r] Replace"}
			for _, h := range hints {
				if lines >= ch {
					break
				}
				b.WriteString(hintStyle.Render(h))
				b.WriteString("\n")
				lines++
			}
		}
	}

	for lines < ch {
		b.WriteString(strings.Repeat(" ", w))
		b.WriteString("\n")
		lines++
	}

	return b.String()
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func assetIcon(t assetslib.AssetType) string {
	switch t {
	case assetslib.TypeTexture2D:
		return "#"
	case assetslib.TypeMaterial:
		return "@"
	case assetslib.TypeGameObject:
		return "*"
	case assetslib.TypeAudioClip:
		return "~"
	case assetslib.TypeShader:
		return "$"
	case assetslib.TypeTextAsset:
		return "="
	default:
		return "."
	}
}

// ansiRe matches ANSI escape sequences (CSI sequences, OSC, etc.)
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x1b]*\x1b\\|\x1b\[[0-9;]*m`)

// stripAnsi removes ANSI escape codes from a string.
func stripAnsi(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	// Strip any ANSI codes first to avoid cutting in the middle of an escape sequence
	s = stripAnsi(s)
	visWidth := lipgloss.Width(s)
	if visWidth <= maxLen {
		return s
	}
	if maxLen <= 3 {
		// For very short widths, take runes carefully
		runes := []rune(s)
		if len(runes) > maxLen {
			return string(runes[:maxLen])
		}
		return s
	}
	// Trim rune-by-rune until visual width fits
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > maxLen-3 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "..."
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// findUnityDataDir locates the Unity data directory inside a game folder.
// Unity games typically have a <GameName>_Data/ folder next to the executable.
func findUnityDataDir(gamePath string) string {
	entries, err := os.ReadDir(gamePath)
	if err != nil {
		return gamePath
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasSuffix(e.Name(), "_Data") {
			return filepath.Join(gamePath, e.Name())
		}
	}
	// Fallback: try common names
	for _, name := range []string{"Data", "GameData", "Managed"} {
		p := filepath.Join(gamePath, name)
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}
	// Just use the game path itself
	return gamePath
}
