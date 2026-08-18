package resolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDockerIDsUseFlagsBeforeProjectDefaults(t *testing.T) {
	dir := t.TempDir()
	data := []byte("server: server-default\ndocker_project: project-default\ndocker_application: app-default\n")
	if err := os.WriteFile(filepath.Join(dir, ".launchctl.yml"), data, 0600); err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	if got, err := ProjectID(""); err != nil || got != "project-default" {
		t.Fatalf("ProjectID default = %q, %v", got, err)
	}
	if got, err := ApplicationID(""); err != nil || got != "app-default" {
		t.Fatalf("ApplicationID default = %q, %v", got, err)
	}
	if got, err := ProjectID("project-flag"); err != nil || got != "project-flag" {
		t.Fatalf("ProjectID flag = %q, %v", got, err)
	}
	if got, err := ApplicationID("app-flag"); err != nil || got != "app-flag" {
		t.Fatalf("ApplicationID flag = %q, %v", got, err)
	}
}

func TestDockerIDErrorsNameRequiredFlags(t *testing.T) {
	dir := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	if _, err := ProjectID(""); err == nil || !strings.Contains(err.Error(), "--project") {
		t.Fatalf("ProjectID error = %v", err)
	}
	if _, err := ApplicationID(""); err == nil || !strings.Contains(err.Error(), "--application") {
		t.Fatalf("ApplicationID error = %v", err)
	}
}
