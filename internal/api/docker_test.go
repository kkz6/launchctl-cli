package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kkz6/launchctl/internal/config"
)

type dockerCapturedRequest struct {
	Method   string
	Path     string
	RawQuery string
	Body     string
}

func TestDockerProjectsResolveAPINamespaceAgainstRootBackend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/servers/server-1/docker/projects" {
			t.Fatalf("path = %q, want host-root Docker projects route", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"success":true,"data":[]}`)
	}))
	defer server.Close()

	client := NewClient(&config.Config{APIURL: server.URL})
	projects, err := client.ListDockerProjects("server-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 0 {
		t.Fatalf("projects = %#v, want empty", projects)
	}
}

func TestDockerDeploymentListContextCancelsInFlightRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client := NewClient(&config.Config{APIURL: server.URL})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.ListDockerApplicationDeploymentsContext(ctx, "server-1", "project-1", "app-1")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context deadline exceeded", err)
	}
}

func TestDockerProjectCRUD(t *testing.T) {
	project := `{
		"id":"project-1",
		"team_id":"team-1",
		"server_id":"server-1",
		"name":"production",
		"description":"Production workloads",
		"applications_count":2,
		"composes_count":1,
		"databases_count":1,
		"created_at":"2026-08-18T04:30:00Z",
		"updated_at":"2026-08-18T04:31:00Z"
	}`
	description := "Production workloads"
	updatedName := "renamed"
	clearedDescription := ""

	tests := []struct {
		name         string
		status       int
		responseData string
		wantMethod   string
		wantPath     string
		wantBody     string
		call         func(*Client) error
	}{
		{
			name:         "list",
			status:       http.StatusOK,
			responseData: "[" + project + "]",
			wantMethod:   http.MethodGet,
			wantPath:     "/api/servers/server-1/docker/projects",
			call: func(client *Client) error {
				projects, err := client.ListDockerProjects("server-1")
				if err == nil && (len(projects) != 1 || projects[0].ID != "project-1") {
					t.Fatalf("unexpected projects: %#v", projects)
				}
				if err == nil && (projects[0].CreatedAt == nil || projects[0].ApplicationsCount != 2) {
					t.Fatalf("project fields not decoded: %#v", projects[0])
				}
				return err
			},
		},
		{
			name:         "get",
			status:       http.StatusOK,
			responseData: project,
			wantMethod:   http.MethodGet,
			wantPath:     "/api/servers/server-1/docker/projects/project-1",
			call: func(client *Client) error {
				got, err := client.GetDockerProject("server-1", "project-1")
				if err == nil && got.ID != "project-1" {
					t.Fatalf("project ID = %q", got.ID)
				}
				return err
			},
		},
		{
			name:         "create",
			status:       http.StatusCreated,
			responseData: project,
			wantMethod:   http.MethodPost,
			wantPath:     "/api/servers/server-1/docker/projects",
			wantBody:     `{"name":"production","description":"Production workloads"}`,
			call: func(client *Client) error {
				got, err := client.CreateDockerProject("server-1", CreateDockerProjectRequest{
					Name:        "production",
					Description: &description,
				})
				if err == nil && got.ID != "project-1" {
					t.Fatalf("project ID = %q", got.ID)
				}
				return err
			},
		},
		{
			name:         "update",
			status:       http.StatusOK,
			responseData: project,
			wantMethod:   http.MethodPatch,
			wantPath:     "/api/servers/server-1/docker/projects/project-1",
			wantBody:     `{"name":"renamed","description":""}`,
			call: func(client *Client) error {
				got, err := client.UpdateDockerProject("server-1", "project-1", UpdateDockerProjectRequest{
					Name:        &updatedName,
					Description: &clearedDescription,
				})
				if err == nil && got.ID != "project-1" {
					t.Fatalf("project ID = %q", got.ID)
				}
				return err
			},
		},
		{
			name:       "delete",
			status:     http.StatusNoContent,
			wantMethod: http.MethodDelete,
			wantPath:   "/api/servers/server-1/docker/projects/project-1",
			call: func(client *Client) error {
				return client.DeleteDockerProject("server-1", "project-1")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, requests := newDockerTestClient(t, test.status, test.responseData)
			if err := test.call(client); err != nil {
				t.Fatal(err)
			}
			assertDockerRequest(t, requests, test.wantMethod, test.wantPath, "", test.wantBody)
		})
	}
}

func TestDockerApplicationCRUD(t *testing.T) {
	application := `{
		"id":"app-1",
		"team_id":"team-1",
		"server_id":"server-1",
		"project_id":"project-1",
		"name":"api",
		"internal_port":3000,
		"source_type":"git",
		"source_config":{"repo":"https://github.com/acme/api.git","branch":"main"},
		"build_type":"dockerfile",
		"build_config":{"dockerfile_path":"services/api/Dockerfile"},
		"status":"running",
		"container_id":"abc123",
		"container_name":"launch-production-api",
		"last_deployed_at":"2026-08-18T04:30:00Z",
		"created_at":"2026-08-18T04:00:00Z",
		"updated_at":"2026-08-18T04:30:00Z",
		"build_location":"server",
		"gha_build_ready":false,
		"gha_out_of_sync":false,
		"gha_pending_changes":0
	}`
	port := 3000
	credentialID := "credential-1"
	sourceControlID := "source-control-1"
	buildType := "dockerfile"
	dockerfilePath := "services/api/Dockerfile"
	buildLocation := "server"
	updatedName := "api-v2"

	tests := []struct {
		name         string
		status       int
		responseData string
		wantMethod   string
		wantPath     string
		wantQuery    string
		wantBody     string
		call         func(*Client) error
	}{
		{
			name:         "list",
			status:       http.StatusOK,
			responseData: "[" + application + "]",
			wantMethod:   http.MethodGet,
			wantPath:     "/api/servers/server-1/docker/projects/project-1/applications",
			call: func(client *Client) error {
				apps, err := client.ListDockerApplications("server-1", "project-1")
				if err == nil && (len(apps) != 1 || apps[0].ID != "app-1") {
					t.Fatalf("unexpected applications: %#v", apps)
				}
				if err == nil && (apps[0].SourceConfig["branch"] != "main" || apps[0].LastDeployedAt == nil) {
					t.Fatalf("application fields not decoded: %#v", apps[0])
				}
				return err
			},
		},
		{
			name:         "get",
			status:       http.StatusOK,
			responseData: application,
			wantMethod:   http.MethodGet,
			wantPath:     "/api/servers/server-1/docker/projects/project-1/applications/app-1",
			call: func(client *Client) error {
				got, err := client.GetDockerApplication("server-1", "project-1", "app-1")
				if err == nil && got.ContainerName != "launch-production-api" {
					t.Fatalf("container name = %q", got.ContainerName)
				}
				return err
			},
		},
		{
			name:         "create image source",
			status:       http.StatusCreated,
			responseData: application,
			wantMethod:   http.MethodPost,
			wantPath:     "/api/servers/server-1/docker/projects/project-1/applications",
			wantBody:     `{"name":"web","internal_port":3000,"source_type":"image","image":{"image":"ghcr.io/acme/web:v1","registry_credential_id":"credential-1"}}`,
			call: func(client *Client) error {
				_, err := client.CreateDockerApplication("server-1", "project-1", CreateDockerApplicationRequest{
					Name:         "web",
					InternalPort: &port,
					SourceType:   "image",
					Image: &DockerImageSourceInput{
						Image:                "ghcr.io/acme/web:v1",
						RegistryCredentialID: &credentialID,
					},
				})
				return err
			},
		},
		{
			name:         "create git source",
			status:       http.StatusCreated,
			responseData: application,
			wantMethod:   http.MethodPost,
			wantPath:     "/api/servers/server-1/docker/projects/project-1/applications",
			wantBody:     `{"name":"api","internal_port":3000,"source_type":"git","git":{"repo":"https://github.com/acme/api.git","branch":"main","source_control_id":"source-control-1","build_type":"dockerfile","dockerfile_path":"services/api/Dockerfile","build_location":"server"}}`,
			call: func(client *Client) error {
				_, err := client.CreateDockerApplication("server-1", "project-1", CreateDockerApplicationRequest{
					Name:         "api",
					InternalPort: &port,
					SourceType:   "git",
					Git: &DockerGitSourceInput{
						Repo:            "https://github.com/acme/api.git",
						Branch:          "main",
						SourceControlID: &sourceControlID,
						BuildType:       &buildType,
						DockerfilePath:  &dockerfilePath,
						BuildLocation:   &buildLocation,
					},
				})
				return err
			},
		},
		{
			name:         "create dockerfile source",
			status:       http.StatusCreated,
			responseData: application,
			wantMethod:   http.MethodPost,
			wantPath:     "/api/servers/server-1/docker/projects/project-1/applications",
			wantBody:     `{"name":"worker","source_type":"dockerfile","dockerfile":{"contents":"FROM alpine:3.22\nCMD [\"true\"]"}}`,
			call: func(client *Client) error {
				_, err := client.CreateDockerApplication("server-1", "project-1", CreateDockerApplicationRequest{
					Name:       "worker",
					SourceType: "dockerfile",
					Dockerfile: &DockerfileSourceInput{Contents: "FROM alpine:3.22\nCMD [\"true\"]"},
				})
				return err
			},
		},
		{
			name:         "update",
			status:       http.StatusOK,
			responseData: application,
			wantMethod:   http.MethodPatch,
			wantPath:     "/api/servers/server-1/docker/projects/project-1/applications/app-1",
			wantBody:     `{"name":"api-v2","build_type":"dockerfile","dockerfile_path":"services/api/Dockerfile"}`,
			call: func(client *Client) error {
				_, err := client.UpdateDockerApplication("server-1", "project-1", "app-1", UpdateDockerApplicationRequest{
					Name:           &updatedName,
					BuildType:      &buildType,
					DockerfilePath: &dockerfilePath,
				})
				return err
			},
		},
		{
			name:       "delete preserves volumes",
			status:     http.StatusNoContent,
			wantMethod: http.MethodDelete,
			wantPath:   "/api/servers/server-1/docker/projects/project-1/applications/app-1",
			call: func(client *Client) error {
				return client.DeleteDockerApplication("server-1", "project-1", "app-1", false)
			},
		},
		{
			name:       "delete removes volumes only when requested",
			status:     http.StatusNoContent,
			wantMethod: http.MethodDelete,
			wantPath:   "/api/servers/server-1/docker/projects/project-1/applications/app-1",
			wantQuery:  "remove_volumes=true",
			call: func(client *Client) error {
				return client.DeleteDockerApplication("server-1", "project-1", "app-1", true)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, requests := newDockerTestClient(t, test.status, test.responseData)
			if err := test.call(client); err != nil {
				t.Fatal(err)
			}
			assertDockerRequest(t, requests, test.wantMethod, test.wantPath, test.wantQuery, test.wantBody)
		})
	}
}

func TestDockerApplicationDeploymentAndActions(t *testing.T) {
	deployment := `{
		"id":"deployment-1",
		"team_id":"team-1",
		"server_id":"server-1",
		"target_type":"application",
		"target_id":"app-1",
		"status":"pending",
		"task_id":"task-1",
		"commit_sha":"abcdef123456",
		"commit_msg":"Release",
		"image_ref":"ghcr.io/acme/api:v1",
		"started_at":"2026-08-18T04:30:00Z",
		"created_at":"2026-08-18T04:30:00Z",
		"updated_at":"2026-08-18T04:30:00Z",
		"trigger_source":"manual"
	}`

	t.Run("deploy", func(t *testing.T) {
		client, requests := newDockerTestClient(t, http.StatusCreated, deployment)
		got, err := client.DeployDockerApplication("server-1", "project-1", "app-1")
		if err != nil {
			t.Fatal(err)
		}
		if got.ID != "deployment-1" || got.Status != "pending" || got.StartedAt == nil {
			t.Fatalf("unexpected deployment: %#v", got)
		}
		assertDockerRequest(t, requests, http.MethodPost, "/api/servers/server-1/docker/projects/project-1/applications/app-1/deploy", "", "")
	})

	t.Run("list deployments", func(t *testing.T) {
		client, requests := newDockerTestClient(t, http.StatusOK, "["+deployment+"]")
		got, err := client.ListDockerApplicationDeployments("server-1", "project-1", "app-1")
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].TaskID == nil || *got[0].TaskID != "task-1" {
			t.Fatalf("unexpected deployments: %#v", got)
		}
		assertDockerRequest(t, requests, http.MethodGet, "/api/servers/server-1/docker/projects/project-1/applications/app-1/deployments", "", "")
	})

	for _, action := range []string{"reload", "start", "stop"} {
		t.Run(action, func(t *testing.T) {
			responseAction := action
			if action == "reload" {
				responseAction = "restart"
			}
			client, requests := newDockerTestClient(t, http.StatusOK, `{"action":"`+responseAction+`"}`)
			if err := client.DockerApplicationAction("server-1", "project-1", "app-1", action); err != nil {
				t.Fatal(err)
			}
			assertDockerRequest(t, requests, http.MethodPost, "/api/servers/server-1/docker/projects/project-1/applications/app-1/"+action, "", "")
		})
	}

	t.Run("rejects unsupported action before request", func(t *testing.T) {
		client := NewClient(&config.Config{APIURL: "https://example.invalid"})
		err := client.DockerApplicationAction("server-1", "project-1", "app-1", "destroy")
		if err == nil || !strings.Contains(err.Error(), "unsupported Docker application action") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestDockerClientReturnsTypedAPIErrors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		response   string
		call       func(*Client) error
		wantStatus int
		wantErrors map[string][]string
	}{
		{
			name:       "404 from project show",
			status:     http.StatusNotFound,
			response:   `{"success":false,"message":"Resource not found"}`,
			call:       func(client *Client) error { _, err := client.GetDockerProject("server-1", "missing"); return err },
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "422 from project delete",
			status:     http.StatusUnprocessableEntity,
			response:   `{"success":false,"message":"Project still has applications"}`,
			call:       func(client *Client) error { return client.DeleteDockerProject("server-1", "project-1") },
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:     "422 validation fields from application create",
			status:   http.StatusUnprocessableEntity,
			response: `{"success":false,"message":"Validation failed","errors":{"source_type":["The source type field is required."]}}`,
			call: func(client *Client) error {
				_, err := client.CreateDockerApplication("server-1", "project-1", CreateDockerApplicationRequest{Name: "api"})
				return err
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantErrors: map[string][]string{"source_type": {"The source type field is required."}},
		},
		{
			name:     "404 from application delete",
			status:   http.StatusNotFound,
			response: `{"success":false,"message":"Resource not found"}`,
			call: func(client *Client) error {
				return client.DeleteDockerApplication("server-1", "project-1", "missing", false)
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, requests := newDockerTestClient(t, test.status, test.response)
			err := test.call(client)
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error = %v, want *APIError", err)
			}
			if apiErr.StatusCode != test.wantStatus {
				t.Fatalf("status = %d, want %d", apiErr.StatusCode, test.wantStatus)
			}
			if !reflect.DeepEqual(apiErr.Errors, test.wantErrors) {
				t.Fatalf("errors = %#v, want %#v", apiErr.Errors, test.wantErrors)
			}
			<-requests
		})
	}
}

func newDockerTestClient(t *testing.T, status int, responseData string) (*Client, <-chan dockerCapturedRequest) {
	t.Helper()

	requests := make(chan dockerCapturedRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests <- dockerCapturedRequest{
			Method:   r.Method,
			Path:     r.URL.Path,
			RawQuery: r.URL.RawQuery,
			Body:     string(body),
		}

		if status == http.StatusNoContent {
			w.WriteHeader(status)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if strings.HasPrefix(strings.TrimSpace(responseData), `{"success":`) {
			_, _ = io.WriteString(w, responseData)
			return
		}
		_, _ = io.WriteString(w, `{"success":true,"message":"ok","data":`+responseData+`}`)
	}))
	t.Cleanup(server.Close)

	client := NewClientWithHTTPClient(&config.Config{
		APIURL:      server.URL + "/api",
		AccessToken: "token",
		TeamID:      "team-1",
	}, server.Client())
	return client, requests
}

func assertDockerRequest(
	t *testing.T,
	requests <-chan dockerCapturedRequest,
	wantMethod, wantPath, wantQuery, wantBody string,
) {
	t.Helper()

	got := <-requests
	if got.Method != wantMethod {
		t.Errorf("method = %q, want %q", got.Method, wantMethod)
	}
	if got.Path != wantPath {
		t.Errorf("path = %q, want %q", got.Path, wantPath)
	}
	if got.RawQuery != wantQuery {
		t.Errorf("query = %q, want %q", got.RawQuery, wantQuery)
	}
	assertDockerJSON(t, got.Body, wantBody)
}

func assertDockerJSON(t *testing.T, got, want string) {
	t.Helper()

	if want == "" {
		if strings.TrimSpace(got) != "" {
			t.Errorf("body = %q, want empty", got)
		}
		return
	}

	var gotValue any
	if err := json.Unmarshal([]byte(got), &gotValue); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	var wantValue any
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("decode expected body: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Errorf("body = %s, want %s", got, want)
	}
}
