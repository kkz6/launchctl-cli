package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/spf13/cobra"
)

func TestValidateApplicationCreateOptions(t *testing.T) {
	tests := []struct {
		name      string
		opts      applicationCreateOptions
		wantError string
	}{
		{
			name: "image",
			opts: applicationCreateOptions{name: "web", source: "image", port: 80, image: "nginx:1.27"},
		},
		{
			name: "git",
			opts: applicationCreateOptions{name: "api", source: "git", port: 3000, repo: "https://example.test/api.git", branch: "main", buildType: "dockerfile", dockerfilePath: "deploy/Dockerfile", buildLocation: "server"},
		},
		{
			name: "git via GitHub Actions",
			opts: applicationCreateOptions{name: "api", source: "git", port: 3000, repo: "https://example.test/api.git", branch: "main", buildLocation: "github_actions", sourceControl: "connection-1"},
		},
		{
			name: "inline Dockerfile",
			opts: applicationCreateOptions{name: "worker", source: "dockerfile", port: 8080, dockerfile: "Dockerfile"},
		},
		{
			name:      "missing name",
			opts:      applicationCreateOptions{source: "image", port: 80, image: "nginx:1.27"},
			wantError: "--name",
		},
		{
			name:      "invalid port",
			opts:      applicationCreateOptions{name: "web", source: "image", port: 70000, image: "nginx:1.27"},
			wantError: "--port",
		},
		{
			name:      "missing image",
			opts:      applicationCreateOptions{name: "web", source: "image", port: 80},
			wantError: "--image",
		},
		{
			name:      "image missing version",
			opts:      applicationCreateOptions{name: "web", source: "image", port: 80, image: "registry.example:5000/acme/web"},
			wantError: "tag or digest",
		},
		{
			name:      "mixed source flags",
			opts:      applicationCreateOptions{name: "web", source: "image", port: 80, image: "nginx:1.27", repo: "https://example.test/repo.git"},
			wantError: "cannot be used",
		},
		{
			name:      "git missing branch",
			opts:      applicationCreateOptions{name: "api", source: "git", port: 3000, repo: "https://example.test/api.git"},
			wantError: "--repo and --branch",
		},
		{
			name:      "invalid build location",
			opts:      applicationCreateOptions{name: "api", source: "git", port: 3000, repo: "https://example.test/api.git", branch: "main", buildLocation: "laptop"},
			wantError: "--build-location",
		},
		{
			name:      "GitHub Actions requires source control",
			opts:      applicationCreateOptions{name: "api", source: "git", port: 3000, repo: "https://example.test/api.git", branch: "main", buildLocation: "github_actions"},
			wantError: "--source-control",
		},
		{
			name:      "unknown source",
			opts:      applicationCreateOptions{name: "api", source: "compose", port: 80},
			wantError: "--source",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateApplicationCreateOptions(test.opts)
			if test.wantError == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %v, want text %q", err, test.wantError)
			}
		})
	}
}

func TestHasExplicitImageVersion(t *testing.T) {
	tests := map[string]bool{
		"nginx:1.27":                        true,
		"registry.example:5000/acme/web:v1": true,
		"acme/web@sha256:abcdef":            true,
		"nginx":                             false,
		"registry.example:5000/acme/web":    false,
		"nginx:":                            false,
		"acme/web@sha256:":                  false,
	}
	for value, want := range tests {
		if got := hasExplicitImageVersion(value); got != want {
			t.Errorf("hasExplicitImageVersion(%q) = %v, want %v", value, got, want)
		}
	}
}

func TestApplicationBuildLabelsSourceWithoutBuildStep(t *testing.T) {
	if got := applicationBuild(api.DockerApplicationResponse{SourceType: "image"}); got != "pre-built" {
		t.Fatalf("image build label = %q", got)
	}
	if got := applicationBuild(api.DockerApplicationResponse{SourceType: "dockerfile"}); got != "dockerfile" {
		t.Fatalf("Dockerfile build label = %q", got)
	}
}

func TestDockerApplicationOutputRedactsSourceCredentials(t *testing.T) {
	application := api.DockerApplicationResponse{
		SourceType: "git",
		SourceConfig: map[string]any{
			"repo":                  "https://user:secret@github.com/acme/api.git?access_token=token",
			"gha_deploy_token_hash": "sensitive-hash",
		},
	}
	sanitized := sanitizedDockerApplication(application)
	if got := sanitized.SourceConfig["repo"]; got != "https://github.com/acme/api.git?access_token=REDACTED" {
		t.Fatalf("sanitized repo = %q", got)
	}
	if got := sanitized.SourceConfig["gha_deploy_token_hash"]; got != "REDACTED" {
		t.Fatalf("sanitized token hash = %q", got)
	}
	if got := application.SourceConfig["gha_deploy_token_hash"]; got != "sensitive-hash" {
		t.Fatal("sanitization mutated the API response")
	}
	if got := applicationSource(&application); strings.Contains(got, "secret") || strings.Contains(got, "token=token") {
		t.Fatalf("human source output leaked credentials: %q", got)
	}
}

func TestValidateDockerWaitOptions(t *testing.T) {
	if err := validateDockerWaitOptions(false, 0); err != nil {
		t.Fatalf("unused zero timeout was rejected: %v", err)
	}
	if err := validateDockerWaitOptions(true, 0); err == nil {
		t.Fatal("zero wait timeout was accepted")
	}
	if err := validateDockerWaitOptions(true, 1); err != nil {
		t.Fatalf("positive wait timeout was rejected: %v", err)
	}
}

func TestBuildApplicationCreateRequestUsesOnlySelectedSource(t *testing.T) {
	cmd := &cobra.Command{}
	request, err := buildApplicationCreateRequest(cmd, applicationCreateOptions{
		name:               "web",
		source:             "image",
		port:               8080,
		image:              "ghcr.io/acme/web:v1",
		registryCredential: "credential-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if request.SourceType != "image" || request.Image == nil || request.Image.Image != "ghcr.io/acme/web:v1" {
		t.Fatalf("request = %#v", request)
	}
	if request.InternalPort == nil || *request.InternalPort != 8080 {
		t.Fatalf("internal port = %v", request.InternalPort)
	}
	if request.Image.RegistryCredentialID == nil || *request.Image.RegistryCredentialID != "credential-1" {
		t.Fatalf("registry credential = %v", request.Image.RegistryCredentialID)
	}
	if request.Git != nil || request.Dockerfile != nil {
		t.Fatalf("unselected source payloads were populated: %#v", request)
	}
}

func TestReadDockerfileFromStdinAndEnforcesLimit(t *testing.T) {
	got, err := readDockerfile(strings.NewReader("FROM caddy:2-alpine\n"), "-")
	if err != nil || got != "FROM caddy:2-alpine\n" {
		t.Fatalf("readDockerfile() = %q, %v", got, err)
	}
	if _, err := readDockerfile(bytes.NewReader(bytes.Repeat([]byte("x"), maxDockerfileBytes+1)), "-"); err == nil {
		t.Fatal("oversized Dockerfile was accepted")
	}
	if _, err := readDockerfile(strings.NewReader(" \n"), "-"); err == nil {
		t.Fatal("empty Dockerfile was accepted")
	}
}

func TestDockerDeploymentOutcome(t *testing.T) {
	message := "build failed"
	tests := []struct {
		status    string
		errorText *string
		terminal  bool
		wantError bool
	}{
		{status: "pending"},
		{status: "deploying"},
		{status: "success", terminal: true},
		{status: "failed", errorText: &message, terminal: true, wantError: true},
		{status: "cancelled", terminal: true, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.status, func(t *testing.T) {
			terminal, err := dockerDeploymentOutcome(&api.DockerDeploymentResponse{
				ID:     "deployment-1",
				Status: test.status,
				Error:  test.errorText,
			})
			if terminal != test.terminal || (err != nil) != test.wantError {
				t.Fatalf("outcome = (%v, %v), want terminal=%v error=%v", terminal, err, test.terminal, test.wantError)
			}
		})
	}
}

func TestValidateDockerApplicationUpdateRequiresExplicitDockerfileBuildType(t *testing.T) {
	path := "deploy/Dockerfile"
	if err := validateDockerApplicationUpdateRequest(api.UpdateDockerApplicationRequest{DockerfilePath: &path}); err == nil {
		t.Fatal("Dockerfile path without an explicit Dockerfile build type was accepted")
	}
	buildType := "dockerfile"
	if err := validateDockerApplicationUpdateRequest(api.UpdateDockerApplicationRequest{
		BuildType:      &buildType,
		DockerfilePath: &path,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestWaitForDockerApplicationRunningReconcilesDelayedStatus(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/servers/server-1/docker/projects/project-1/applications/app-1" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		status := "deploying"
		if calls.Add(1) > 1 {
			status = "running"
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"success":true,"data":{"id":"app-1","status":%q}}`, status)
	}))
	defer server.Close()

	client := api.NewClient(&config.Config{APIURL: server.URL})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	application, err := waitForDockerApplicationRunning(ctx, client, "server-1", "project-1", "app-1")
	if err != nil {
		t.Fatal(err)
	}
	if application.Status != "running" || calls.Load() < 2 {
		t.Fatalf("application = %#v, calls = %d", application, calls.Load())
	}
}

func TestWaitForDockerDeploymentHonorsContextDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client := api.NewClient(&config.Config{APIURL: server.URL})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := waitForDockerDeployment(ctx, client, "server-1", "project-1", "app-1", "deployment-1")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context deadline exceeded", err)
	}
}
