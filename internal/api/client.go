package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kkz6/launchctl/internal/config"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	cfg        *config.Config
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		baseURL: cfg.APIURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cfg: cfg,
	}
}

func (c *Client) doRequest(method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.cfg.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.AccessToken)
	}
	if c.cfg.TeamID != "" {
		req.Header.Set("X-Team-ID", c.cfg.TeamID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized && c.cfg.RefreshToken != "" {
		resp.Body.Close()

		if err := c.refreshToken(); err != nil {
			return nil, fmt.Errorf("session expired, please login again")
		}

		req.Header.Set("Authorization", "Bearer "+c.cfg.AccessToken)
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed after token refresh: %w", err)
		}
	}

	return resp, nil
}

func (c *Client) refreshToken() error {
	body := RefreshTokenRequest{
		RefreshToken: c.cfg.RefreshToken,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	url := c.baseURL + "/api/auth/refresh"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refresh failed with status %d", resp.StatusCode)
	}

	var result APIResponse[AuthResponse]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("refresh failed: %s", result.Message)
	}

	c.cfg.AccessToken = result.Data.AccessToken
	if result.Data.RefreshToken != "" {
		c.cfg.RefreshToken = result.Data.RefreshToken
	}

	return c.cfg.Save()
}

func decodeResponse[T any](resp *http.Response) (*APIResponse[T], error) {
	defer resp.Body.Close()

	var result APIResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		if len(result.Errors) > 0 {
			return &result, fmt.Errorf("%s", formatErrors(result.Errors))
		}
		return &result, fmt.Errorf("%s", result.Message)
	}

	return &result, nil
}

func UnmarshalEventData(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func formatErrors(errors map[string][]string) string {
	var msg string
	for field, errs := range errors {
		for _, e := range errs {
			if msg != "" {
				msg += "\n"
			}
			msg += fmt.Sprintf("%s: %s", field, e)
		}
	}
	return msg
}
