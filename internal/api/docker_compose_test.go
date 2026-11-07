package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/kkz6/launchctl/internal/config"
)

func TestDockerComposeContracts(t *testing.T) {
	const base = "/servers/server-1/docker/projects/project-1/composes"
	const stack = `{"id":"compose-1","server_id":"server-1","project_id":"project-1","name":"web","compose_source_type":"raw_yaml","raw_yaml":"services: {}","env_file":"KEY=value","status":"idle","build_location":"server","registry_credentials":[{"id":"registry-1","name":"Private","registry_url":"https://registry.test"}]}`
	const deployment = `{"id":"deploy-1","server_id":"server-1","target_type":"compose","target_id":"compose-1","status":"pending"}`
	empty := ""
	tests := []struct {
		name, method, path, query, body, data string
		status                                int
		call                                  func(*Client) (any, error)
	}{
		{"list", "GET", base, "", "", "[" + stack + "]", 200, func(c *Client) (any, error) {
			return c.ListDockerComposesContext(context.Background(), "server-1", "project-1")
		}},
		{"show", "GET", base + "/compose-1", "", "", stack, 200, func(c *Client) (any, error) {
			return c.GetDockerComposeContext(context.Background(), "server-1", "project-1", "compose-1")
		}},
		{"create", "POST", base, "", `{"name":"web","compose_source_type":"raw_yaml","raw_yaml":{"contents":"services: {}"}}`, stack, 201, func(c *Client) (any, error) {
			return c.CreateDockerComposeContext(context.Background(), "server-1", "project-1", CreateDockerComposeRequest{Name: "web", ComposeSourceType: "raw_yaml", RawYAML: &DockerComposeRawYAML{Contents: "services: {}"}})
		}},
		{"update clears only requested fields", "PATCH", base + "/compose-1", "", `{"env_file":"","run_command":""}`, stack, 200, func(c *Client) (any, error) {
			return c.UpdateDockerComposeContext(context.Background(), "server-1", "project-1", "compose-1", UpdateDockerComposeRequest{EnvFile: &empty, RunCommand: &empty})
		}},
		{"delete preserves volumes", "DELETE", base + "/compose-1", "", "", "", 204, func(c *Client) (any, error) {
			return nil, c.DeleteDockerComposeContext(context.Background(), "server-1", "project-1", "compose-1", false)
		}},
		{"delete removes volumes explicitly", "DELETE", base + "/compose-1", "remove_volumes=true", "", "", 204, func(c *Client) (any, error) {
			return nil, c.DeleteDockerComposeContext(context.Background(), "server-1", "project-1", "compose-1", true)
		}},
		{"deploy", "POST", base + "/compose-1/deploy", "", "", deployment, 201, func(c *Client) (any, error) {
			return c.DeployDockerComposeContext(context.Background(), "server-1", "project-1", "compose-1")
		}},
		{"reload returns no deployment", "POST", base + "/compose-1/reload", "", "", "null", 200, func(c *Client) (any, error) {
			return nil, c.ReloadDockerComposeContext(context.Background(), "server-1", "project-1", "compose-1")
		}},
		{"deployments", "GET", base + "/compose-1/deployments", "", "", "[" + deployment + "]", 200, func(c *Client) (any, error) {
			return c.ListDockerComposeDeploymentsContext(context.Background(), "server-1", "project-1", "compose-1")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				if r.Method != tt.method || r.URL.Path != tt.path || r.URL.RawQuery != tt.query {
					t.Errorf("request = %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				}
				if tt.body != "" {
					var got, want any
					if err := json.Unmarshal(body, &got); err != nil {
						t.Error(err)
					}
					_ = json.Unmarshal([]byte(tt.body), &want)
					if !reflect.DeepEqual(got, want) {
						t.Errorf("body = %s, want %s", body, tt.body)
					}
				} else if len(body) != 0 {
					t.Errorf("unexpected body: %s", body)
				}
				w.WriteHeader(tt.status)
				if tt.status != 204 {
					_, _ = io.WriteString(w, `{"success":true,"data":`+tt.data+`}`)
				}
			}))
			defer server.Close()
			got, err := tt.call(NewClient(&config.Config{APIURL: server.URL}))
			if err != nil {
				t.Fatal(err)
			}
			switch result := got.(type) {
			case *DockerComposeResponse:
				if result.ID != "compose-1" || result.EnvFile == nil || len(result.RegistryCredentials) != 1 {
					t.Fatalf("missing fields: %#v", result)
				}
			case *DockerDeploymentResponse:
				if result.ID != "deploy-1" || result.TargetType != "compose" {
					t.Fatalf("wrong deployment: %#v", result)
				}
			case []DockerDeploymentResponse:
				if len(result) != 1 || result[0].ID != "deploy-1" {
					t.Fatalf("wrong history: %#v", result)
				}
			case []DockerComposeResponse:
				if len(result) != 1 || result[0].ID != "compose-1" {
					t.Fatalf("wrong stacks: %#v", result)
				}
			}
		})
	}
}

func TestDockerComposeActionErrorsAreNotSuccess(t *testing.T) {
	for _, action := range []string{"delete", "reload", "deploy"} {
		t.Run(action, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusConflict)
				_, _ = io.WriteString(w, `{"success":false,"message":"deployment already running"}`)
			}))
			defer server.Close()
			client := NewClient(&config.Config{APIURL: server.URL})
			var err error
			switch action {
			case "delete":
				err = client.DeleteDockerComposeContext(context.Background(), "s", "p", "c", false)
			case "reload":
				err = client.ReloadDockerComposeContext(context.Background(), "s", "p", "c")
			case "deploy":
				_, err = client.DeployDockerComposeContext(context.Background(), "s", "p", "c")
			}
			var apiErr *APIError
			if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestDockerComposeContextCancelsInFlightRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	client := NewClient(&config.Config{APIURL: server.URL})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := client.ListDockerComposeDeploymentsContext(ctx, "s", "p", "c")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v", err)
	}
}
