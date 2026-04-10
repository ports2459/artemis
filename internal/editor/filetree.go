package editor

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/eden/artemis/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// TreeNode represents a single node (file or directory) in the file tree.
type TreeNode struct {
	Name     string
	Path     string
	IsDir    bool
	Expanded bool
	Children []*TreeNode
}

// FlatNode is a flattened tree node with depth info for rendering.
type FlatNode struct {
	Node  *TreeNode
	Depth int
}

// FileTree is a navigable file tree model.
type FileTree struct {
	root    *TreeNode
	visible []FlatNode
	cursor  int
	width   int
	height  int
	focused bool
}

// NewFileTree creates a new empty FileTree.
func NewFileTree() *FileTree {
	return &FileTree{
		cursor: 0,
	}
}

// SetRoot sets the root directory of the tree, builds the tree, expands the root,
// and rebuilds the visible list.
func (ft *FileTree) SetRoot(dir string) {
	ft.root = BuildTree(dir)
	if ft.root != nil {
		ft.root.Expanded = true
	}
	ft.rebuildVisible()
	ft.cursor = 0
}

// BuildTree recursively scans a directory and returns a TreeNode hierarchy.
// Hidden files (starting with '.') are skipped. Directories come first, sorted alphabetically.
func BuildTree(dir string) *TreeNode {
	info, err := os.Stat(dir)
	if err != nil {
		return nil
	}

	node := &TreeNode{
		Name:  info.Name(),
		Path:  dir,
		IsDir: info.IsDir(),
	}

	if !info.IsDir() {
		return node
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return node
	}

	var dirs []*TreeNode
	var files []*TreeNode

	for _, entry := range entries {
		name := entry.Name()
		// Skip hidden files/directories
		if strings.HasPrefix(name, ".") {
			continue
		}
		childPath := filepath.Join(dir, name)
		child := BuildTree(childPath)
		if child != nil {
			if child.IsDir {
				dirs = append(dirs, child)
			} else {
				files = append(files, child)
			}
		}
	}

	// Sort dirs and files alphabetically
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	// Dirs first, then files
	node.Children = append(dirs, files...)
	return node
}

// FlattenTree flattens a tree node into a list of FlatNodes starting at the given depth.
// Only expanded directories have their children included.
func FlattenTree(node *TreeNode, depth int) []FlatNode {
	if node == nil {
		return nil
	}
	result := []FlatNode{{Node: node, Depth: depth}}
	if node.IsDir && node.Expanded {
		for _, child := range node.Children {
			result = append(result, FlattenTree(child, depth+1)...)
		}
	}
	return result
}

func (ft *FileTree) rebuildVisible() {
	if ft.root == nil {
		ft.visible = nil
		return
	}
	ft.visible = FlattenTree(ft.root, 0)
}

// CursorUp moves the cursor up one position, clamped to 0.
func (ft *FileTree) CursorUp() {
	if ft.cursor > 0 {
		ft.cursor--
	}
}

// CursorDown moves the cursor down one position, clamped to len(visible)-1.
func (ft *FileTree) CursorDown() {
	if ft.cursor < len(ft.visible)-1 {
		ft.cursor++
	}
}

// ToggleExpand toggles the expanded state of the currently selected directory.
func (ft *FileTree) ToggleExpand() {
	if ft.cursor >= 0 && ft.cursor < len(ft.visible) {
		node := ft.visible[ft.cursor].Node
		if node.IsDir {
			node.Expanded = !node.Expanded
			ft.rebuildVisible()
			// Clamp cursor after rebuild
			if ft.cursor >= len(ft.visible) {
				ft.cursor = len(ft.visible) - 1
			}
		}
	}
}

// SelectedPath returns the path of the currently selected node.
func (ft *FileTree) SelectedPath() string {
	if ft.cursor >= 0 && ft.cursor < len(ft.visible) {
		return ft.visible[ft.cursor].Node.Path
	}
	return ""
}

// SelectedIsDir returns whether the currently selected node is a directory.
func (ft *FileTree) SelectedIsDir() bool {
	if ft.cursor >= 0 && ft.cursor < len(ft.visible) {
		return ft.visible[ft.cursor].Node.IsDir
	}
	return false
}

// SetSize sets the viewport dimensions of the tree.
func (ft *FileTree) SetSize(width, height int) {
	ft.width = width
	ft.height = height
}

// SetFocused sets the focus state.
func (ft *FileTree) SetFocused(focused bool) {
	ft.focused = focused
}

// Visible returns the current visible flat nodes (for testing).
func (ft *FileTree) Visible() []FlatNode {
	return ft.visible
}

// Cursor returns the current cursor position (for testing).
func (ft *FileTree) Cursor() int {
	return ft.cursor
}

// Update handles key messages for the file tree.
func (ft *FileTree) Update(msg tea.Msg) tea.Cmd {
	if !ft.focused {
		return nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			ft.CursorUp()
		case "down", "j":
			ft.CursorDown()
		case "right":
			if ft.cursor >= 0 && ft.cursor < len(ft.visible) {
				node := ft.visible[ft.cursor].Node
				if node.IsDir && !node.Expanded {
					node.Expanded = true
					ft.rebuildVisible()
				}
			}
		case "left":
			if ft.cursor >= 0 && ft.cursor < len(ft.visible) {
				node := ft.visible[ft.cursor].Node
				if node.IsDir && node.Expanded {
					node.Expanded = false
					ft.rebuildVisible()
				}
			}
		case "enter":
			if ft.cursor >= 0 && ft.cursor < len(ft.visible) {
				node := ft.visible[ft.cursor].Node
				if node.IsDir {
					ft.ToggleExpand()
				} else {
					return func() tea.Msg {
						return app.FileOpenMsg{Path: node.Path}
					}
				}
			}
		}
	}
	return nil
}

// View renders the file tree as a string.
func (ft *FileTree) View() string {
	if ft.root == nil || len(ft.visible) == 0 {
		return app.StyleDim.Render("  No files")
	}

	// Calculate scroll offset
	scrollOffset := 0
	viewHeight := ft.height
	if viewHeight <= 0 {
		viewHeight = len(ft.visible)
	}
	if ft.cursor >= scrollOffset+viewHeight {
		scrollOffset = ft.cursor - viewHeight + 1
	}
	if ft.cursor < scrollOffset {
		scrollOffset = ft.cursor
	}

	var lines []string
	end := scrollOffset + viewHeight
	if end > len(ft.visible) {
		end = len(ft.visible)
	}

	cursorStyle := lipgloss.NewStyle().Background(app.ColorSurface1).Foreground(app.ColorText)
	dirStyle := lipgloss.NewStyle().Foreground(app.ColorBlue).Bold(true)
	fileStyle := lipgloss.NewStyle().Foreground(app.ColorText)

	for i := scrollOffset; i < end; i++ {
		fn := ft.visible[i]
		indent := strings.Repeat("  ", fn.Depth)

		var icon string
		var name string
		if fn.Node.IsDir {
			if fn.Node.Expanded {
				icon = "▼ "
			} else {
				icon = "▶ "
			}
			name = dirStyle.Render(fn.Node.Name)
		} else {
			icon = fileIcon(fn.Node.Name) + " "
			name = fileStyle.Render(fn.Node.Name)
		}

		line := indent + icon + name

		// Pad to width
		lineLen := lipgloss.Width(line)
		if ft.width > 0 && lineLen < ft.width {
			line += strings.Repeat(" ", ft.width-lineLen)
		}

		if i == ft.cursor && ft.focused {
			line = cursorStyle.Render(line)
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// fileIcon returns a single character icon based on file extension.
func fileIcon(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".cs":
		return "C"
	case ".json":
		return "J"
	case ".toml":
		return "T"
	case ".lua":
		return "L"
	case ".go":
		return "G"
	case ".md":
		return "M"
	case ".txt":
		return "t"
	case ".yaml", ".yml":
		return "Y"
	case ".xml":
		return "X"
	case ".py":
		return "P"
	case ".rs":
		return "R"
	case ".sh":
		return "S"
	case ".cfg", ".ini", ".conf":
		return "c"
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp":
		return "i"
	default:
		return "·"
	}
}
