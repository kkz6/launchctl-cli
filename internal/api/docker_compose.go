package api

import (
	"context"
	"net/http"
	"net/url"
)

func (c *Client) ListDockerComposesContext(ctx context.Context, serverID, projectID string) ([]DockerComposeResponse, error) {
	resp, err := c.doRequestContext(ctx, http.MethodGet, dockerComposesPath(serverID, projectID), nil)
	if err != nil {
		return nil, err
	}
	result, err := decodeResponse[[]DockerComposeResponse](resp)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (c *Client) GetDockerComposeContext(ctx context.Context, serverID, projectID, composeID string) (*DockerComposeResponse, error) {
	resp, err := c.doRequestContext(ctx, http.MethodGet, dockerComposePath(serverID, projectID, composeID), nil)
	if err != nil {
		return nil, err
	}
	result, err := decodeResponse[DockerComposeResponse](resp)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) CreateDockerComposeContext(ctx context.Context, serverID, projectID string, req CreateDockerComposeRequest) (*DockerComposeResponse, error) {
	resp, err := c.doRequestContext(ctx, http.MethodPost, dockerComposesPath(serverID, projectID), req)
	if err != nil {
		return nil, err
	}
	result, err := decodeResponse[DockerComposeResponse](resp)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) UpdateDockerComposeContext(ctx context.Context, serverID, projectID, composeID string, req UpdateDockerComposeRequest) (*DockerComposeResponse, error) {
	resp, err := c.doRequestContext(ctx, http.MethodPatch, dockerComposePath(serverID, projectID, composeID), req)
	if err != nil {
		return nil, err
	}
	result, err := decodeResponse[DockerComposeResponse](resp)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// DeleteDockerComposeContext queues deletion, preserving named volumes by default.
func (c *Client) DeleteDockerComposeContext(ctx context.Context, serverID, projectID, composeID string, removeVolumes bool) error {
	path := dockerComposePath(serverID, projectID, composeID)
	if removeVolumes {
		path += "?remove_volumes=true"
	}
	resp, err := c.doRequestContext(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	_, err = decodeResponse[struct{}](resp)
	return err
}

func (c *Client) DeployDockerComposeContext(ctx context.Context, serverID, projectID, composeID string) (*DockerDeploymentResponse, error) {
	resp, err := c.doRequestContext(ctx, http.MethodPost, dockerComposePath(serverID, projectID, composeID)+"/deploy", nil)
	if err != nil {
		return nil, err
	}
	result, err := decodeResponse[DockerDeploymentResponse](resp)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// ReloadDockerComposeContext queues recreation; the API returns no deployment row.
func (c *Client) ReloadDockerComposeContext(ctx context.Context, serverID, projectID, composeID string) error {
	resp, err := c.doRequestContext(ctx, http.MethodPost, dockerComposePath(serverID, projectID, composeID)+"/reload", nil)
	if err != nil {
		return err
	}
	_, err = decodeResponse[struct{}](resp)
	return err
}

func (c *Client) ListDockerComposeDeploymentsContext(ctx context.Context, serverID, projectID, composeID string) ([]DockerDeploymentResponse, error) {
	resp, err := c.doRequestContext(ctx, http.MethodGet, dockerComposePath(serverID, projectID, composeID)+"/deployments", nil)
	if err != nil {
		return nil, err
	}
	result, err := decodeResponse[[]DockerDeploymentResponse](resp)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

func dockerComposesPath(serverID, projectID string) string {
	return dockerProjectPath(url.PathEscape(serverID), url.PathEscape(projectID)) + "/composes"
}

func dockerComposePath(serverID, projectID, composeID string) string {
	return dockerComposesPath(serverID, projectID) + "/" + url.PathEscape(composeID)
}
