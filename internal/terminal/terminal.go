package terminal

import (
	"bytes"
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
	ServerID   string
	SiteID     string
	Username   string
	Token      string
	ServerName string
	ServerIP   string
}

func Connect(cfg *config.Config, opts Options) error {
	wsURL, err := buildURL(cfg, opts)
	if err != nil {
		return fmt.Errorf("failed to build terminal URL: %w", err)
	}

	cols, rows, _ := getTerminalSize()

	frame := newFrame(opts.ServerName, opts.ServerIP, opts.Username)
	frame.draw(cols, rows)
	setupScrollRegion(frame.headerRows, frame.footerRows, rows)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		resetScrollRegion()
		return fmt.Errorf("failed to connect to terminal: %w", err)
	}

	enableMouseTracking()

	oldState, err := makeRaw(os.Stdin.Fd())
	if err != nil {
		disableMouseTracking()
		resetScrollRegion()
		conn.Close()
		return fmt.Errorf("failed to set terminal to raw mode: %w", err)
	}

	adjustedRows := rows - frame.headerRows - frame.footerRows
	if adjustedRows < 1 {
		adjustedRows = 1
	}
	if err := sendResizeWithSize(conn, cols, adjustedRows); err != nil {
		restore(os.Stdin.Fd(), oldState)
		resetScrollRegion()
		conn.Close()
		return fmt.Errorf("failed to send initial resize: %w", err)
	}

	done := make(chan struct{})
	var once sync.Once
	closeDone := func() { once.Do(func() { close(done) }) }

	go handleOutput(conn, os.Stdout, closeDone, frame)
	go handleInput(conn, os.Stdin, closeDone)
	go handleResizeWithFrame(conn, done, frame)
	go keepAlive(conn, done)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-done:
	case <-sigCh:
		closeDone()
	}

	signal.Stop(sigCh)

	conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	time.Sleep(150 * time.Millisecond)
	conn.Close()

	restore(os.Stdin.Fd(), oldState)
	disableMouseTracking()
	resetScrollRegion()

	return nil
}

func buildURL(cfg *config.Config, opts Options) (string, error) {
	baseURL := strings.TrimRight(cfg.EffectiveAPIURL(), "/")
	endpoint := "/api/terminal/ws"
	if strings.HasSuffix(baseURL, "/api") {
		endpoint = "/terminal/ws"
	}
	u, err := url.Parse(baseURL + endpoint)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return "", fmt.Errorf("unsupported API URL scheme %q", u.Scheme)
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

func handleOutput(conn *websocket.Conn, w io.Writer, closeDone func(), f *frame) {
	defer closeDone()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}

		// Data arrived — session is alive, clear any pending deadline
		conn.SetReadDeadline(time.Time{})

		_, rows, _ := getTerminalSize()
		regionTop := []byte(fmt.Sprintf("\033[%d;1H", f.headerRows+1))
		regionClear := []byte(f.regionClearSeq())
		scrollRegion := []byte(f.scrollRegionSeq(rows))

		// Filter sequences that break the frame
		message = bytes.ReplaceAll(message, []byte{0x1b, 'c'}, regionClear)
		message = bytes.ReplaceAll(message, []byte("\033[r"), scrollRegion)
		message = bytes.ReplaceAll(message, []byte("\033[2J"), regionClear)
		message = bytes.ReplaceAll(message, []byte("\033[3J"), nil)
		message = bytes.ReplaceAll(message, []byte("\033[H"), regionTop)

		w.Write(message)

		// Redraw footer in case clear sequences wiped it
		f.ensureFooter()

		// Detect shell exit — "logout" appears when bash/zsh session ends
		if bytes.Contains(message, []byte("logout")) {
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		}
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

		// Detect Ctrl+D (EOF signal) — the user intends to disconnect.
		// Set a delayed read deadline so handleOutput's ReadMessage unblocks
		// if the server stops sending data after the shell exits.
		if bytes.IndexByte(buf[:n], 0x04) >= 0 {
			go func() {
				time.Sleep(3 * time.Second)
				conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			}()
		}
	}
}

func handleResizeWithFrame(conn *websocket.Conn, done <-chan struct{}, f *frame) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	for {
		select {
		case <-done:
			return
		case <-sigCh:
			cols, rows, err := getTerminalSize()
			if err != nil {
				continue
			}

			f.redraw(cols, rows)
			setupScrollRegion(f.headerRows, f.footerRows, rows)

			adjustedRows := rows - f.headerRows - f.footerRows
			if adjustedRows < 1 {
				adjustedRows = 1
			}
			sendResizeWithSize(conn, cols, adjustedRows)
		}
	}
}

func sendResizeWithSize(conn *websocket.Conn, cols, rows int) error {
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
