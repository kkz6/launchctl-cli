package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadProjectSupportsSiteAndDockerContexts(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want ProjectConfig
	}{
		{
			name: "legacy site",
			yaml: "server: server-1\nsite: site-1\n",
			want: ProjectConfig{Server: "server-1", Site: "site-1"},
		},
		{
			name: "docker application",
			yaml: "server: server-2\ndocker_project: project-1\ndocker_application: app-1\n",
			want: ProjectConfig{
				Server:            "server-2",
				DockerProject:     "project-1",
				DockerApplication: "app-1",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, projectConfigFile), []byte(test.yaml), 0600); err != nil {
				t.Fatal(err)
			}
			withWorkingDirectory(t, dir)

			got, err := LoadProject()
			if err != nil {
				t.Fatal(err)
			}
			if *got != test.want {
				t.Fatalf("LoadProject() = %#v, want %#v", *got, test.want)
			}
		})
	}
}

func TestSaveProjectOmitsUnusedTargetKind(t *testing.T) {
	dir := t.TempDir()
	withWorkingDirectory(t, dir)

	if err := SaveProject(&ProjectConfig{
		Server:        "server-1",
		DockerProject: "project-1",
	}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, projectConfigFile))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"server: server-1", "docker_project: project-1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("saved config is missing %q:\n%s", want, text)
		}
	}
	for _, unwanted := range []string{"site:", "docker_application:"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("saved config unexpectedly contains %q:\n%s", unwanted, text)
		}
	}
}

func withWorkingDirectory(t *testing.T, dir string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}
