package config

import "testing"

func TestEffectiveAPIURLAndEnvironmentOverrides(t *testing.T) {
	cfg := DefaultConfig()
	if got := cfg.EffectiveAPIURL(); got != DefaultAPIURL {
		t.Fatalf("default API URL = %q", got)
	}

	t.Setenv("LAUNCHCTL_API_URL", "https://staging.launchctl.example/api")
	t.Setenv("LAUNCHCTL_TOKEN", "token")
	t.Setenv("LAUNCHCTL_TEAM_ID", "team-a")
	cfg.ApplyEnvOverrides()
	if cfg.EffectiveAPIURL() != "https://staging.launchctl.example/api" || cfg.AccessToken != "token" || cfg.TeamID != "team-a" {
		t.Fatalf("environment overrides not applied: %#v", cfg)
	}
}

func TestActivateProfileIncludesAPIURL(t *testing.T) {
	cfg := &Config{Profiles: map[string]*Profile{
		"staging": {APIURL: "https://staging.example", AccessToken: "secret", TeamID: "team-a"},
	}}
	if err := cfg.ActivateProfile("staging"); err != nil {
		t.Fatal(err)
	}
	if cfg.ActiveProfile != "staging" || cfg.APIURL != "https://staging.example" || cfg.AccessToken != "secret" {
		t.Fatalf("profile not activated: %#v", cfg)
	}
}
