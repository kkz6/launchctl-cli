package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configDir  = "launchctl"
	configFile = "config.json"
)

const APIURL = "https://launchctl.io"

type Config struct {
	AccessToken string `json:"access_token,omitempty"`
	TeamID      string `json:"team_id,omitempty"`
	TeamName    string `json:"team_name,omitempty"`
	UserID      string `json:"user_id,omitempty"`
	UserName    string `json:"user_name,omitempty"`
	UserEmail   string `json:"user_email,omitempty"`
}

func DefaultConfig() *Config {
	return &Config{}
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine home directory: %w", err)
	}

	return filepath.Join(home, ".config", configDir, configFile), nil
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}

func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func (c *Config) IsAuthenticated() bool {
	return c.AccessToken != ""
}

func (c *Config) ClearAuth() {
	c.AccessToken = ""
	c.UserID = ""
	c.UserName = ""
	c.UserEmail = ""
	c.TeamID = ""
	c.TeamName = ""
}

func (c *Config) Get(key string) string {
	switch key {
	case "access_token":
		return c.AccessToken
	case "team_id":
		return c.TeamID
	case "team_name":
		return c.TeamName
	case "user_id":
		return c.UserID
	case "user_name":
		return c.UserName
	case "user_email":
		return c.UserEmail
	default:
		return ""
	}
}

func (c *Config) Set(key, value string) error {
	switch key {
	case "team_id":
		c.TeamID = value
	case "team_name":
		c.TeamName = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return c.Save()
}
