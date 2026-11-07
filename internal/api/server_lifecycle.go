package api

import (
	"context"
	"fmt"
	"net/http"
)

type UpdateServerRequest struct {
	Name              *string `json:"name,omitempty"`
	Description       *string `json:"description,omitempty"`
	MonitoringEnabled *bool   `json:"monitoring_enabled,omitempty"`
	AutoUpdate        *bool   `json:"auto_update,omitempty"`
	SSHPort           *int    `json:"ssh_port,omitempty"`
}

func (c *Client) UpdateServer(ctx context.Context, id string, req UpdateServerRequest) (*ServerResponse, error) {
	resp, err := c.doRequestContext(ctx, http.MethodPatch, fmt.Sprintf("/api/servers/%s", id), req)
	if err != nil {
		return nil, err
	}
	result, err := decodeResponse[ServerResponse](resp)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) RetryServerProvision(ctx context.Context, id string) error {
	return c.serverAction(ctx, id, "retry-provision")
}

func (c *Client) TryServerConnection(ctx context.Context, id string) error {
	return c.serverAction(ctx, id, "try-connection")
}

func (c *Client) ArchiveServer(ctx context.Context, id string) error {
	return c.serverAction(ctx, id, "archive")
}

func (c *Client) UnarchiveServer(ctx context.Context, id string) error {
	return c.serverAction(ctx, id, "unarchive")
}

func (c *Client) RebootServerContext(ctx context.Context, id string) error {
	return c.serverAction(ctx, id, "reboot")
}

func (c *Client) serverAction(ctx context.Context, id, action string) error {
	resp, err := c.doRequestContext(ctx, http.MethodPost, fmt.Sprintf("/api/servers/%s/%s", id, action), nil)
	if err != nil {
		return err
	}
	_, err = decodeResponse[any](resp)
	return err
}
