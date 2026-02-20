package api

import (
	"fmt"
	"net/http"
)

func (c *Client) ListDaemons(serverID string) ([]DaemonResponse, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/daemons", serverID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]DaemonResponse](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *Client) CreateDaemon(serverID string, req CreateDaemonRequest) (*DaemonResponse, error) {
	resp, err := c.doRequest(http.MethodPost, fmt.Sprintf("/api/servers/%s/daemons", serverID), req)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[DaemonResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) RestartDaemon(serverID, daemonID string) error {
	resp, err := c.doRequest(http.MethodPost, fmt.Sprintf("/api/servers/%s/daemons/%s/restart", serverID, daemonID), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

func (c *Client) DeleteDaemon(serverID, daemonID string) error {
	resp, err := c.doRequest(http.MethodDelete, fmt.Sprintf("/api/servers/%s/daemons/%s", serverID, daemonID), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}
