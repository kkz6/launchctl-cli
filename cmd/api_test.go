package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAPIData(t *testing.T) {
	value, err := parseAPIData(`{"enabled":true}`)
	if err != nil || value == nil {
		t.Fatalf("inline JSON: value=%v err=%v", value, err)
	}

	file := filepath.Join(t.TempDir(), "body.json")
	if err := os.WriteFile(file, []byte(`[1,2,3]`), 0600); err != nil {
		t.Fatal(err)
	}
	if value, err = parseAPIData("@" + file); err != nil || value == nil {
		t.Fatalf("file JSON: value=%v err=%v", value, err)
	}
	if _, err := parseAPIData(`{"broken"`); err == nil {
		t.Fatal("invalid JSON was accepted")
	}
}

func TestAPIHelpDoesNotAdvertiseLegacyDockerProjectRoute(t *testing.T) {
	if strings.Contains(apiCmd.Example, "/api/docker/projects") {
		t.Fatalf("API help advertises the removed Docker project route: %s", apiCmd.Example)
	}
	if !strings.Contains(apiCmd.Example, "/api/servers/01ABC/backups") {
		t.Fatalf("API help is missing a valid nested server example: %s", apiCmd.Example)
	}
}
