package api

import (
	"context"
	"fmt"
	"net/http"
)

func (c *Client) ListSites(serverID string) ([]SiteResponse, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/sites/", serverID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]SiteResponse](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *Client) GetSite(serverID, siteID string) (*SiteResponse, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/sites/%s", serverID, siteID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[SiteResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) CreateSite(serverID string, req CreateSiteRequest) (*SiteResponse, error) {
	return c.CreateSiteContext(context.Background(), serverID, req)
}

func (c *Client) CreateSiteContext(ctx context.Context, serverID string, req CreateSiteRequest) (*SiteResponse, error) {
	resp, err := c.doRequestContext(ctx, http.MethodPost, fmt.Sprintf("/api/servers/%s/sites/", serverID), req)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[SiteResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) DeploySite(serverID, siteID string) (*DeploymentResponse, error) {
	return c.DeploySiteContext(context.Background(), serverID, siteID)
}

func (c *Client) DeploySiteContext(ctx context.Context, serverID, siteID string) (*DeploymentResponse, error) {
	resp, err := c.doRequestContext(ctx, http.MethodPost, fmt.Sprintf("/api/servers/%s/sites/%s/deploy", serverID, siteID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[DeploymentResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) ListDeployments(serverID, siteID string) ([]DeploymentResponse, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/sites/%s/deployments", serverID, siteID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]DeploymentResponse](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *Client) GetDeployment(serverID, siteID, deploymentID string) (*DeploymentResponse, error) {
	return c.GetDeploymentContext(context.Background(), serverID, siteID, deploymentID)
}

func (c *Client) GetDeploymentContext(ctx context.Context, serverID, siteID, deploymentID string) (*DeploymentResponse, error) {
	resp, err := c.doRequestContext(ctx, http.MethodGet, fmt.Sprintf("/api/servers/%s/sites/%s/deployments/%s", serverID, siteID, deploymentID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[DeploymentResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) RollbackDeployment(serverID, siteID, deploymentID string) (*DeploymentResponse, error) {
	return c.RollbackDeploymentContext(context.Background(), serverID, siteID, deploymentID)
}

func (c *Client) RollbackDeploymentContext(ctx context.Context, serverID, siteID, deploymentID string) (*DeploymentResponse, error) {
	resp, err := c.doRequestContext(ctx, http.MethodPost, fmt.Sprintf("/api/servers/%s/sites/%s/rollback/%s", serverID, siteID, deploymentID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[DeploymentResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) GetDashboard() (*DashboardResponse, error) {
	resp, err := c.doRequest(http.MethodGet, "/api/dashboard", nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[DashboardResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}
