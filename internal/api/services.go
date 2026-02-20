package api

import (
	"fmt"
	"net/http"
)

func (c *Client) ListServices(serverID string) ([]ServiceResponse, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/services", serverID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]ServiceResponse](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *Client) ServiceOperation(serverID, serviceID string, req ServiceOperationRequest) error {
	resp, err := c.doRequest(http.MethodPost, fmt.Sprintf("/api/servers/%s/services/%s", serverID, serviceID), req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}
