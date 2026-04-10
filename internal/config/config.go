package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type GlobalConfig struct {
	Editor   EditorConfig   `toml:"editor"`
	Terminal TerminalConfig `toml:"terminal"`
}

type EditorConfig struct {
	TabSize int `toml:"tab_size"`
}

type TerminalConfig struct {
	ImageProtocol string `toml:"image_protocol"`
}

func DefaultGlobal() GlobalConfig {
	return GlobalConfig{
		Editor: EditorConfig{
			TabSize: 4,
		},
		Terminal: TerminalConfig{
			ImageProtocol: "auto",
		},
	}
}

func LoadGlobal(path string) (GlobalConfig, error) {
	cfg := DefaultGlobal()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (c GlobalConfig) Save(path string) error {
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

func DefaultGlobalPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "artemis", "config.toml")
}
