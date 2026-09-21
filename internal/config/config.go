package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	ProjectRoots []string `json:"project_roots"`
	Editor       string   `json:"editor"`
}

func Load() (Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return Config{}, fmt.Errorf("get config directory: %w", err)
	}

	path := filepath.Join(configDir, "dev", "config.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config

	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}