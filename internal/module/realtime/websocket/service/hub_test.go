package websocket_service

import (
	"testing"

	"github.com/google/uuid"
)

func TestHubSubscribePublishUnsubscribe(t *testing.T) {
	hub := NewHub()
	client := NewClient(uuid.New(), "user@example.com", 1)

	hub.Register(client)
	hub.Subscribe(client, "project.1")

	hub.Publish(Envelope{
		Type:  "test.event",
		Topic: "project.1",
		Data:  "ok",
	})

	select {
	case got := <-client.Send():
		if got.Type != "test.event" {
			t.Fatalf("event type = %s, want test.event", got.Type)
		}
	default:
		t.Fatal("expected published event")
	}

	hub.Unsubscribe(client, "project.1")
	hub.Publish(Envelope{
		Type:  "test.event",
		Topic: "project.1",
		Data:  "ok",
	})

	select {
	case got := <-client.Send():
		t.Fatalf("unexpected event after unsubscribe: %#v", got)
	default:
	}

	hub.Unregister(client)
}
