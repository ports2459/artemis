package editor

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eden/artemis/internal/app"
)

func TestBuildTree(t *testing.T) {
	// Create a temp directory structure
	tmp := t.TempDir()

	// Create dirs
	os.MkdirAll(filepath.Join(tmp, "src", "sub"), 0755)
	os.MkdirAll(filepath.Join(tmp, "assets"), 0755)

	// Create files
	os.WriteFile(filepath.Join(tmp, "readme.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(tmp, "src", "main.cs"), []byte("code"), 0644)
	os.WriteFile(filepath.Join(tmp, "src", "sub", "helper.cs"), []byte("help"), 0644)
	os.WriteFile(filepath.Join(tmp, "assets", "icon.png"), []byte("img"), 0644)

	// Create a hidden file that should be skipped
	os.WriteFile(filepath.Join(tmp, ".hidden"), []byte("secret"), 0644)

	tree := BuildTree(tmp)
	if tree == nil {
		t.Fatal("BuildTree returned nil")
	}

	if !tree.IsDir {
		t.Error("root should be a directory")
	}

	// Should have 3 children: assets, src, readme.txt (dirs first, alphabetical)
	if len(tree.Children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(tree.Children))
	}

	// Dirs should come first
	if tree.Children[0].Name != "assets" {
		t.Errorf("first child should be 'assets', got %q", tree.Children[0].Name)
	}
	if tree.Children[1].Name != "src" {
		t.Errorf("second child should be 'src', got %q", tree.Children[1].Name)
	}
	if tree.Children[2].Name != "readme.txt" {
		t.Errorf("third child should be 'readme.txt', got %q", tree.Children[2].Name)
	}

	// Check nested structure
	src := tree.Children[1]
	if len(src.Children) != 2 { // sub dir + main.cs
		t.Fatalf("src should have 2 children, got %d", len(src.Children))
	}
	if src.Children[0].Name != "sub" {
		t.Errorf("first src child should be 'sub', got %q", src.Children[0].Name)
	}
	if src.Children[1].Name != "main.cs" {
		t.Errorf("second src child should be 'main.cs', got %q", src.Children[1].Name)
	}
}

func TestFlattenTree(t *testing.T) {
	root := &TreeNode{
		Name:     "root",
		Path:     "/root",
		IsDir:    true,
		Expanded: true,
		Children: []*TreeNode{
			{
				Name:     "src",
				Path:     "/root/src",
				IsDir:    true,
				Expanded: true,
				Children: []*TreeNode{
					{Name: "main.cs", Path: "/root/src/main.cs", IsDir: false},
				},
			},
			{Name: "readme.txt", Path: "/root/readme.txt", IsDir: false},
		},
	}

	flat := FlattenTree(root, 0)
	if len(flat) != 4 { // root, src, main.cs, readme.txt
		t.Fatalf("expected 4 flat nodes, got %d", len(flat))
	}

	// Check depths
	expectedDepths := []int{0, 1, 2, 1}
	for i, fn := range flat {
		if fn.Depth != expectedDepths[i] {
			t.Errorf("node %d (%s): expected depth %d, got %d", i, fn.Node.Name, expectedDepths[i], fn.Depth)
		}
	}

	// Now collapse src and re-flatten
	root.Children[0].Expanded = false
	flat = FlattenTree(root, 0)
	if len(flat) != 3 { // root, src (collapsed), readme.txt
		t.Fatalf("after collapse, expected 3 flat nodes, got %d", len(flat))
	}
}

func TestNavigateCursor(t *testing.T) {
	ft := NewFileTree()

	// Build a simple tree
	root := &TreeNode{
		Name:     "root",
		Path:     "/root",
		IsDir:    true,
		Expanded: true,
		Children: []*TreeNode{
			{Name: "a.txt", Path: "/root/a.txt", IsDir: false},
			{Name: "b.txt", Path: "/root/b.txt", IsDir: false},
			{Name: "c.txt", Path: "/root/c.txt", IsDir: false},
		},
	}
	ft.root = root
	ft.rebuildVisible()

	// Should start at 0
	if ft.Cursor() != 0 {
		t.Errorf("expected cursor at 0, got %d", ft.Cursor())
	}

	// Move down
	ft.CursorDown()
	if ft.Cursor() != 1 {
		t.Errorf("expected cursor at 1, got %d", ft.Cursor())
	}

	ft.CursorDown()
	ft.CursorDown()
	if ft.Cursor() != 3 {
		t.Errorf("expected cursor at 3, got %d", ft.Cursor())
	}

	// Should clamp at bottom (4 visible nodes: root + 3 files)
	ft.CursorDown()
	if ft.Cursor() != 3 {
		t.Errorf("expected cursor clamped at 3, got %d", ft.Cursor())
	}

	// Move up
	ft.CursorUp()
	if ft.Cursor() != 2 {
		t.Errorf("expected cursor at 2, got %d", ft.Cursor())
	}

	// Clamp at top
	ft.CursorUp()
	ft.CursorUp()
	ft.CursorUp()
	if ft.Cursor() != 0 {
		t.Errorf("expected cursor clamped at 0, got %d", ft.Cursor())
	}
}

func TestFileTreeEnterOpensFile(t *testing.T) {
	ft := NewFileTree()
	ft.focused = true

	root := &TreeNode{
		Name:     "root",
		Path:     "/root",
		IsDir:    true,
		Expanded: true,
		Children: []*TreeNode{
			{Name: "a.txt", Path: "/root/a.txt", IsDir: false},
		},
	}
	ft.root = root
	ft.rebuildVisible()

	// Move to file
	ft.CursorDown()
	if ft.SelectedPath() != "/root/a.txt" {
		t.Errorf("expected selected path /root/a.txt, got %q", ft.SelectedPath())
	}

	// Press enter - should return FileOpenMsg command
	cmd := ft.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command from enter on file")
	}
	msg := cmd()
	fileMsg, ok := msg.(app.FileOpenMsg)
	if !ok {
		t.Fatalf("expected FileOpenMsg, got %T", msg)
	}
	if fileMsg.Path != "/root/a.txt" {
		t.Errorf("expected path /root/a.txt, got %q", fileMsg.Path)
	}
}

func TestFileTreeToggleExpand(t *testing.T) {
	ft := NewFileTree()

	root := &TreeNode{
		Name:     "root",
		Path:     "/root",
		IsDir:    true,
		Expanded: true,
		Children: []*TreeNode{
			{
				Name:     "dir",
				Path:     "/root/dir",
				IsDir:    true,
				Expanded: false,
				Children: []*TreeNode{
					{Name: "file.cs", Path: "/root/dir/file.cs", IsDir: false},
				},
			},
		},
	}
	ft.root = root
	ft.rebuildVisible()

	// Initially: root, dir (collapsed) = 2 visible
	if len(ft.Visible()) != 2 {
		t.Fatalf("expected 2 visible, got %d", len(ft.Visible()))
	}

	// Select dir and expand
	ft.CursorDown()
	ft.ToggleExpand()

	// Now: root, dir (expanded), file.cs = 3 visible
	if len(ft.Visible()) != 3 {
		t.Fatalf("expected 3 visible after expand, got %d", len(ft.Visible()))
	}

	// Collapse again
	ft.ToggleExpand()
	if len(ft.Visible()) != 2 {
		t.Fatalf("expected 2 visible after collapse, got %d", len(ft.Visible()))
	}
}
