package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Editor string   `yaml:"editor"`
	UI     UIConfig `yaml:"ui"`
}

type UIConfig struct {
	LineNumbers string `yaml:"line_numbers"`
	Theme       string `yaml:"theme"`
	DiffAddBg   string `yaml:"diff_add_bg"`
	DiffDelBg   string `yaml:"diff_del_bg"`
}

func Load() Config {
	cfg := Config{
		UI: UIConfig{
			LineNumbers: "hybrid",
			Theme:       "default",
		},
	}

	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "difi", "config.yaml")

	if data, err := os.ReadFile(configPath); err == nil {
		_ = yaml.Unmarshal(data, &cfg)
	}

	if cfg.Editor == "" {
		cfg.Editor = os.Getenv("DIFI_EDITOR")
	}
	if cfg.Editor == "" {
		cfg.Editor = os.Getenv("EDITOR")
	}
	if cfg.Editor == "" {
		cfg.Editor = os.Getenv("VISUAL")
	}
	if cfg.Editor == "" {
		cfg.Editor = "vi"
	}

	return cfg
}
