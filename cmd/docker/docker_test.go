package docker

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/spf13/cobra"
)

func TestDockerCommandTreeAndAliases(t *testing.T) {
	root := newDockerCommand()
	projects := childCommand(t, root, "projects")
	applications := childCommand(t, root, "applications")

	if !contains(projects.Aliases, "project") {
		t.Fatalf("projects aliases = %v", projects.Aliases)
	}
	for _, alias := range []string{"application", "apps", "app"} {
		if !contains(applications.Aliases, alias) {
			t.Fatalf("applications aliases = %v, missing %q", applications.Aliases, alias)
		}
	}

	for _, name := range []string{"list", "show", "create", "update", "delete"} {
		childCommand(t, projects, name)
	}
	for _, name := range []string{"list", "show", "create", "update", "deploy", "reload", "start", "stop", "deployments", "delete"} {
		childCommand(t, applications, name)
	}

	if childCommand(t, projects, "create").Flags().Lookup("name") == nil {
		t.Fatal("projects create is missing --name")
	}
	appCreate := childCommand(t, applications, "create")
	for _, flag := range []string{"name", "source", "port", "image", "repo", "branch", "dockerfile"} {
		if appCreate.Flags().Lookup(flag) == nil {
			t.Fatalf("applications create is missing --%s", flag)
		}
	}
	appDelete := childCommand(t, applications, "delete")
	for _, flag := range []string{"yes", "remove-volumes"} {
		if appDelete.Flags().Lookup(flag) == nil {
			t.Fatalf("applications delete is missing --%s", flag)
		}
	}
}

func TestConfirmDestructiveRequiresYesOutsideInteractiveTerminal(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("ci", false, "")
	cmd.Flags().Bool("json", false, "")

	confirmed, err := confirmDestructive(cmd, true, "title", "description")
	if err != nil || !confirmed {
		t.Fatalf("--yes confirmation = %v, %v", confirmed, err)
	}

	if err := cmd.Flags().Set("ci", "true"); err != nil {
		t.Fatal(err)
	}
	confirmed, err = confirmDestructive(cmd, false, "title", "description")
	if err == nil || confirmed || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("CI confirmation = %v, %v", confirmed, err)
	}
}

func TestProjectsCreateJSONUsesNestedDockerRoute(t *testing.T) {
	var paths []string
	var createBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/servers/server-1":
			_, _ = io.WriteString(w, `{"success":true,"data":{"id":"server-1","name":"Docker","type":"docker"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/servers/server-1/docker/projects":
			createBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"success":true,"data":{"id":"project-1","server_id":"server-1","name":"production"}}`)
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"success":false,"message":"not found"}`)
		}
	}))
	defer server.Close()

	appstate.SetConfig(&config.Config{APIURL: server.URL})
	t.Cleanup(func() { appstate.SetConfig(nil) })
	root := newDockerCommand()
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("ci", false, "")
	root.SilenceErrors = true
	root.SilenceUsage = true
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"projects", "create", "--server", "server-1", "--name", "production", "--json", "--ci"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	wantPaths := []string{
		"GET /servers/server-1",
		"POST /servers/server-1/docker/projects",
	}
	if strings.Join(paths, "\n") != strings.Join(wantPaths, "\n") {
		t.Fatalf("requests = %v, want %v", paths, wantPaths)
	}
	var request api.CreateDockerProjectRequest
	if err := json.Unmarshal(createBody, &request); err != nil {
		t.Fatal(err)
	}
	if request.Name != "production" {
		t.Fatalf("create request = %#v", request)
	}
	if strings.Contains(stdout.String(), "\x1b[") {
		t.Fatalf("JSON output contains ANSI: %q", stdout.String())
	}
	var project api.DockerProjectResponse
	if err := json.Unmarshal(stdout.Bytes(), &project); err != nil {
		t.Fatalf("decode JSON output %q: %v", stdout.String(), err)
	}
	if project.ID != "project-1" || project.Name != "production" {
		t.Fatalf("project output = %#v", project)
	}
}

func childCommand(t *testing.T, parent *cobra.Command, name string) *cobra.Command {
	t.Helper()
	for _, child := range parent.Commands() {
		if child.Name() == name {
			return child
		}
	}
	t.Fatalf("%s is missing child command %q", parent.Name(), name)
	return nil
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
