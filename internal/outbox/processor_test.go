package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	websocket_service "investment_monitoring/internal/module/realtime/websocket/service"
	"investment_monitoring/internal/repository/entity"

	"github.com/google/uuid"
)

type fakeProcessorStore struct {
	entity.Store

	events        []entity.OutboxEvent
	listErr       error
	processingErr error
	processedErr  error
	failedIDs     []uuid.UUID
	processedIDs  []uuid.UUID
}

func (f *fakeProcessorStore) ListPendingOutboxEvents(ctx context.Context, limit int32) ([]entity.OutboxEvent, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.events, nil
}

func (f *fakeProcessorStore) MarkOutboxEventProcessing(ctx context.Context, id uuid.UUID) (entity.OutboxEvent, error) {
	if f.processingErr != nil {
		return entity.OutboxEvent{}, f.processingErr
	}
	for _, event := range f.events {
		if event.ID == id {
			return event, nil
		}
	}
	return entity.OutboxEvent{}, sql.ErrNoRows
}

func (f *fakeProcessorStore) MarkOutboxEventProcessed(ctx context.Context, id uuid.UUID) (entity.OutboxEvent, error) {
	f.processedIDs = append(f.processedIDs, id)
	if f.processedErr != nil {
		return entity.OutboxEvent{}, f.processedErr
	}
	return entity.OutboxEvent{ID: id}, nil
}

func (f *fakeProcessorStore) MarkOutboxEventFailed(ctx context.Context, id uuid.UUID) (entity.OutboxEvent, error) {
	f.failedIDs = append(f.failedIDs, id)
	return entity.OutboxEvent{ID: id}, nil
}

type fakePublisher struct {
	envelopes []websocket_service.Envelope
	err       error
}

func (f *fakePublisher) Publish(ctx context.Context, envelope websocket_service.Envelope) error {
	if f.err != nil {
		return f.err
	}
	f.envelopes = append(f.envelopes, envelope)
	return nil
}

func TestProcessorPublishesAndMarksProcessed(t *testing.T) {
	projectID := uuid.New()
	ticketID := uuid.New()
	eventID := uuid.New()
	store := &fakeProcessorStore{
		events: []entity.OutboxEvent{{
			ID:            eventID,
			EventType:     EventTicketCreated,
			AggregateType: AggregateTicket,
			AggregateID:   ticketID,
			Payload:       mustPayload(t, map[string]string{"projectId": projectID.String(), "ticketId": ticketID.String()}),
			CreatedAt:     time.Now(),
		}},
	}
	publisher := &fakePublisher{}

	if err := NewProcessor(store, publisher).ProcessBatch(context.Background(), 0); err != nil {
		t.Fatalf("ProcessBatch returned error: %v", err)
	}
	if len(publisher.envelopes) != 2 {
		t.Fatalf("published envelopes = %d, want 2", len(publisher.envelopes))
	}
	if len(store.processedIDs) != 1 || store.processedIDs[0] != eventID {
		t.Fatalf("processed ids = %+v, want event id", store.processedIDs)
	}
	if len(store.failedIDs) != 0 {
		t.Fatalf("failed ids = %+v, want none", store.failedIDs)
	}
}

func TestProcessorMarksFailedWhenPublishFails(t *testing.T) {
	eventID := uuid.New()
	store := &fakeProcessorStore{
		events: []entity.OutboxEvent{{
			ID:            eventID,
			EventType:     EventMemberInvited,
			AggregateType: AggregateUser,
			AggregateID:   uuid.New(),
			Payload:       mustPayload(t, map[string]string{"userId": uuid.NewString()}),
			CreatedAt:     time.Now(),
		}},
	}
	publisher := &fakePublisher{err: errors.New("publish failed")}

	if err := NewProcessor(store, publisher).ProcessBatch(context.Background(), 10); err == nil {
		t.Fatal("expected publish error")
	}
	if len(store.failedIDs) != 1 || store.failedIDs[0] != eventID {
		t.Fatalf("failed ids = %+v, want event id", store.failedIDs)
	}
}

func TestProcessorSkipsAlreadyClaimedEvent(t *testing.T) {
	store := &fakeProcessorStore{
		events:        []entity.OutboxEvent{{ID: uuid.New()}},
		processingErr: sql.ErrNoRows,
	}
	if err := NewProcessor(store, &fakePublisher{}).ProcessBatch(context.Background(), 10); err != nil {
		t.Fatalf("ProcessBatch returned error: %v", err)
	}
	if len(store.processedIDs) != 0 {
		t.Fatalf("processed ids = %+v, want none", store.processedIDs)
	}
}

func mustPayload(t *testing.T, payload any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return raw
}
