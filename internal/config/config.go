package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultEditor is used when the config does not set an editor.
const DefaultEditor = "code"

type Config struct {
	ProjectRoots []string `json:"project_roots"`
	ArchiveRoot  string   `json:"archive_root"`
	Editor       string   `json:"editor"`
}

// Path returns the location of the config file, which may not exist yet.
func Path() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get config directory: %w", err)
	}

	return filepath.Join(configDir, "dev", "config.json"), nil
}

func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}

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
