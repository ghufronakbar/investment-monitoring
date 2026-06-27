package websocket_controller

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"investment_monitoring/config"
	websocket_service "investment_monitoring/internal/module/realtime/websocket/service"
	token_service "investment_monitoring/internal/module/support/token/service"

	"github.com/google/uuid"
	"nhooyr.io/websocket"
)

func TestWebSocketConnectAndPing(t *testing.T) {
	cfg := &config.Config{
		JWT_ACCESS_TOKEN_SECRET:      "access-secret",
		JWT_REFRESH_TOKEN_SECRET:     "refresh-secret",
		JWT_ACCESS_TOKEN_EXPIRED:     time.Minute,
		WEBSOCKET_ENABLED:            true,
		WEBSOCKET_ALLOWED_ORIGINS:    []string{"*"},
		WEBSOCKET_READ_LIMIT_BYTES:   65536,
		WEBSOCKET_WRITE_TIMEOUT:      time.Second,
		WEBSOCKET_IDLE_TIMEOUT:       5 * time.Second,
		WEBSOCKET_CLIENT_SEND_BUFFER: 4,
	}
	tokenSvc := token_service.NewService(cfg)
	wsSvc := websocket_service.NewService(cfg, tokenSvc)
	handler := NewHandler(wsSvc)

	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	accessToken, err := tokenSvc.GenerateAccessToken(token_service.GenerateAccessTokenInput{
		ID:    uuid.New(),
		Email: "user@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/?accessToken=" + url.QueryEscape(accessToken)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "test done")

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var connected websocket_service.Envelope
	if err := json.Unmarshal(data, &connected); err != nil {
		t.Fatal(err)
	}
	if connected.Type != websocket_service.EventRealtimeConnected {
		t.Fatalf("initial event = %s, want %s", connected.Type, websocket_service.EventRealtimeConnected)
	}

	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"type":"ping"}`)); err != nil {
		t.Fatal(err)
	}

	_, data, err = conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var pong websocket_service.Envelope
	if err := json.Unmarshal(data, &pong); err != nil {
		t.Fatal(err)
	}
	if pong.Type != websocket_service.EventRealtimePong {
		t.Fatalf("event = %s, want %s", pong.Type, websocket_service.EventRealtimePong)
	}
}
