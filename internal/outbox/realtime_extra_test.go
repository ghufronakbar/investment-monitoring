package outbox

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"investment_monitoring/internal/repository/entity"

	"github.com/google/uuid"
)

func TestRealtimeMapperReturnsNilForUnknownEvent(t *testing.T) {
	envelopes, err := NewRealtimeMapper(nil).Map(context.Background(), entity.OutboxEvent{
		ID:        uuid.New(),
		EventType: "UnknownEvent",
	})
	if err != nil {
		t.Fatalf("Map returned error: %v", err)
	}
	if envelopes != nil {
		t.Fatalf("envelopes = %+v, want nil", envelopes)
	}
}

func TestRealtimeMapperRequiresStoreWhenProjectIDMissing(t *testing.T) {
	_, err := NewRealtimeMapper(nil).Map(context.Background(), entity.OutboxEvent{
		ID:            uuid.New(),
		EventType:     EventTicketCreated,
		AggregateType: AggregateTicket,
		AggregateID:   uuid.New(),
		Payload:       json.RawMessage(`{}`),
	})
	if err == nil {
		t.Fatal("expected missing project id/store error")
	}
	if !strings.Contains(err.Error(), "missing projectId") {
		t.Fatalf("err = %v, want missing projectId", err)
	}
}
