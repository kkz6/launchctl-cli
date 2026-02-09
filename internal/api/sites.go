package api

import (
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
	resp, err := c.doRequest(http.MethodPost, fmt.Sprintf("/api/servers/%s/sites/", serverID), req)
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
	resp, err := c.doRequest(http.MethodPost, fmt.Sprintf("/api/servers/%s/sites/%s/deploy", serverID, siteID), nil)
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
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/sites/%s/deployments/%s", serverID, siteID, deploymentID), nil)
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
	resp, err := c.doRequest(http.MethodPost, fmt.Sprintf("/api/servers/%s/sites/%s/rollback/%s", serverID, siteID, deploymentID), nil)
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
