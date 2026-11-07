package sites

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/spf13/cobra"
)

func TestCreateSiteWithRepositoryAndProjectServer(t *testing.T) {
	setSiteProjectDefaults(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/servers/default-server/sites/" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		assertSiteRequestBody(t, r, map[string]any{
			"address": "app.example.com", "type": "laravel", "php_version": "php84", "web_folder": "public",
			"zero_downtime_deployment": true, "source_control_id": "source-1", "source_control_repositories_id": "123", "repository_branch": "main",
		})
		_, _ = io.WriteString(w, `{"success":true,"data":{"id":"site-1","address":"app.example.com","status":"installing"}}`)
	}))
	defer server.Close()
	setSiteTestConfig(t, server.URL)
	cmd, out := siteTestCommand(newCreateCommand())
	cmd.SetArgs([]string{"--address", "app.example.com", "--type", "laravel", "--php-version", "php84", "--web-folder", "public", "--zero-downtime", "--source-control", "source-1", "--repository-id", "123", "--branch", "main", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var site api.SiteResponse
	if err := json.Unmarshal(out.Bytes(), &site); err != nil || site.ID != "site-1" || site.Status != "installing" {
		t.Fatalf("output = %q, error = %v", out.String(), err)
	}
}

func TestUpdateSitePreservesOmittedAndExplicitValues(t *testing.T) {
	setSiteProjectDefaults(t)
	for _, test := range []struct {
		name string
		args []string
		path string
		body map[string]any
	}{
		{"default-and-clear", []string{"--web-folder=", "--branch=", "--queue-deployments=false"}, "/servers/default-server/sites/default-site", map[string]any{"web_folder": "", "repository_branch": "", "queue_deployments": false}},
		{"override-and-set", []string{"site-1", "--server", "server-1", "--php-version", "php84", "--release-retention", "3", "--queue-deployments"}, "/servers/server-1/sites/site-1", map[string]any{"php_version": "php84", "deployment_releases_retention": float64(3), "queue_deployments": true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut || r.URL.Path != test.path {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				assertSiteRequestBody(t, r, test.body)
				_, _ = io.WriteString(w, `{"success":true,"data":{"id":"site-1","php_version":"php83","pending_php_version":"php84"}}`)
			}))
			defer server.Close()
			setSiteTestConfig(t, server.URL)
			cmd, out := siteTestCommand(newUpdateCommand())
			cmd.SetArgs(append(test.args, "--json"))
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			var site api.SiteResponse
			if err := json.Unmarshal(out.Bytes(), &site); err != nil || site.PendingPHPVersion != "php84" {
				t.Fatalf("output = %q, error = %v", out.String(), err)
			}
		})
	}
}

func TestSiteInputValidationPreventsRequests(t *testing.T) {
	for _, test := range []struct {
		name        string
		constructor func() *cobra.Command
		args        []string
		want        string
	}{
		{"missing-address", newCreateCommand, []string{"--type", "laravel", "--php-version", "php84"}, "--address"},
		{"missing-type", newCreateCommand, []string{"--address", "a.example.com", "--php-version", "php84"}, "--type"},
		{"missing-php", newCreateCommand, []string{"--address", "a.example.com", "--type", "laravel"}, "--php-version"},
		{"unpaired-source", newCreateCommand, []string{"--address", "a.example.com", "--type", "laravel", "--php-version", "php84", "--source-control", "source-1"}, "provided together"},
		{"invalid-repository", newCreateCommand, []string{"--address", "a.example.com", "--type", "laravel", "--php-version", "php84", "--source-control", "source-1", "--repository-id", "invalid"}, "numeric repository"},
		{"unsupported-zero-downtime", newCreateCommand, []string{"--address", "a.example.com", "--type", "wordpress", "--php-version", "php84", "--zero-downtime"}, "does not support"},
		{"empty-update", newUpdateCommand, nil, "at least one setting"},
		{"invalid-retention", newUpdateCommand, []string{"--release-retention", "0"}, "between 1 and 50"},
		{"empty-php", newUpdateCommand, []string{"--php-version="}, "--php-version"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1) }))
			defer server.Close()
			setSiteTestConfig(t, server.URL)
			cmd, out := siteTestCommand(test.constructor())
			cmd.SetArgs(append(test.args, "--server", "server-1", "--json"))
			if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
			if requests.Load() != 0 || out.Len() != 0 {
				t.Fatalf("requests = %d, output = %q", requests.Load(), out.String())
			}
		})
	}
}

func TestDeleteSiteRequiresConfirmationAndReportsAsyncRequest(t *testing.T) {
	setSiteProjectDefaults(t)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodDelete || r.URL.Path != "/servers/default-server/sites/default-site" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"success":true,"message":"Site deletion initiated","data":null}`)
	}))
	defer server.Close()
	setSiteTestConfig(t, server.URL)
	for _, mode := range []string{"--json", "--ci"} {
		cmd, out := siteTestCommand(newDeleteCommand())
		cmd.SetArgs([]string{mode})
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "--yes") {
			t.Fatalf("error = %v, want confirmation required", err)
		}
		if requests.Load() != 0 || out.Len() != 0 {
			t.Fatalf("requests = %d, output = %q", requests.Load(), out.String())
		}
	}
	cmd, out := siteTestCommand(newDeleteCommand())
	cmd.SetArgs([]string{"--yes", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil || result["deletion_requested"] != true || result["site_id"] != "default-site" || result["deleted"] != nil || requests.Load() != 1 {
		t.Fatalf("output = %q, requests = %d, error = %v", out.String(), requests.Load(), err)
	}
}

func TestAutoDeployUsesDefaultSite(t *testing.T) {
	setSiteProjectDefaults(t)
	for _, action := range []string{"enable", "disable"} {
		t.Run(action, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/servers/default-server/sites/default-site/auto-deployment/"+action {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				_, _ = io.WriteString(w, `{"success":true,"data":null}`)
			}))
			defer server.Close()
			setSiteTestConfig(t, server.URL)
			cmd := newAutoDeployCommand()
			cmd.PersistentFlags().Bool("json", false, "")
			cmd.SilenceErrors = true
			out := &bytes.Buffer{}
			cmd.SetOut(out)
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs([]string{action, "--json"})
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			var result map[string]any
			if err := json.Unmarshal(out.Bytes(), &result); err != nil || result["auto_deployment"] != (action == "enable") {
				t.Fatalf("output = %q, error = %v", out.String(), err)
			}
		})
	}
}

func TestDeleteSiteReportsAPIErrorAndUsesCommandContext(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"success":false,"message":"permission denied"}`)
	}))
	defer server.Close()
	setSiteTestConfig(t, server.URL)
	cmd, out := siteTestCommand(newDeleteCommand())
	cmd.SetArgs([]string{"site-1", "--server", "server-1", "--yes", "--json"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("error = %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("error emitted success output: %q", out.String())
	}
	cmd, _ = siteTestCommand(newDeleteCommand())
	cmd.SetArgs([]string{"site-1", "--server", "server-1", "--yes", "--json"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := cmd.ExecuteContext(ctx); !errors.Is(err, context.Canceled) || requests.Load() != 1 {
		t.Fatalf("error = %v, requests = %d", err, requests.Load())
	}
}

func setSiteProjectDefaults(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := config.SaveProject(&config.ProjectConfig{Server: "default-server", Site: "default-site"}); err != nil {
		t.Fatal(err)
	}
}

func setSiteTestConfig(t *testing.T, url string) {
	t.Helper()
	previous := appstate.GetConfig()
	appstate.SetConfig(&config.Config{APIURL: url})
	t.Cleanup(func() { appstate.SetConfig(previous) })
}

func siteTestCommand(cmd *cobra.Command) (*cobra.Command, *bytes.Buffer) {
	cmd.Flags().Bool("json", false, "")
	cmd.Flags().Bool("ci", false, "")
	cmd.SilenceErrors = true
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	return cmd, out
}

func assertSiteRequestBody(t *testing.T, r *http.Request, want map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("request body = %#v, want %#v", got, want)
	}
}
