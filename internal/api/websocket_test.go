package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kkz6/launchctl/internal/config"
)

func TestWSManagerReconnectsAndReplaysSubscriptions(t *testing.T) {
	var connections atomic.Int32
	serverErrors := make(chan string, 4)
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ws" || r.URL.Query().Get("token") != "jwt" || r.URL.Query().Get("team_id") != "team-a" {
			serverErrors <- r.URL.String()
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			serverErrors <- err.Error()
			return
		}
		defer conn.Close()

		connection := connections.Add(1)
		var subscription WSMessage
		if err := conn.ReadJSON(&subscription); err != nil {
			serverErrors <- err.Error()
			return
		}
		if subscription.Action != "subscribe" || subscription.Channel != "server.srv-a" {
			data, _ := json.Marshal(subscription)
			serverErrors <- string(data)
			return
		}
		if connection == 1 {
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseServiceRestart, "restart"), time.Now().Add(time.Second))
			return
		}
		_ = conn.WriteJSON(WSMessage{Event: "server.updated", Channel: "server.srv-a", Data: json.RawMessage(`{"server_id":"srv-a"}`)})
		<-time.After(50 * time.Millisecond)
	}))
	defer server.Close()

	manager := NewWSManager(&config.Config{APIURL: server.URL, TeamID: "team-a"}, "jwt")
	manager.minBackoff = 5 * time.Millisecond
	manager.maxBackoff = 10 * time.Millisecond
	manager.pingInterval = 20 * time.Millisecond
	manager.pongWait = 200 * time.Millisecond
	if err := manager.Subscribe("server.srv-a"); err != nil {
		t.Fatal(err)
	}
	if err := manager.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	select {
	case message := <-manager.Events():
		if message.Event != "server.updated" || message.Channel != "server.srv-a" {
			t.Fatalf("unexpected event: %#v", message)
		}
	case problem := <-serverErrors:
		t.Fatalf("websocket server error: %s", problem)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event after reconnect")
	}
	if connections.Load() < 2 {
		t.Fatalf("connections = %d, want at least 2", connections.Load())
	}
}

func TestWSManagerRejectsMalformedChannel(t *testing.T) {
	manager := NewWSManager(&config.Config{}, "jwt")
	for _, channel := range []string{"team:abc", "team.", "unknown.abc", "server.a.b"} {
		if err := manager.Subscribe(channel); err == nil {
			t.Fatalf("Subscribe(%q) succeeded", channel)
		}
	}
}
