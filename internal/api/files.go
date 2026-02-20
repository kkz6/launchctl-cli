package api

import (
	"fmt"
	"net/http"
)

func (c *Client) ListFiles(serverID, siteID string) ([]FileOnServer, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/sites/%s/files", serverID, siteID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]FileOnServer](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *Client) GetFileContent(serverID, siteID, fileParam string) (*FileContent, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/sites/%s/files/%s", serverID, siteID, fileParam), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[FileContent](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) UpdateFileContent(serverID, siteID, fileParam string, req UpdateFileRequest) error {
	resp, err := c.doRequest(http.MethodPut, fmt.Sprintf("/api/servers/%s/sites/%s/files/%s", serverID, siteID, fileParam), req)
	if err != nil {
		return err
	}

	_, err = decodeResponse[FileContent](resp)
	return err
}
