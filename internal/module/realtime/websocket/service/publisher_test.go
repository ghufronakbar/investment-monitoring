package websocket_service

import (
	"context"
	"testing"

	"investment_monitoring/config"

	"github.com/google/uuid"
)

func TestPublishFallbackToLocalHubWhenRedisUnavailable(t *testing.T) {
	svc := NewService(&config.Config{}, nil, nil, nil)
	client := NewClient(uuid.New(), "user@example.com", 1)

	svc.hub.Register(client)
	defer svc.hub.Unregister(client)
	svc.hub.Subscribe(client, "project.1")

	err := svc.Publish(context.Background(), Envelope{
		Type:  "test.event",
		Topic: "project.1",
		Data:  "ok",
	})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-client.Send():
		if got.Type != "test.event" {
			t.Fatalf("event type = %s, want test.event", got.Type)
		}
	default:
		t.Fatal("expected local hub event")
	}
}
