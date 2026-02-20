package api

import (
	"fmt"
	"net/http"
)

func (c *Client) ListSSHKeys() ([]SSHKeyResponse, error) {
	resp, err := c.doRequest(http.MethodGet, "/api/ssh-keys", nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]SSHKeyResponse](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *Client) CreateSSHKey(req CreateSSHKeyRequest) (*SSHKeyResponse, error) {
	resp, err := c.doRequest(http.MethodPost, "/api/ssh-keys", req)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[SSHKeyResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) DeleteSSHKey(keyID string) error {
	resp, err := c.doRequest(http.MethodDelete, fmt.Sprintf("/api/ssh-keys/%s", keyID), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

func (c *Client) ListServerSSHKeys(serverID string) ([]SSHKeyResponse, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/ssh-keys", serverID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]SSHKeyResponse](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *Client) AttachSSHKey(serverID string, req AttachSSHKeyRequest) error {
	resp, err := c.doRequest(http.MethodPost, fmt.Sprintf("/api/servers/%s/ssh-keys", serverID), req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

func (c *Client) DetachSSHKey(serverID, keyID string) error {
	resp, err := c.doRequest(http.MethodDelete, fmt.Sprintf("/api/servers/%s/ssh-keys/%s", serverID, keyID), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}
