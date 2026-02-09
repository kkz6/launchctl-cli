package api

import (
	"net/http"
)

func (c *Client) Login(email, password string) (*AuthResponse, error) {
	body := LoginRequest{
		Email:    email,
		Password: password,
		Remember: true,
	}

	resp, err := c.doRequest(http.MethodPost, "/api/auth/login", body)
	if err != nil {
		return nil, err
	}

	result, err := decodeResponse[AuthResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
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
