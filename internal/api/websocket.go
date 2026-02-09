package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/kkz6/launchctl/internal/config"
)

type WSClient struct {
	conn *websocket.Conn
	cfg  *config.Config
}

type WSMessage struct {
	Action  string `json:"action,omitempty"`
	Channel string `json:"channel,omitempty"`
	Event   string `json:"event,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type DeploymentLogEvent struct {
	DeploymentID string `json:"deployment_id"`
	SiteID       string `json:"site_id"`
	ServerID     string `json:"server_id"`
	Output       string `json:"output,omitempty"`
	Status       string `json:"status,omitempty"`
	Step         string `json:"step,omitempty"`
}

func NewWSClient(cfg *config.Config) (*WSClient, error) {
	baseURL := cfg.APIURL
	baseURL = strings.Replace(baseURL, "https://", "wss://", 1)
	baseURL = strings.Replace(baseURL, "http://", "ws://", 1)

	u, err := url.Parse(baseURL + "/api/ws")
	if err != nil {
		return nil, fmt.Errorf("failed to parse websocket URL: %w", err)
	}

	q := u.Query()
	q.Set("token", cfg.AccessToken)
	q.Set("team_id", cfg.TeamID)
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to websocket: %w", err)
	}

	return &WSClient{conn: conn, cfg: cfg}, nil
}

func (c *WSClient) Subscribe(channel string) error {
	msg := WSMessage{
		Action:  "subscribe",
		Channel: channel,
	}

	return c.conn.WriteJSON(msg)
}

func (c *WSClient) ReadMessage() (*WSMessage, error) {
	_, data, err := c.conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	var msg WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

func (c *WSClient) Close() error {
	return c.conn.Close()
}
