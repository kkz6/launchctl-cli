package api

import (
	"fmt"
	"net/http"
)

func (c *Client) ListCertificates(serverID, siteID string) ([]CertificateResponse, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/sites/%s/certificates", serverID, siteID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]CertificateResponse](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}
