package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kkz6/launchctl/internal/config"
)

const (
	defaultRequestTimeout = 45 * time.Second
	maxErrorBodyBytes     = 1 << 20
	maxResponseBodyBytes  = 32 << 20
	maxGETAttempts        = 3
)

// APIError describes a non-successful HTTP response from the launchctl API.
// Callers can use errors.As to inspect the status code without parsing text.
type APIError struct {
	StatusCode int
	Message    string
	Errors     map[string][]string
	RequestID  string
}

func (e *APIError) Error() string {
	message := strings.TrimSpace(e.Message)
	if len(e.Errors) > 0 {
		message = formatErrors(e.Errors)
	}
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}
	if e.RequestID != "" {
		return fmt.Sprintf("API request failed (%d, request %s): %s", e.StatusCode, e.RequestID, message)
	}
	return fmt.Sprintf("API request failed (%d): %s", e.StatusCode, message)
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	cfg        *config.Config
}

func NewClient(cfg *config.Config) *Client {
	return NewClientWithHTTPClient(cfg, &http.Client{Timeout: defaultRequestTimeout})
}

// NewClientWithHTTPClient allows tests and embedding applications to supply
// their own transport while retaining launchctl authentication behavior.
func NewClientWithHTTPClient(cfg *config.Config, httpClient *http.Client) *Client {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultRequestTimeout}
	}
	return &Client{
		baseURL:    strings.TrimRight(cfg.EffectiveAPIURL(), "/"),
		httpClient: httpClient,
		cfg:        cfg,
	}
}

func (c *Client) BaseURL() string { return c.baseURL }

func (c *Client) doRequest(method, path string, body any) (*http.Response, error) {
	return c.doRequestContext(context.Background(), method, path, body)
}

func (c *Client) doRequestContext(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	attempts := 1
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		attempts = maxGETAttempts
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		var bodyReader io.Reader
		if payload != nil {
			bodyReader = bytes.NewReader(payload)
		}

		requestURL, err := resolveRequestURL(c.baseURL, path)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, method, requestURL, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "lctl")
		if c.cfg.AccessToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.cfg.AccessToken)
		}
		if c.cfg.TeamID != "" {
			req.Header.Set("X-Team-ID", c.cfg.TeamID)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if attempt < attempts && ctx.Err() == nil {
				if err := waitForRetry(ctx, time.Duration(attempt)*200*time.Millisecond); err != nil {
					return nil, err
				}
				continue
			}
			return nil, lastErr
		}

		if attempt < attempts && isRetryableStatus(resp.StatusCode) {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrorBodyBytes))
			_ = resp.Body.Close()
			if err := waitForRetry(ctx, retryDelay(resp, attempt)); err != nil {
				return nil, err
			}
			continue
		}

		return resp, nil
	}

	return nil, lastErr
}

// RawRequest calls any authenticated API endpoint and returns its response
// body. It powers `lctl api` so newly shipped endpoints are usable before a
// dedicated high-level command is added.
func (c *Client) RawRequest(ctx context.Context, method, path string, body any) ([]byte, int, error) {
	resp, err := c.doRequestContext(ctx, strings.ToUpper(method), path, body)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes+1))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}
	if len(data) > maxResponseBodyBytes {
		return nil, resp.StatusCode, fmt.Errorf("response exceeds %d MiB limit", maxResponseBodyBytes>>20)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, resp.StatusCode, apiErrorFromBody(resp, data)
	}
	return data, resp.StatusCode, nil
}

func decodeResponse[T any](resp *http.Response) (*APIResponse[T], error) {
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if len(data) > maxResponseBodyBytes {
		return nil, fmt.Errorf("response exceeds %d MiB limit", maxResponseBodyBytes>>20)
	}

	var result APIResponse[T]
	if len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &result); err != nil {
			if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
				return nil, apiErrorFromBody(resp, data)
			}
			return nil, fmt.Errorf("failed to decode API response (status %d): %w", resp.StatusCode, err)
		}
	}

	if resp.StatusCode == http.StatusNoContent {
		result.Success = true
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices || !result.Success {
		return &result, &APIError{
			StatusCode: resp.StatusCode,
			Message:    result.Message,
			Errors:     result.Errors,
			RequestID:  responseRequestID(resp),
		}
	}

	return &result, nil
}

func (c *Client) ExchangeToken() (string, error) {
	resp, err := c.doRequest(http.MethodPost, "/api/auth/token", nil)
	if err != nil {
		return "", fmt.Errorf("failed to exchange token: %w", err)
	}

	result, err := decodeResponse[AuthResponse](resp)
	if err != nil {
		return "", fmt.Errorf("failed to exchange token: %w", err)
	}

	return result.Data.AccessToken, nil
}

func UnmarshalEventData(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func resolveRequestURL(baseURL, requestPath string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("invalid API URL %q", baseURL)
	}
	if base.Scheme != "http" && base.Scheme != "https" {
		return "", fmt.Errorf("unsupported API URL scheme %q", base.Scheme)
	}

	requested, err := url.Parse(requestPath)
	if err != nil {
		return "", fmt.Errorf("invalid request path %q: %w", requestPath, err)
	}
	path := "/" + strings.TrimLeft(requested.Path, "/")
	// lctl historically exposed /api-prefixed paths while the Go API now
	// serves routes from its host root. Keep accepting the public CLI path
	// format and translate it at the transport boundary. A custom base URL
	// ending in /api retains that prefix for legacy development deployments.
	if path == "/api" {
		path = "/"
	} else if strings.HasPrefix(path, "/api/") {
		path = strings.TrimPrefix(path, "/api")
	}
	basePath := strings.TrimRight(base.Path, "/")
	base.Path = basePath + path
	base.RawQuery = requested.RawQuery
	base.Fragment = ""
	return base.String(), nil
}

func apiErrorFromBody(resp *http.Response, data []byte) error {
	var envelope struct {
		Message string              `json:"message"`
		Errors  map[string][]string `json:"errors"`
	}
	_ = json.Unmarshal(data, &envelope)
	if envelope.Message == "" {
		envelope.Message = strings.TrimSpace(string(data))
	}
	if len(envelope.Message) > 500 {
		envelope.Message = envelope.Message[:500] + "…"
	}
	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    envelope.Message,
		Errors:     envelope.Errors,
		RequestID:  responseRequestID(resp),
	}
}

func responseRequestID(resp *http.Response) string {
	for _, key := range []string{"X-Request-ID", "X-Trace-ID", "Traceparent"} {
		if value := resp.Header.Get(key); value != "" {
			return value
		}
	}
	return ""
}

func isRetryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout
}

func retryDelay(resp *http.Response, attempt int) time.Duration {
	if value := resp.Header.Get("Retry-After"); value != "" {
		if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 && seconds <= 30 {
			return time.Duration(seconds) * time.Second
		}
	}
	return time.Duration(attempt) * 250 * time.Millisecond
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func formatErrors(apiErrors map[string][]string) string {
	var messages []string
	fields := make([]string, 0, len(apiErrors))
	for field := range apiErrors {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	for _, field := range fields {
		fieldErrors := apiErrors[field]
		for _, message := range fieldErrors {
			messages = append(messages, fmt.Sprintf("%s: %s", field, message))
		}
	}
	return strings.Join(messages, "\n")
}

var _ error = (*APIError)(nil)
