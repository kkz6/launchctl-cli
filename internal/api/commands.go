package api

import (
	"fmt"
	"net/http"
)

func (c *Client) CreateCommand(serverID, siteID string, req CreateCommandRequest) (*CommandResponse, error) {
	resp, err := c.doRequest(http.MethodPost, fmt.Sprintf("/api/servers/%s/sites/%s/commands", serverID, siteID), req)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[CommandResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) ListCommands(serverID, siteID string) ([]CommandResponse, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/sites/%s/commands", serverID, siteID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]CommandResponse](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *Client) DeleteCommand(serverID, siteID, commandID string) error {
	resp, err := c.doRequest(http.MethodDelete, fmt.Sprintf("/api/servers/%s/sites/%s/commands/%s", serverID, siteID, commandID), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}
