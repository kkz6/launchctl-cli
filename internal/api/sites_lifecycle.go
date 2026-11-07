package api

import (
	"context"
	"fmt"
	"net/http"
)

// UpdateSiteRequest includes only fields explicitly changed by the caller.
type UpdateSiteRequest struct {
	PHPVersion                  *string `json:"php_version,omitempty"`
	WebFolder                   *string `json:"web_folder,omitempty"`
	RepositoryBranch            *string `json:"repository_branch,omitempty"`
	DeploymentReleasesRetention *int    `json:"deployment_releases_retention,omitempty"`
	QueueDeployments            *bool   `json:"queue_deployments,omitempty"`
}

func (c *Client) UpdateSite(ctx context.Context, serverID, siteID string, req UpdateSiteRequest) (*SiteResponse, error) {
	resp, err := c.doRequestContext(ctx, http.MethodPut, fmt.Sprintf("/api/servers/%s/sites/%s", serverID, siteID), req)
	if err != nil {
		return nil, err
	}
	result, err := decodeResponse[SiteResponse](resp)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// DeleteSite requests asynchronous site deletion.
func (c *Client) DeleteSite(ctx context.Context, serverID, siteID string) error {
	resp, err := c.doRequestContext(ctx, http.MethodDelete, fmt.Sprintf("/api/servers/%s/sites/%s", serverID, siteID), nil)
	if err != nil {
		return err
	}
	_, err = decodeResponse[any](resp)
	return err
}

func (c *Client) SetSiteAutoDeployment(ctx context.Context, serverID, siteID string, enabled bool) error {
	action := "disable"
	if enabled {
		action = "enable"
	}
	resp, err := c.doRequestContext(ctx, http.MethodPost, fmt.Sprintf("/api/servers/%s/sites/%s/auto-deployment/%s", serverID, siteID, action), nil)
	if err != nil {
		return err
	}
	_, err = decodeResponse[any](resp)
	return err
}
