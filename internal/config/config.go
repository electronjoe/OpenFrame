package config

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
)

const (
    DefaultConfigPath = ".openframe/config.json"
)

// Config represents the JSON config structure.
type Config struct {
    Albums      []string `json:"albums"`
    DateOverlay bool     `json:"dateOverlay"`
    Interval    int      `json:"interval"`
    Randomize   bool     `json:"randomize"`
}

// Read retrieves and parses the JSON config from ~/.openframe/config.json.
func Read() (Config, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return Config{}, fmt.Errorf("failed to get user home directory: %w", err)
    }
    configPath := filepath.Join(homeDir, DefaultConfigPath)

    data, err := os.ReadFile(configPath)
    if err != nil {
        return Config{}, fmt.Errorf("failed to read config file at %s: %w", configPath, err)
    }

    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return Config{}, fmt.Errorf("failed to parse config JSON: %w", err)
    }

    // Default interval if not set or invalid
    if cfg.Interval <= 0 {
        cfg.Interval = 10
    }

    return cfg, nil
}
