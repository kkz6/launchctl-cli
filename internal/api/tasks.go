package api

import (
	"fmt"
	"net/http"
)

func (c *Client) ListTasks(serverID string) ([]TaskResponse, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/tasks", serverID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]TaskResponse](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *Client) GetLatestTask(serverID string) (*TaskResponse, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/tasks/latest", serverID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[TaskResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}
