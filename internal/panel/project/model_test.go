package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldBlank(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "TestMod")

	err := Scaffold(projectDir, "TestMod", TemplateBlank)
	if err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

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

	data, err := os.ReadFile(filepath.Join(projectDir, "src/Plugin.cs"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "BepInPlugin") {
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
	if !strings.Contains(string(data), "MelonMod") {
		t.Error("Mod.cs should contain MelonMod")
	}
}
