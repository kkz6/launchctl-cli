package terminal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kkz6/launchctl/internal/config"
	"golang.org/x/sys/unix"
)

type resizeMessage struct {
	Type string `json:"type"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

type Options struct {
	ServerID string
	SiteID   string
	Username string
	Token    string
}

func Connect(cfg *config.Config, opts Options) error {
	wsURL, err := buildURL(cfg, opts)
	if err != nil {
		return fmt.Errorf("failed to build terminal URL: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to terminal: %w", err)
	}
	defer conn.Close()

	oldState, err := makeRaw(os.Stdin.Fd())
	if err != nil {
		return fmt.Errorf("failed to set terminal to raw mode: %w", err)
	}
	defer restore(os.Stdin.Fd(), oldState)

	if err := sendResize(conn); err != nil {
		return fmt.Errorf("failed to send initial resize: %w", err)
	}

	done := make(chan struct{})
	var once sync.Once
	closeDone := func() { once.Do(func() { close(done) }) }

	go handleOutput(conn, os.Stdout, closeDone)
	go handleInput(conn, os.Stdin, closeDone)
	go handleResize(conn, done)
	go keepAlive(conn, done)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-done:
	case <-sigCh:
		closeDone()
	}

	conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	return nil
}

func buildURL(cfg *config.Config, opts Options) (string, error) {
	baseURL := config.APIURL
	baseURL = strings.Replace(baseURL, "https://", "wss://", 1)
	baseURL = strings.Replace(baseURL, "http://", "ws://", 1)

	u, err := url.Parse(baseURL + "/api/terminal/ws")
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("serverId", opts.ServerID)
	q.Set("token", opts.Token)
	q.Set("team_id", cfg.TeamID)

	if opts.SiteID != "" {
		q.Set("siteId", opts.SiteID)
	}

	username := opts.Username
	if username == "" {
		username = "launcher"
	}
	q.Set("username", username)

	u.RawQuery = q.Encode()

	return u.String(), nil
}

func handleOutput(conn *websocket.Conn, w io.Writer, closeDone func()) {
	defer closeDone()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}

		w.Write(message)
	}
}

func handleInput(conn *websocket.Conn, r io.Reader, closeDone func()) {
	defer closeDone()

	buf := make([]byte, 8192)
	for {
		n, err := r.Read(buf)
		if err != nil {
			return
		}

		if err := conn.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
			return
		}
	}
}

func handleResize(conn *websocket.Conn, done <-chan struct{}) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	for {
		select {
		case <-done:
			return
		case <-sigCh:
			sendResize(conn)
		}
	}
}

func sendResize(conn *websocket.Conn) error {
	cols, rows, err := getTerminalSize()
	if err != nil {
		return err
	}

	msg := resizeMessage{
		Type: "resize",
		Cols: cols,
		Rows: rows,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, data)
}

func keepAlive(conn *websocket.Conn, done <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
				return
			}
		}
	}
}

func getTerminalSize() (int, int, error) {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 80, 24, nil
	}

	return int(ws.Col), int(ws.Row), nil
}
