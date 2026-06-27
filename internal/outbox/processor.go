package outbox

import (
	"context"
	"database/sql"
	"log"

	"investment_monitoring/internal/repository/entity"
)

type Processor struct {
	store entity.Store
}

func NewProcessor(store entity.Store) *Processor {
	return &Processor{store: store}
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

		if _, err := p.store.MarkOutboxEventProcessed(ctx, claimed.ID); err != nil {
			_, _ = p.store.MarkOutboxEventFailed(ctx, claimed.ID)
			return err
		}
	}
	return nil
}
