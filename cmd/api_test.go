package cmd

import (
	"os"
	"path/filepath"
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
