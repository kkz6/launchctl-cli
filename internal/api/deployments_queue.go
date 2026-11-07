package api

import (
	"context"
	"fmt"
	"net/http"
)

type CancelQueuedDeploymentsResponse struct {
	Cancelled int64 `json:"cancelled"`
}

func (c *Client) CancelQueuedDeployments(ctx context.Context, serverID, siteID string) (*CancelQueuedDeploymentsResponse, error) {
	resp, err := c.doRequestContext(ctx, http.MethodDelete, fmt.Sprintf("/api/servers/%s/sites/%s/deployments/queued", serverID, siteID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[CancelQueuedDeploymentsResponse](resp)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}
