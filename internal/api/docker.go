package api

import (
	"context"
	"fmt"
	"net/http"
)

func (c *Client) ListDockerProjects(serverID string) ([]DockerProjectResponse, error) {
	resp, err := c.doRequest(http.MethodGet, dockerProjectsPath(serverID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]DockerProjectResponse](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *Client) GetDockerProject(serverID, projectID string) (*DockerProjectResponse, error) {
	resp, err := c.doRequest(http.MethodGet, dockerProjectPath(serverID, projectID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[DockerProjectResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) CreateDockerProject(serverID string, req CreateDockerProjectRequest) (*DockerProjectResponse, error) {
	resp, err := c.doRequest(http.MethodPost, dockerProjectsPath(serverID), req)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[DockerProjectResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) UpdateDockerProject(serverID, projectID string, req UpdateDockerProjectRequest) (*DockerProjectResponse, error) {
	resp, err := c.doRequest(http.MethodPatch, dockerProjectPath(serverID, projectID), req)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[DockerProjectResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) DeleteDockerProject(serverID, projectID string) error {
	resp, err := c.doRequest(http.MethodDelete, dockerProjectPath(serverID, projectID), nil)
	if err != nil {
		return err
	}

	_, err = decodeResponse[struct{}](resp)
	return err
}

func (c *Client) ListDockerApplications(serverID, projectID string) ([]DockerApplicationResponse, error) {
	resp, err := c.doRequest(http.MethodGet, dockerApplicationsPath(serverID, projectID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]DockerApplicationResponse](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *Client) GetDockerApplication(serverID, projectID, applicationID string) (*DockerApplicationResponse, error) {
	return c.GetDockerApplicationContext(context.Background(), serverID, projectID, applicationID)
}

func (c *Client) GetDockerApplicationContext(ctx context.Context, serverID, projectID, applicationID string) (*DockerApplicationResponse, error) {
	resp, err := c.doRequestContext(ctx, http.MethodGet, dockerApplicationPath(serverID, projectID, applicationID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[DockerApplicationResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) CreateDockerApplication(serverID, projectID string, req CreateDockerApplicationRequest) (*DockerApplicationResponse, error) {
	resp, err := c.doRequest(http.MethodPost, dockerApplicationsPath(serverID, projectID), req)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[DockerApplicationResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) UpdateDockerApplication(serverID, projectID, applicationID string, req UpdateDockerApplicationRequest) (*DockerApplicationResponse, error) {
	resp, err := c.doRequest(http.MethodPatch, dockerApplicationPath(serverID, projectID, applicationID), req)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[DockerApplicationResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// DeleteDockerApplication preserves named volumes unless removeVolumes is true.
func (c *Client) DeleteDockerApplication(serverID, projectID, applicationID string, removeVolumes bool) error {
	path := dockerApplicationPath(serverID, projectID, applicationID)
	if removeVolumes {
		path += "?remove_volumes=true"
	}

	resp, err := c.doRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	_, err = decodeResponse[struct{}](resp)
	return err
}

func (c *Client) DeployDockerApplication(serverID, projectID, applicationID string) (*DockerDeploymentResponse, error) {
	path := dockerApplicationPath(serverID, projectID, applicationID) + "/deploy"
	resp, err := c.doRequest(http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[DockerDeploymentResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// DockerApplicationAction queues reload, start, or stop for an application.
func (c *Client) DockerApplicationAction(serverID, projectID, applicationID, action string) error {
	switch action {
	case "reload", "start", "stop":
	default:
		return fmt.Errorf("unsupported Docker application action %q", action)
	}

	path := dockerApplicationPath(serverID, projectID, applicationID) + "/" + action
	resp, err := c.doRequest(http.MethodPost, path, nil)
	if err != nil {
		return err
	}

	_, err = decodeResponse[struct{}](resp)
	return err
}

func (c *Client) ListDockerApplicationDeployments(serverID, projectID, applicationID string) ([]DockerDeploymentResponse, error) {
	return c.ListDockerApplicationDeploymentsContext(context.Background(), serverID, projectID, applicationID)
}

func (c *Client) ListDockerApplicationDeploymentsContext(ctx context.Context, serverID, projectID, applicationID string) ([]DockerDeploymentResponse, error) {
	path := dockerApplicationPath(serverID, projectID, applicationID) + "/deployments"
	resp, err := c.doRequestContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]DockerDeploymentResponse](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func dockerProjectsPath(serverID string) string {
	return fmt.Sprintf("/api/servers/%s/docker/projects", serverID)
}

func dockerProjectPath(serverID, projectID string) string {
	return fmt.Sprintf("%s/%s", dockerProjectsPath(serverID), projectID)
}

func dockerApplicationsPath(serverID, projectID string) string {
	return dockerProjectPath(serverID, projectID) + "/applications"
}

func dockerApplicationPath(serverID, projectID, applicationID string) string {
	return fmt.Sprintf("%s/%s", dockerApplicationsPath(serverID, projectID), applicationID)
}
