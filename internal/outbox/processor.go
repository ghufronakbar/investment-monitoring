package outbox

import (
	"context"
	"database/sql"
	"log"

	"investment_monitoring/internal/repository/entity"
)

type Processor struct {
	store     entity.Store
	publisher RealtimePublisher
	mapper    *RealtimeMapper
}

func NewProcessor(store entity.Store, publisher RealtimePublisher) *Processor {
	return &Processor{
		store:     store,
		publisher: publisher,
		mapper:    NewRealtimeMapper(store),
	}
}

func (p *Processor) ProcessBatch(ctx context.Context, limit int32) error {
	if limit <= 0 {
		limit = 10
	}

	events, err := p.store.ListPendingOutboxEvents(ctx, limit)
	if err != nil {
		return err
	}

	for _, event := range events {
		claimed, err := p.store.MarkOutboxEventProcessing(ctx, event.ID)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return err
		}

		log.Printf("processing outbox event id=%s type=%s aggregate=%s:%s", claimed.ID, claimed.EventType, claimed.AggregateType, claimed.AggregateID)

		if p.publisher != nil {
			envelopes, err := p.mapper.Map(ctx, claimed)
			if err != nil {
				_, _ = p.store.MarkOutboxEventFailed(ctx, claimed.ID)
				return err
			}
			for _, envelope := range envelopes {
				if err := p.publisher.Publish(ctx, envelope); err != nil {
					_, _ = p.store.MarkOutboxEventFailed(ctx, claimed.ID)
					return err
				}
				log.Printf("published realtime event outbox_id=%s type=%s topic=%s", claimed.ID, envelope.Type, envelope.Topic)
			}
		}

		if _, err := p.store.MarkOutboxEventProcessed(ctx, claimed.ID); err != nil {
			_, _ = p.store.MarkOutboxEventFailed(ctx, claimed.ID)
			return err
		}
	}
	return nil
}
