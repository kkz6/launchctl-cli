package api

import (
	"fmt"
	"net/http"
)

func (c *Client) ListCrons(serverID string) ([]CronResponse, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/crons", serverID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]CronResponse](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *Client) CreateCron(serverID string, req CreateCronRequest) (*CronResponse, error) {
	resp, err := c.doRequest(http.MethodPost, fmt.Sprintf("/api/servers/%s/crons", serverID), req)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[CronResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) DeleteCron(serverID, cronID string) error {
	resp, err := c.doRequest(http.MethodDelete, fmt.Sprintf("/api/servers/%s/crons/%s", serverID, cronID), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}
