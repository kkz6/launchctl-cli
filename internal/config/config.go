package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	configDir  = "launchctl"
	configFile = "config.json"
)

const (
	DefaultAPIURL   = "https://api.launchctl.io"
	legacyHostedURL = "https://launchctl.io"
)

type Favorite struct {
	ServerID    string `json:"server_id"`
	ServerName  string `json:"server_name"`
	SiteID      string `json:"site_id"`
	SiteAddress string `json:"site_address"`
}

type Profile struct {
	APIURL      string     `json:"api_url,omitempty"`
	AccessToken string     `json:"access_token,omitempty"`
	TeamID      string     `json:"team_id,omitempty"`
	TeamName    string     `json:"team_name,omitempty"`
	UserID      string     `json:"user_id,omitempty"`
	UserName    string     `json:"user_name,omitempty"`
	UserEmail   string     `json:"user_email,omitempty"`
	Favorites   []Favorite `json:"favorites,omitempty"`
}

type Config struct {
	ActiveProfile string              `json:"active_profile,omitempty"`
	Profiles      map[string]*Profile `json:"profiles,omitempty"`

	// Flat fields populated from the active profile for convenience.
	// These are the fields all existing code uses.
	APIURL      string     `json:"api_url,omitempty"`
	AccessToken string     `json:"access_token,omitempty"`
	TeamID      string     `json:"team_id,omitempty"`
	TeamName    string     `json:"team_name,omitempty"`
	UserID      string     `json:"user_id,omitempty"`
	UserName    string     `json:"user_name,omitempty"`
	UserEmail   string     `json:"user_email,omitempty"`
	Favorites   []Favorite `json:"favorites,omitempty"`
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

	// Migrate: if we have flat credentials but no profiles, create a "default" profile
	if cfg.ActiveProfile == "" && cfg.AccessToken != "" && len(cfg.Profiles) == 0 {
		cfg.Profiles = map[string]*Profile{
			"default": {
				APIURL:      cfg.APIURL,
				AccessToken: cfg.AccessToken,
				TeamID:      cfg.TeamID,
				TeamName:    cfg.TeamName,
				UserID:      cfg.UserID,
				UserName:    cfg.UserName,
				UserEmail:   cfg.UserEmail,
				Favorites:   cfg.Favorites,
			},
		}
		cfg.ActiveProfile = "default"
	}

	// If we have profiles, load the active one into flat fields
	if cfg.ActiveProfile != "" && cfg.Profiles != nil {
		if p, ok := cfg.Profiles[cfg.ActiveProfile]; ok {
			cfg.APIURL = p.APIURL
			cfg.AccessToken = p.AccessToken
			cfg.TeamID = p.TeamID
			cfg.TeamName = p.TeamName
			cfg.UserID = p.UserID
			cfg.UserName = p.UserName
			cfg.UserEmail = p.UserEmail
			cfg.Favorites = p.Favorites
		}
	}

	return cfg, nil
}

func (c *Config) Save() error {
	// Sync flat fields back to the active profile before saving
	if c.ActiveProfile != "" && c.Profiles != nil {
		if p, ok := c.Profiles[c.ActiveProfile]; ok {
			p.APIURL = c.APIURL
			p.AccessToken = c.AccessToken
			p.TeamID = c.TeamID
			p.TeamName = c.TeamName
			p.UserID = c.UserID
			p.UserName = c.UserName
			p.UserEmail = c.UserEmail
			p.Favorites = c.Favorites
		}
	}

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
	case "api_url":
		return c.EffectiveAPIURL()
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
	case "api_url":
		c.APIURL = value
	case "team_id":
		c.TeamID = value
	case "team_name":
		c.TeamName = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return c.Save()
}

func (c *Config) AddFavorite(fav Favorite) error {
	for _, f := range c.Favorites {
		if f.SiteID == fav.SiteID {
			return nil
		}
	}
	c.Favorites = append(c.Favorites, fav)
	return c.Save()
}

func (c *Config) RemoveFavorite(siteID string) error {
	for i, f := range c.Favorites {
		if f.SiteID == siteID {
			c.Favorites = append(c.Favorites[:i], c.Favorites[i+1:]...)
			return c.Save()
		}
	}
	return nil
}

func (c *Config) IsFavorite(siteID string) bool {
	for _, f := range c.Favorites {
		if f.SiteID == siteID {
			return true
		}
	}
	return false
}

func (c *Config) ApplyEnvOverrides() {
	if apiURL := os.Getenv("LAUNCHCTL_API_URL"); apiURL != "" {
		c.APIURL = apiURL
	}

	if token := os.Getenv("LAUNCHCTL_TOKEN"); token != "" {
		c.AccessToken = token
	}

	if teamID := os.Getenv("LAUNCHCTL_TEAM_ID"); teamID != "" {
		c.TeamID = teamID
	}
}

func (c *Config) ActiveProfileName() string {
	if c.ActiveProfile != "" {
		return c.ActiveProfile
	}
	return "default"
}

func (c *Config) ListProfiles() []string {
	var names []string
	for name := range c.Profiles {
		names = append(names, name)
	}
	return names
}

func (c *Config) AddProfile(name string, profile *Profile) error {
	if c.Profiles == nil {
		c.Profiles = make(map[string]*Profile)
	}

	c.Profiles[name] = profile
	return c.Save()
}

func (c *Config) UseProfile(name string) error {
	if c.Profiles == nil {
		return fmt.Errorf("no profiles configured")
	}

	p, ok := c.Profiles[name]
	if !ok {
		return fmt.Errorf("profile %q not found", name)
	}

	c.ActiveProfile = name
	c.APIURL = p.APIURL
	c.AccessToken = p.AccessToken
	c.TeamID = p.TeamID
	c.TeamName = p.TeamName
	c.UserID = p.UserID
	c.UserName = p.UserName
	c.UserEmail = p.UserEmail
	c.Favorites = p.Favorites

	return c.Save()
}

// ActivateProfile loads a profile into the flat fields for the current
// process without persisting the change to disk. Used for --profile flag.
func (c *Config) ActivateProfile(name string) error {
	if c.Profiles == nil {
		return fmt.Errorf("no profiles configured")
	}

	p, ok := c.Profiles[name]
	if !ok {
		return fmt.Errorf("profile %q not found", name)
	}

	c.ActiveProfile = name
	c.APIURL = p.APIURL
	c.AccessToken = p.AccessToken
	c.TeamID = p.TeamID
	c.TeamName = p.TeamName
	c.UserID = p.UserID
	c.UserName = p.UserName
	c.UserEmail = p.UserEmail
	c.Favorites = p.Favorites

	return nil
}

// EffectiveAPIURL returns the configured API origin, falling back to the
// hosted launchctl endpoint. Keeping the fallback here ensures HTTP and
// WebSocket clients always resolve the same endpoint.
func (c *Config) EffectiveAPIURL() string {
	if c != nil && c.APIURL != "" {
		// Profiles created before the API moved to its own host still contain
		// the frontend origin. Treat those values as the hosted default so an
		// existing login starts working without requiring a config rewrite.
		legacyURL := strings.TrimRight(c.APIURL, "/")
		if legacyURL == legacyHostedURL || legacyURL == legacyHostedURL+"/api" {
			return DefaultAPIURL
		}
		return c.APIURL
	}
	return DefaultAPIURL
}

func (c *Config) RemoveProfile(name string) error {
	if c.Profiles == nil {
		return fmt.Errorf("no profiles configured")
	}

	if _, ok := c.Profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}

	if c.ActiveProfile == name {
		return fmt.Errorf("cannot remove the active profile — switch to another profile first")
	}

	delete(c.Profiles, name)
	return c.Save()
}
