package outbox

import (
	"context"
	"encoding/json"

	"investment_monitoring/internal/repository/entity"
)

type Service struct {
	store entity.Store
}

func NewService(store entity.Store) *Service {
	return &Service{store: store}
}

func (s *Service) Publish(ctx context.Context, event Event) error {
	_, err := s.store.CreateOutboxEvent(ctx, toCreateParams(event))
	return err
}

func (s *Service) PublishTx(ctx context.Context, q *entity.Queries, event Event) error {
	_, err := q.CreateOutboxEvent(ctx, toCreateParams(event))
	return err
}

func toCreateParams(event Event) entity.CreateOutboxEventParams {
	return entity.CreateOutboxEventParams{
		EventType:     event.EventType,
		AggregateType: event.AggregateType,
		AggregateID:   event.AggregateID,
		Payload:       marshalPayload(event.Payload),
	}
}

func marshalPayload(payload any) []byte {
	if payload == nil {
		return []byte("{}")
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return []byte(`{"marshal_error":true}`)
	}
	return b
}
