package api

import (
	"net/http"
)

func (c *Client) ValidateToken(token string) (*UserResponse, error) {
	original := c.cfg.AccessToken
	c.cfg.AccessToken = token

	resp, err := c.doRequest(http.MethodGet, "/api/auth/user", nil)
	if err != nil {
		c.cfg.AccessToken = original
		return nil, err
	}

	result, err := decodeResponse[UserResponse](resp)
	if err != nil {
		c.cfg.AccessToken = original
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) VerifyTwoFactor(code string) error {
	body := TwoFactorChallengeRequest{
		Code: code,
	}

	resp, err := c.doRequest(http.MethodPost, "/api/auth/two-factor/challenge", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) Logout() error {
	resp, err := c.doRequest(http.MethodPost, "/api/auth/logout", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) GetUser() (*UserResponse, error) {
	resp, err := c.doRequest(http.MethodGet, "/api/auth/user", nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[UserResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) ListTeams() ([]TeamResponse, error) {
	resp, err := c.doRequest(http.MethodGet, "/api/teams/", nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]TeamResponse](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *Client) SwitchTeam(teamID string) error {
	body := SwitchTeamRequest{TeamID: teamID}

	resp, err := c.doRequest(http.MethodPut, "/api/auth/current-team", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) GetTeamMembers(teamID string) ([]TeamMemberResponse, error) {
	resp, err := c.doRequest(http.MethodGet, "/api/teams/"+teamID+"/members", nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[[]TeamMemberResponse](resp)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}
