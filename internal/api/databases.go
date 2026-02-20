package api

import (
	"fmt"
	"net/http"
)

func (c *Client) ListDatabases(serverID string) ([]DatabaseResponse, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/databases", serverID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]DatabaseResponse](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *Client) GetDatabase(serverID, dbID string) (*DatabaseResponse, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/databases/%s", serverID, dbID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[DatabaseResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) CreateDatabase(serverID string, req CreateDatabaseRequest) (*DatabaseResponse, error) {
	resp, err := c.doRequest(http.MethodPost, fmt.Sprintf("/api/servers/%s/databases", serverID), req)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[DatabaseResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) DeleteDatabase(serverID, dbID string) error {
	resp, err := c.doRequest(http.MethodDelete, fmt.Sprintf("/api/servers/%s/databases/%s", serverID, dbID), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

func (c *Client) ListDatabaseUsers(serverID string) ([]DatabaseUserResponse, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/database-users", serverID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]DatabaseUserResponse](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}
