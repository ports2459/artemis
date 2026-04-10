package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type ProjectConfig struct {
	Project   ProjectInfo     `toml:"project"`
	Framework FrameworkConfig `toml:"framework"`
	Game      GameConfig      `toml:"game"`
	Build     BuildConfig     `toml:"build"`
}

type ProjectInfo struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
	Author  string `toml:"author,omitempty"`
}

type FrameworkConfig struct {
	Type    string `toml:"type"`
	Version string `toml:"version"`
}

type GameConfig struct {
	Path      string `toml:"path"`
	DeployDir string `toml:"deploy_dir"`
}

type BuildConfig struct {
	Configuration string `toml:"configuration"`
}

func LoadProject(path string) (ProjectConfig, error) {
	var cfg ProjectConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (c ProjectConfig) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(c)
}

func FindProjectConfig(dir string) string {
	for {
		path := filepath.Join(dir, "artemis.toml")
		if _, err := os.Stat(path); err == nil {
			return path
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
