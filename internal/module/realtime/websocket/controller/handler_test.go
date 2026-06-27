package websocket_controller

import (
	"context"
	"encoding/json"
	"net/http"
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
	wsSvc := websocket_service.NewService(cfg, tokenSvc, nil, nil)
	handler := NewHandler(wsSvc)

	server := newSkippableTestServer(t, handler.Routes())
	defer server.Close()

	userID := uuid.New()
	accessToken, err := tokenSvc.GenerateAccessToken(token_service.GenerateAccessTokenInput{
		ID:    userID,
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

	subscribeMessage := `{"type":"subscribe","topics":["user.` + userID.String() + `","user.` + uuid.NewString() + `"]}`
	if err := conn.Write(ctx, websocket.MessageText, []byte(subscribeMessage)); err != nil {
		t.Fatal(err)
	}

	_, data, err = conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var subscription websocket_service.Envelope
	if err := json.Unmarshal(data, &subscription); err != nil {
		t.Fatal(err)
	}
	if subscription.Type != websocket_service.EventRealtimeSubscriptionUpdated {
		t.Fatalf("event = %s, want %s", subscription.Type, websocket_service.EventRealtimeSubscriptionUpdated)
	}

	dataBytes, err := json.Marshal(subscription.Data)
	if err != nil {
		t.Fatal(err)
	}
	var updated websocket_service.SubscriptionUpdatedData
	if err := json.Unmarshal(dataBytes, &updated); err != nil {
		t.Fatal(err)
	}
	if len(updated.Subscribed) != 1 || updated.Subscribed[0] != "user."+userID.String() {
		t.Fatalf("subscribed = %#v, want own user topic", updated.Subscribed)
	}
	if len(updated.Rejected) != 1 {
		t.Fatalf("rejected = %#v, want one rejected topic", updated.Rejected)
	}
}

func newSkippableTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Skipf("httptest listener unavailable in this environment: %v", r)
		}
	}()

	return httptest.NewServer(handler)
}
