package api

import (
	"context"
	"net/http"
)

type DeviceAuthorization struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type DeviceAuthorizationResult struct {
	Status      string        `json:"status"`
	AccessToken string        `json:"access_token,omitempty"`
	User        *UserResponse `json:"user,omitempty"`
}

func (c *Client) StartDeviceAuthorization(ctx context.Context) (*DeviceAuthorization, error) {
	resp, err := c.doRequestContext(ctx, http.MethodPost, "/api/auth/device/code", nil)
	if err != nil {
		return nil, err
	}
	result, err := decodeResponse[DeviceAuthorization](resp)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) PollDeviceAuthorization(ctx context.Context, deviceCode string) (*DeviceAuthorizationResult, error) {
	resp, err := c.doRequestContext(ctx, http.MethodPost, "/api/auth/device/token", map[string]string{"device_code": deviceCode})
	if err != nil {
		return nil, err
	}
	result, err := decodeResponse[DeviceAuthorizationResult](resp)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}
