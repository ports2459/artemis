package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGlobalConfig_Defaults(t *testing.T) {
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
