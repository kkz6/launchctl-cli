package api

import "time"

// DockerProjectResponse is a Docker workload project on a server.
type DockerProjectResponse struct {
	ID                string     `json:"id"`
	TeamID            string     `json:"team_id"`
	ServerID          string     `json:"server_id"`
	Name              string     `json:"name"`
	Description       *string    `json:"description,omitempty"`
	ApplicationsCount int64      `json:"applications_count"`
	ComposesCount     int64      `json:"composes_count"`
	DatabasesCount    int64      `json:"databases_count"`
	CreatedAt         *time.Time `json:"created_at,omitempty"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`
}

type CreateDockerProjectRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

type UpdateDockerProjectRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// CreateDockerApplicationRequest uses SourceType to select exactly one of
// Image, Git, or Dockerfile.
type CreateDockerApplicationRequest struct {
	Name         string                  `json:"name"`
	InternalPort *int                    `json:"internal_port,omitempty"`
	SourceType   string                  `json:"source_type"`
	Image        *DockerImageSourceInput `json:"image,omitempty"`
	Git          *DockerGitSourceInput   `json:"git,omitempty"`
	Dockerfile   *DockerfileSourceInput  `json:"dockerfile,omitempty"`
}

type DockerImageSourceInput struct {
	Image                string  `json:"image"`
	RegistryCredentialID *string `json:"registry_credential_id,omitempty"`
	RegistryUsername     *string `json:"registry_username,omitempty"`
	RegistryPassword     *string `json:"registry_password,omitempty"`
	RegistryURL          *string `json:"registry_url,omitempty"`
}

type DockerGitSourceInput struct {
	Repo            string  `json:"repo"`
	Branch          string  `json:"branch"`
	SourceControlID *string `json:"source_control_id,omitempty"`
	BuildType       *string `json:"build_type,omitempty"`
	DockerfilePath  *string `json:"dockerfile_path,omitempty"`
	BuildLocation   *string `json:"build_location,omitempty"`
}

type DockerfileSourceInput struct {
	Contents string `json:"contents"`
}

type UpdateDockerApplicationRequest struct {
	Name           *string `json:"name,omitempty"`
	BuildType      *string `json:"build_type,omitempty"`
	DockerfilePath *string `json:"dockerfile_path,omitempty"`
}

// DockerApplicationResponse is a single-container application nested under a
// Docker project. SourceConfig and BuildConfig are source-specific JSON maps.
type DockerApplicationResponse struct {
	ID                string         `json:"id"`
	TeamID            string         `json:"team_id"`
	ServerID          string         `json:"server_id"`
	ProjectID         string         `json:"project_id"`
	Name              string         `json:"name"`
	InternalPort      int            `json:"internal_port"`
	SourceType        string         `json:"source_type"`
	SourceConfig      map[string]any `json:"source_config,omitempty"`
	BuildType         *string        `json:"build_type,omitempty"`
	BuildConfig       map[string]any `json:"build_config,omitempty"`
	Status            string         `json:"status"`
	ContainerID       *string        `json:"container_id,omitempty"`
	ContainerName     string         `json:"container_name,omitempty"`
	LastDeployedAt    *time.Time     `json:"last_deployed_at,omitempty"`
	CreatedAt         *time.Time     `json:"created_at,omitempty"`
	UpdatedAt         *time.Time     `json:"updated_at,omitempty"`
	BuildLocation     string         `json:"build_location"`
	GHABuildReady     bool           `json:"gha_build_ready"`
	GHAInstallStatus  string         `json:"gha_install_status,omitempty"`
	GHAOutOfSync      bool           `json:"gha_out_of_sync"`
	GHAPendingChanges int            `json:"gha_pending_changes"`
}

// DockerDeploymentResponse is a Docker application deployment attempt.
type DockerDeploymentResponse struct {
	ID            string     `json:"id"`
	TeamID        string     `json:"team_id"`
	ServerID      string     `json:"server_id"`
	TargetType    string     `json:"target_type"`
	TargetID      string     `json:"target_id"`
	Action        *string    `json:"action,omitempty"`
	Status        string     `json:"status"`
	TaskID        *string    `json:"task_id,omitempty"`
	CommitSHA     *string    `json:"commit_sha,omitempty"`
	CommitMsg     *string    `json:"commit_msg,omitempty"`
	ImageRef      *string    `json:"image_ref,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	Error         *string    `json:"error,omitempty"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
	TriggerSource string     `json:"trigger_source"`
	GHARunURL     *string    `json:"gha_run_url,omitempty"`
}
