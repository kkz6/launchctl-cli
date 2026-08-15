package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kkz6/launchctl/internal/config"
)

const (
	defaultPingInterval = 25 * time.Second
	defaultPongWait     = 40 * time.Second
	defaultWriteWait    = 10 * time.Second
	defaultMinBackoff   = time.Second
	defaultMaxBackoff   = 30 * time.Second
)

var channelPattern = regexp.MustCompile(`^(team|server|site|deployment|task|user)\.[A-Za-z0-9_-]+$`)

type ConnectionState string

const (
	StateConnected    ConnectionState = "connected"
	StateReconnecting ConnectionState = "reconnecting"
	StateDisconnected ConnectionState = "disconnected"
	StateClosed       ConnectionState = "closed"
)

type WSState struct {
	State   ConnectionState
	Attempt int
	Err     error
}

type WSMessage struct {
	Action  string          `json:"action,omitempty"`
	Channel string          `json:"channel,omitempty"`
	Event   string          `json:"event,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type DeploymentLogEvent struct {
	DeploymentID string `json:"deployment_id"`
	SiteID       string `json:"site_id"`
	ServerID     string `json:"server_id"`
	Output       string `json:"output,omitempty"`
	Message      string `json:"message,omitempty"`
	Status       string `json:"status,omitempty"`
	Step         string `json:"step,omitempty"`
	Progress     int    `json:"progress,omitempty"`
}

// WSManager owns a single pub/sub connection and keeps it alive across API
// restarts, laptop sleep, and transient network failures. Subscriptions are
// retained and replayed after every reconnect.
type WSManager struct {
	cfg    *config.Config
	token  string
	dialer *websocket.Dialer

	events chan *WSMessage
	states chan WSState
	done   chan struct{}

	mu            sync.RWMutex
	writeMu       sync.Mutex
	conn          *websocket.Conn
	subscriptions map[string]struct{}
	ctx           context.Context
	cancel        context.CancelFunc
	started       bool
	closed        bool
	wg            sync.WaitGroup

	pingInterval time.Duration
	pongWait     time.Duration
	writeWait    time.Duration
	minBackoff   time.Duration
	maxBackoff   time.Duration
}

func NewWSManager(cfg *config.Config, token string) *WSManager {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return &WSManager{
		cfg:           cfg,
		token:         token,
		dialer:        websocket.DefaultDialer,
		events:        make(chan *WSMessage, 256),
		states:        make(chan WSState, 32),
		done:          make(chan struct{}),
		subscriptions: make(map[string]struct{}),
		pingInterval:  defaultPingInterval,
		pongWait:      defaultPongWait,
		writeWait:     defaultWriteWait,
		minBackoff:    defaultMinBackoff,
		maxBackoff:    defaultMaxBackoff,
	}
}

func (m *WSManager) Connect(ctx context.Context) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return errors.New("websocket manager is closed")
	}
	if m.started {
		m.mu.Unlock()
		return nil
	}
	m.ctx, m.cancel = context.WithCancel(ctx)
	m.started = true
	m.mu.Unlock()

	conn, err := m.dial(m.ctx)
	if err != nil {
		m.mu.Lock()
		m.started = false
		m.cancel()
		m.cancel = nil
		m.mu.Unlock()
		return err
	}

	m.setConnection(conn)
	if err := m.replaySubscriptions(conn); err != nil {
		m.clearConnection(conn)
		_ = conn.Close()
		m.mu.Lock()
		m.started = false
		m.cancel()
		m.cancel = nil
		m.mu.Unlock()
		return fmt.Errorf("failed to restore websocket subscriptions: %w", err)
	}
	m.emitState(WSState{State: StateConnected})
	m.wg.Add(1)
	go m.run(conn)
	return nil
}

func (m *WSManager) Events() <-chan *WSMessage { return m.events }

func (m *WSManager) States() <-chan WSState { return m.states }

func (m *WSManager) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.conn != nil && !m.closed
}

func (m *WSManager) Subscribe(channel string) error {
	channel = strings.TrimSpace(channel)
	if !channelPattern.MatchString(channel) {
		return fmt.Errorf("invalid websocket channel %q (expected resource.id)", channel)
	}

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return errors.New("websocket manager is closed")
	}
	m.subscriptions[channel] = struct{}{}
	conn := m.conn
	m.mu.Unlock()

	if conn != nil {
		return m.writeJSON(conn, WSMessage{Action: "subscribe", Channel: channel})
	}
	return nil
}

func (m *WSManager) Unsubscribe(channel string) error {
	m.mu.Lock()
	delete(m.subscriptions, channel)
	conn := m.conn
	m.mu.Unlock()
	if conn != nil {
		return m.writeJSON(conn, WSMessage{Action: "unsubscribe", Channel: channel})
	}
	return nil
}

func (m *WSManager) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	if m.cancel != nil {
		m.cancel()
	}
	conn := m.conn
	m.conn = nil
	close(m.done)
	m.mu.Unlock()

	if conn != nil {
		m.writeMu.Lock()
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(m.writeWait))
		m.writeMu.Unlock()
		_ = conn.Close()
	}
	m.wg.Wait()
	m.emitState(WSState{State: StateClosed})
	return nil
}

func (m *WSManager) run(conn *websocket.Conn) {
	defer m.wg.Done()
	attempt := 0

	for {
		err := m.readConnection(conn)
		m.clearConnection(conn)
		_ = conn.Close()

		if m.isDone() {
			return
		}

		attempt++
		m.emitState(WSState{State: StateReconnecting, Attempt: attempt, Err: err})
		if !m.waitBackoff(attempt) {
			return
		}

		for {
			conn, err = m.dial(m.context())
			if err == nil {
				break
			}
			if m.isDone() {
				return
			}
			attempt++
			m.emitState(WSState{State: StateReconnecting, Attempt: attempt, Err: err})
			if !m.waitBackoff(attempt) {
				return
			}
		}

		m.setConnection(conn)
		if err := m.replaySubscriptions(conn); err != nil {
			_ = conn.Close()
			continue
		}
		attempt = 0
		m.emitState(WSState{State: StateConnected})
	}
}

func (m *WSManager) readConnection(conn *websocket.Conn) error {
	_ = conn.SetReadDeadline(time.Now().Add(m.pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(m.pongWait))
	})

	heartbeatDone := make(chan struct{})
	go m.heartbeat(conn, heartbeatDone)
	defer close(heartbeatDone)

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		_ = conn.SetReadDeadline(time.Now().Add(m.pongWait))

		var message WSMessage
		if err := json.Unmarshal(data, &message); err != nil {
			continue
		}
		select {
		case m.events <- &message:
		case <-m.done:
			return context.Canceled
		}
	}
}

func (m *WSManager) heartbeat(conn *websocket.Conn, done <-chan struct{}) {
	ticker := time.NewTicker(m.pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-m.done:
			return
		case <-ticker.C:
			m.writeMu.Lock()
			err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(m.writeWait))
			m.writeMu.Unlock()
			if err != nil {
				_ = conn.Close()
				return
			}
		}
	}
}

func (m *WSManager) dial(ctx context.Context) (*websocket.Conn, error) {
	query := url.Values{}
	query.Set("token", m.token)
	query.Set("team_id", m.cfg.TeamID)
	wsURL, err := buildWebSocketURL(m.cfg.EffectiveAPIURL(), "/api/ws", query)
	if err != nil {
		return nil, err
	}
	conn, resp, err := m.dialer.DialContext(ctx, wsURL, http.Header{"User-Agent": []string{"lctl"}})
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("websocket connection failed (status %d): %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("websocket connection failed: %w", err)
	}
	return conn, nil
}

func (m *WSManager) replaySubscriptions(conn *websocket.Conn) error {
	m.mu.RLock()
	channels := make([]string, 0, len(m.subscriptions))
	for channel := range m.subscriptions {
		channels = append(channels, channel)
	}
	m.mu.RUnlock()
	for _, channel := range channels {
		if err := m.writeJSON(conn, WSMessage{Action: "subscribe", Channel: channel}); err != nil {
			return err
		}
	}
	return nil
}

func (m *WSManager) writeJSON(conn *websocket.Conn, value any) error {
	m.writeMu.Lock()
	defer m.writeMu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(m.writeWait))
	return conn.WriteJSON(value)
}

func (m *WSManager) setConnection(conn *websocket.Conn) {
	m.mu.Lock()
	m.conn = conn
	m.mu.Unlock()
}

func (m *WSManager) clearConnection(conn *websocket.Conn) {
	m.mu.Lock()
	if m.conn == conn {
		m.conn = nil
	}
	m.mu.Unlock()
}

func (m *WSManager) emitState(state WSState) {
	select {
	case m.states <- state:
	default:
	}
}

func (m *WSManager) context() context.Context {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ctx
}

func (m *WSManager) isDone() bool {
	select {
	case <-m.done:
		return true
	default:
		return false
	}
}

func (m *WSManager) waitBackoff(attempt int) bool {
	delay := m.minBackoff << min(attempt-1, 5)
	if delay > m.maxBackoff {
		delay = m.maxBackoff
	}
	if jitter := delay / 5; jitter > 0 {
		delay += time.Duration(rand.Int63n(int64(jitter)))
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-m.done:
		return false
	case <-m.context().Done():
		return false
	case <-timer.C:
		return true
	}
}

// WSClient is the compatibility facade used by existing Bubble Tea models.
// Its ReadMessage call survives reconnects because the manager owns the wire.
type WSClient struct{ manager *WSManager }

func NewWSClient(cfg *config.Config, token string) (*WSClient, error) {
	manager := NewWSManager(cfg, token)
	if err := manager.Connect(context.Background()); err != nil {
		return nil, err
	}
	return &WSClient{manager: manager}, nil
}

func (c *WSClient) Subscribe(channel string) error { return c.manager.Subscribe(channel) }

func (c *WSClient) Unsubscribe(channel string) error { return c.manager.Unsubscribe(channel) }

func (c *WSClient) ReadMessage() (*WSMessage, error) {
	select {
	case message := <-c.manager.Events():
		return message, nil
	case <-c.manager.done:
		return nil, errors.New("websocket closed")
	}
}

func (c *WSClient) States() <-chan WSState { return c.manager.States() }

func (c *WSClient) Events() <-chan *WSMessage { return c.manager.Events() }

func (c *WSClient) Close() error { return c.manager.Close() }

type MetricsWSClient struct{ conn *websocket.Conn }

func NewMetricsWSClient(cfg *config.Config, token, serverID string) (*MetricsWSClient, error) {
	query := url.Values{}
	query.Set("token", token)
	query.Set("team_id", cfg.TeamID)
	query.Set("serverId", serverID)
	query.Set("interval", "2")
	wsURL, err := buildWebSocketURL(cfg.EffectiveAPIURL(), "/api/metrics/stream", query)
	if err != nil {
		return nil, err
	}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to metrics stream: %w", err)
	}
	return &MetricsWSClient{conn: conn}, nil
}

func (c *MetricsWSClient) ReadRaw() ([]byte, error) {
	_, data, err := c.conn.ReadMessage()
	return data, err
}

func (c *MetricsWSClient) Close() error { return c.conn.Close() }

type LogsWSClient struct{ conn *websocket.Conn }

func NewLogsWSClient(cfg *config.Config, token, serverID, entity, entityID string) (*LogsWSClient, error) {
	return newLogsWSClient(cfg.EffectiveAPIURL(), token, cfg.TeamID, serverID, entity, entityID)
}

func NewLogsWSClientDirect(token, teamID, serverID, entity, entityID string, apiURLs ...string) (*LogsWSClient, error) {
	baseURL := config.DefaultAPIURL
	if len(apiURLs) > 0 && apiURLs[0] != "" {
		baseURL = apiURLs[0]
	}
	return newLogsWSClient(baseURL, token, teamID, serverID, entity, entityID)
}

func newLogsWSClient(baseURL, token, teamID, serverID, entity, entityID string) (*LogsWSClient, error) {
	query := url.Values{}
	query.Set("token", token)
	query.Set("team_id", teamID)
	query.Set("serverId", serverID)
	query.Set("entity", entity)
	query.Set("entityId", entityID)
	query.Set("tail", "500")
	wsURL, err := buildWebSocketURL(baseURL, "/api/terminal/logs", query)
	if err != nil {
		return nil, err
	}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to log stream: %w", err)
	}
	return &LogsWSClient{conn: conn}, nil
}

func (c *LogsWSClient) ReadMessage() (string, error) {
	_, data, err := c.conn.ReadMessage()
	return string(data), err
}

func (c *LogsWSClient) Close() error { return c.conn.Close() }

func buildWebSocketURL(baseURL, requestPath string, query url.Values) (string, error) {
	httpURL, err := resolveRequestURL(baseURL, requestPath)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(httpURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse websocket URL: %w", err)
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return "", fmt.Errorf("unsupported websocket URL scheme %q", u.Scheme)
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}
