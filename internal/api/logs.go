package api

import (
	"fmt"
	"net/http"
)

func (c *Client) ListServerLogs(serverID string) ([]LogInfo, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/logs", serverID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]LogInfo](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *Client) GetServerLogContent(serverID, logParam string) (*FileContent, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/logs/%s", serverID, logParam), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[FileContent](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) ListSiteLogs(serverID, siteID string) ([]FileOnServer, error) {
	return c.ListFiles(serverID, siteID)
}
