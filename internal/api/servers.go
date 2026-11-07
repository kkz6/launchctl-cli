package api

import (
	"context"
	"fmt"
	"net/http"
)

func (c *Client) ListServers() ([]ServerResponse, *PaginationMeta, error) {
	resp, err := c.doRequest(http.MethodGet, "/api/servers/?per_page=100", nil)
	if err != nil {
		return nil, nil, err
	}

	result, err := decodeResponse[[]ServerResponse](resp)
	if err != nil {
		return nil, nil, err
	}

	return result.Data, result.Meta, nil
}

func (c *Client) GetServer(id string) (*ServerResponse, error) {
	return c.GetServerContext(context.Background(), id)
}

func (c *Client) GetServerContext(ctx context.Context, id string) (*ServerResponse, error) {
	resp, err := c.doRequestContext(ctx, http.MethodGet, fmt.Sprintf("/api/servers/%s", id), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[ServerResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) GetCreateServerOptions() (*CreateServerOptions, error) {
	return c.GetCreateServerOptionsContext(context.Background())
}

func (c *Client) GetCreateServerOptionsContext(ctx context.Context) (*CreateServerOptions, error) {
	resp, err := c.doRequestContext(ctx, http.MethodGet, "/api/servers/create-options", nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[CreateServerOptions](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) CreateServer(req CreateServerRequest) (*ServerResponse, error) {
	return c.CreateServerContext(context.Background(), req)
}

func (c *Client) CreateServerContext(ctx context.Context, req CreateServerRequest) (*ServerResponse, error) {
	resp, err := c.doRequestContext(ctx, http.MethodPost, "/api/servers/", req)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[ServerResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) RebootServer(id string) error {
	return c.RebootServerContext(context.Background(), id)
}

func (c *Client) GetTask(serverID, taskID string) (*TaskResponse, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/tasks/%s", serverID, taskID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[TaskResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) GetServerMetrics(id string) (*MetricResponse, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/metrics/latest", id), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[MetricResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}
