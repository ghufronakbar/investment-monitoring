package outbox

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestToCreateParamsMarshalsPayload(t *testing.T) {
	aggregateID := uuid.New()

	got := toCreateParams(Event{
		EventType:     EventTicketCreated,
		AggregateType: AggregateTicket,
		AggregateID:   aggregateID,
		Payload: map[string]string{
			"projectId": uuid.NewString(),
		},
	})

	if got.EventType != EventTicketCreated {
		t.Fatalf("event type = %s, want %s", got.EventType, EventTicketCreated)
	}
	if got.AggregateType != AggregateTicket {
		t.Fatalf("aggregate type = %s, want %s", got.AggregateType, AggregateTicket)
	}
	if got.AggregateID != aggregateID {
		t.Fatalf("aggregate id = %s, want %s", got.AggregateID, aggregateID)
	}

	var payload map[string]string
	if err := json.Unmarshal(got.Payload, &payload); err != nil {
		t.Fatalf("payload unmarshal: %v", err)
	}
	if payload["projectId"] == "" {
		t.Fatalf("payload = %s, want projectId", got.Payload)
	}
}

func TestMarshalPayloadFallbacks(t *testing.T) {
	if got := string(marshalPayload(nil)); got != "{}" {
		t.Fatalf("nil payload = %s, want {}", got)
	}
	if got := string(marshalPayload(func() {})); got != `{"marshal_error":true}` {
		t.Fatalf("bad payload = %s, want marshal error marker", got)
	}
}
