package api

import (
	"fmt"
	"net/http"
)

func (c *Client) ListFirewallRules(serverID string) ([]FirewallRuleResponse, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/api/servers/%s/firewall-rules", serverID), nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]FirewallRuleResponse](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *Client) CreateFirewallRule(serverID string, req CreateFirewallRuleRequest) (*FirewallRuleResponse, error) {
	resp, err := c.doRequest(http.MethodPost, fmt.Sprintf("/api/servers/%s/firewall-rules", serverID), req)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[FirewallRuleResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) DeleteFirewallRule(serverID, ruleID string) error {
	resp, err := c.doRequest(http.MethodDelete, fmt.Sprintf("/api/servers/%s/firewall-rules/%s", serverID, ruleID), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}
