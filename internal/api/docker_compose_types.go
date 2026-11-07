package api

import "time"

type DockerComposeResponse struct {
	ID                  string                            `json:"id"`
	TeamID              string                            `json:"team_id"`
	ServerID            string                            `json:"server_id"`
	ProjectID           string                            `json:"project_id"`
	Name                string                            `json:"name"`
	ComposeSourceType   string                            `json:"compose_source_type"`
	SourceConfig        map[string]any                    `json:"source_config,omitempty"`
	ComposeFilePath     *string                           `json:"compose_file_path,omitempty"`
	RawYAML             *string                           `json:"raw_yaml,omitempty"`
	EnvFile             *string                           `json:"env_file,omitempty"`
	RunCommand          *string                           `json:"run_command,omitempty"`
	Status              string                            `json:"status"`
	LastDeployedAt      *time.Time                        `json:"last_deployed_at,omitempty"`
	CreatedAt           *time.Time                        `json:"created_at,omitempty"`
	UpdatedAt           *time.Time                        `json:"updated_at,omitempty"`
	RegistryCredentials []DockerRegistryCredentialSummary `json:"registry_credentials,omitempty"`
	BuildLocation       string                            `json:"build_location"`
	GHABuildReady       bool                              `json:"gha_build_ready"`
	GHAInstallStatus    string                            `json:"gha_install_status,omitempty"`
}

type DockerRegistryCredentialSummary struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	RegistryURL *string `json:"registry_url,omitempty"`
}

// Compose creation currently supports inline YAML in the CLI. The API also
// supports Git sources; those can still be listed, deployed, and reloaded.
type CreateDockerComposeRequest struct {
	Name              string                `json:"name"`
	ComposeSourceType string                `json:"compose_source_type"`
	RawYAML           *DockerComposeRawYAML `json:"raw_yaml,omitempty"`
}

type DockerComposeRawYAML struct {
	Contents string `json:"contents"`
}

// Source and compose YAML cannot be changed by the backend's PATCH endpoint.
type UpdateDockerComposeRequest struct {
	Name       *string `json:"name,omitempty"`
	EnvFile    *string `json:"env_file,omitempty"`
	RunCommand *string `json:"run_command,omitempty"`
}
