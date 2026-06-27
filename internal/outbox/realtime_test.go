package outbox

import (
	"encoding/json"
	"testing"
	"time"

	websocket_service "investment_monitoring/internal/module/realtime/websocket/service"
	"investment_monitoring/internal/repository/entity"

	"github.com/google/uuid"
)

func TestRealtimeMapperMapTicketEvent(t *testing.T) {
	projectID := uuid.New()
	ticketID := uuid.New()
	event := testOutboxEvent(EventTicketCreated, AggregateTicket, ticketID, map[string]any{
		"projectId": projectID,
		"ticketId":  ticketID,
	})

	envelopes, err := NewRealtimeMapper(nil).Map(t.Context(), event)
	if err != nil {
		t.Fatal(err)
	}

	if len(envelopes) != 2 {
		t.Fatalf("len(envelopes) = %d, want 2", len(envelopes))
	}
	assertEnvelope(t, envelopes[0], RealtimeTicketCreated, "project."+projectID.String()+".ticket")
	assertEnvelope(t, envelopes[1], RealtimeTicketCreated, "ticket."+ticketID.String())
}

func TestRealtimeMapperMapPaymentEvent(t *testing.T) {
	projectID := uuid.New()
	paymentID := uuid.New()
	event := testOutboxEvent(EventPaymentPaid, AggregatePaymentRequest, paymentID, map[string]any{
		"projectId":        projectID,
		"paymentRequestId": paymentID,
	})

	envelopes, err := NewRealtimeMapper(nil).Map(t.Context(), event)
	if err != nil {
		t.Fatal(err)
	}

	if len(envelopes) != 2 {
		t.Fatalf("len(envelopes) = %d, want 2", len(envelopes))
	}
	assertEnvelope(t, envelopes[0], RealtimePaymentPaid, "project."+projectID.String()+".finance")
	assertEnvelope(t, envelopes[1], RealtimePaymentPaid, "payment."+paymentID.String())
}

func TestRealtimeMapperMapMemberEvent(t *testing.T) {
	organizationID := uuid.New()
	userID := uuid.New()
	event := testOutboxEvent(EventMemberInvited, AggregateUser, userID, map[string]any{
		"organizationId": organizationID,
		"userId":         userID,
	})

	envelopes, err := NewRealtimeMapper(nil).Map(t.Context(), event)
	if err != nil {
		t.Fatal(err)
	}

	if len(envelopes) != 2 {
		t.Fatalf("len(envelopes) = %d, want 2", len(envelopes))
	}
	assertEnvelope(t, envelopes[0], RealtimeMemberInvited, "organization."+organizationID.String())
	assertEnvelope(t, envelopes[1], RealtimeMemberInvited, "user."+userID.String())
}

func TestRealtimeMapperUnknownEvent(t *testing.T) {
	event := testOutboxEvent("Unknown", "unknown", uuid.New(), map[string]any{})

	envelopes, err := NewRealtimeMapper(nil).Map(t.Context(), event)
	if err != nil {
		t.Fatal(err)
	}
	if len(envelopes) != 0 {
		t.Fatalf("len(envelopes) = %d, want 0", len(envelopes))
	}
}

func testOutboxEvent(eventType, aggregateType string, aggregateID uuid.UUID, payload map[string]any) entity.OutboxEvent {
	b, _ := json.Marshal(payload)
	return entity.OutboxEvent{
		ID:            uuid.New(),
		EventType:     eventType,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		Payload:       b,
		Status:        "PENDING",
		CreatedAt:     time.Now().UTC(),
	}
}

func assertEnvelope(t *testing.T, envelope websocket_service.Envelope, eventType, topic string) {
	t.Helper()

	if envelope.Type != eventType {
		t.Fatalf("type = %s, want %s", envelope.Type, eventType)
	}
	if envelope.Topic != topic {
		t.Fatalf("topic = %s, want %s", envelope.Topic, topic)
	}
}
