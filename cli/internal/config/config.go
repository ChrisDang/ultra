package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	configDir  = ".vibecloud"
	configFile = "config.json"
)

// CLIInfo tracks the install/auth state of a provider CLI.
type CLIInfo struct {
	Installed bool `json:"installed"`
	LoggedIn  bool `json:"logged_in"`
}

// Config holds the CLI configuration persisted in ~/.vibecloud/config.json.
type Config struct {
	CLIStatus map[string]CLIInfo `json:"cli_status,omitempty"`
}

// ConfigPath returns the absolute path to the config file.
func ConfigPath() string {
	dir := os.Getenv("VIBECLOUD_CONFIG_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		dir = filepath.Join(home, configDir)
	}
	return filepath.Join(dir, configFile)
}

// Load reads the config from disk. If the file does not exist, it returns a
// Config with default values and no error.
func Load() (*Config, error) {
	cfg := &Config{}

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes the config to disk, creating the directory if necessary.
func Save(cfg *Config) error {
	p := ConfigPath()
	dir := filepath.Dir(p)

	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(p, data, 0600)
}
