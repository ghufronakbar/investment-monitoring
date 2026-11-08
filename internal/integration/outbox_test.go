//go:build integration

package integration

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"

	websocket_service "investment_monitoring/internal/module/realtime/websocket/service"
	"investment_monitoring/internal/outbox"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/internal/testutil"

	"github.com/google/uuid"
)

// recordingPublisher counts deliveries per aggregate and fails on demand.
type recordingPublisher struct {
	mu        sync.Mutex
	failFor   string
	delivered map[string]int
}

func (p *recordingPublisher) Publish(_ context.Context, env websocket_service.Envelope) error {
	if p.failFor != "" && strings.Contains(env.Topic, p.failFor) {
		return errors.New("simulated broker outage")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.delivered == nil {
		p.delivered = map[string]int{}
	}
	p.delivered[env.Topic]++
	return nil
}

func (p *recordingPublisher) count(topic string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.delivered[topic]
}

func paymentPaidEvent(projectID, paymentID uuid.UUID) outbox.Event {
	return outbox.Event{
		EventType:     outbox.EventPaymentPaid,
		AggregateType: outbox.AggregatePaymentRequest,
		AggregateID:   paymentID,
		Payload:       map[string]any{"projectId": projectID, "paymentRequestId": paymentID},
	}
}

func outboxStatus(t *testing.T, db *sql.DB, aggregateID uuid.UUID) (status string, found bool) {
	t.Helper()
	err := db.QueryRow(`SELECT status FROM outbox_events WHERE aggregate_id = $1`, aggregateID).Scan(&status)
	if err == sql.ErrNoRows {
		return "", false
	}
	if err != nil {
		t.Fatalf("outbox status: %v", err)
	}
	return status, true
}

func paymentStatus(t *testing.T, db *sql.DB, id uuid.UUID) string {
	t.Helper()
	var status string
	if err := db.QueryRow(`SELECT status FROM payment_requests WHERE id = $1`, id).Scan(&status); err != nil {
		t.Fatalf("payment status: %v", err)
	}
	return status
}

// drain runs the worker until the given event leaves PENDING, the way the
// polling loop in cmd/worker would.
func drain(t *testing.T, db *sql.DB, p *outbox.Processor, aggregateID uuid.UUID) {
	t.Helper()
	for i := 0; i < 50; i++ {
		if status, _ := outboxStatus(t, db, aggregateID); status != "PENDING" {
			return
		}
		_ = p.ProcessBatch(context.Background(), 100)
	}
	t.Fatalf("event %s still PENDING after draining", aggregateID)
}

func TestOutboxDeliveryGuarantees(t *testing.T) {
	db, store := testutil.OpenTestDB(t)
	f := testutil.NewFixture(t, db)
	ctx := context.Background()
	project, _ := projectWithAllPositions(f)
	pub := outbox.NewService(store)

	// writeBusinessChange updates the business row and records the event in one
	// transaction, as every governed action does.
	writeBusinessChange := func(paymentID uuid.UUID, failAfterWrites bool) error {
		return store.ExecTx(ctx, func(q *entity.Queries) error {
			if _, err := q.DisburseByLoanProvider(ctx, entity.DisburseByLoanProviderParams{
				ID: paymentID, PaymentProofImageUrls: []string{"https://example.test/proof.png"},
			}); err != nil {
				return err
			}
			if err := pub.PublishTx(ctx, q, paymentPaidEvent(project.ID, paymentID)); err != nil {
				return err
			}
			if failAfterWrites {
				return errors.New("simulated failure after both writes")
			}
			return nil
		})
	}

	t.Run("OUTBOX-01 rollback discards both the business change and the event", func(t *testing.T) {
		paymentID := f.PaymentRequest(project.ContractID, entity.PaymentRequestStatusEnumAGENCYAPPROVED)
		if err := writeBusinessChange(paymentID, true); err == nil {
			t.Fatal("expected the transaction to fail")
		}
		if got := paymentStatus(t, db, paymentID); got != "AGENCY_APPROVED" {
			t.Fatalf("payment status = %s, want AGENCY_APPROVED", got)
		}
		if _, found := outboxStatus(t, db, paymentID); found {
			t.Fatal("outbox event persisted although the transaction rolled back")
		}
	})

	t.Run("OUTBOX-02 commit persists the business change and a PENDING event together", func(t *testing.T) {
		paymentID := f.PaymentRequest(project.ContractID, entity.PaymentRequestStatusEnumAGENCYAPPROVED)
		if err := writeBusinessChange(paymentID, false); err != nil {
			t.Fatalf("transaction: %v", err)
		}
		if got := paymentStatus(t, db, paymentID); got != "PAID" {
			t.Fatalf("payment status = %s, want PAID", got)
		}
		if status, found := outboxStatus(t, db, paymentID); !found || status != "PENDING" {
			t.Fatalf("outbox status = %q (found=%v), want PENDING", status, found)
		}
	})

	t.Run("OUTBOX-03 concurrent workers claim an event exactly once", func(t *testing.T) {
		paymentID := uuid.New()
		event, err := store.CreateOutboxEvent(ctx, entity.CreateOutboxEventParams{
			EventType: outbox.EventPaymentPaid, AggregateType: outbox.AggregatePaymentRequest,
			AggregateID: paymentID, Payload: []byte(`{}`),
		})
		if err != nil {
			t.Fatalf("create event: %v", err)
		}

		const workers = 16
		var wg sync.WaitGroup
		var mu sync.Mutex
		claimed, lost := 0, 0
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := store.MarkOutboxEventProcessing(ctx, event.ID)
				mu.Lock()
				defer mu.Unlock()
				switch {
				case err == nil:
					claimed++
				case errors.Is(err, sql.ErrNoRows):
					lost++
				default:
					t.Errorf("claim: %v", err)
				}
			}()
		}
		wg.Wait()
		if claimed != 1 || lost != workers-1 {
			t.Fatalf("claimed=%d lost=%d, want 1 and %d", claimed, lost, workers-1)
		}
	})

	t.Run("OUTBOX-04 worker publishes a committed event once and marks it PROCESSED", func(t *testing.T) {
		paymentID := f.PaymentRequest(project.ContractID, entity.PaymentRequestStatusEnumAGENCYAPPROVED)
		if err := writeBusinessChange(paymentID, false); err != nil {
			t.Fatalf("transaction: %v", err)
		}
		recorder := &recordingPublisher{}
		processor := outbox.NewProcessor(store, recorder)
		drain(t, db, processor, paymentID)
		_ = processor.ProcessBatch(ctx, 100)

		if status, _ := outboxStatus(t, db, paymentID); status != "PROCESSED" {
			t.Fatalf("outbox status = %s, want PROCESSED", status)
		}
		if n := recorder.count("payment." + paymentID.String()); n != 1 {
			t.Fatalf("deliveries = %d, want 1", n)
		}
	})

	// The next two cases document current behaviour that the paper reports as a
	// limitation: failed or interrupted events are not retried, so the realtime
	// signal is at-most-once and clients rely on REST refetch for consistency.
	t.Run("OUTBOX-05 publish failure marks the event FAILED and it is not retried", func(t *testing.T) {
		paymentID := f.PaymentRequest(project.ContractID, entity.PaymentRequestStatusEnumAGENCYAPPROVED)
		if err := writeBusinessChange(paymentID, false); err != nil {
			t.Fatalf("transaction: %v", err)
		}
		recorder := &recordingPublisher{failFor: paymentID.String()}
		processor := outbox.NewProcessor(store, recorder)
		drain(t, db, processor, paymentID)

		if status, _ := outboxStatus(t, db, paymentID); status != "FAILED" {
			t.Fatalf("outbox status = %s, want FAILED", status)
		}
		recorder.failFor = ""
		_ = processor.ProcessBatch(ctx, 100)
		if n := recorder.count("payment." + paymentID.String()); n != 0 {
			t.Fatalf("FAILED event was re-delivered %d times", n)
		}
		if got := paymentStatus(t, db, paymentID); got != "PAID" {
			t.Fatalf("payment status = %s, want PAID: broker failure must not affect the source of truth", got)
		}
	})

	t.Run("OUTBOX-06 event claimed by a crashed worker stays PROCESSING", func(t *testing.T) {
		paymentID := f.PaymentRequest(project.ContractID, entity.PaymentRequestStatusEnumAGENCYAPPROVED)
		if err := writeBusinessChange(paymentID, false); err != nil {
			t.Fatalf("transaction: %v", err)
		}
		var eventID uuid.UUID
		if err := db.QueryRow(`SELECT id FROM outbox_events WHERE aggregate_id = $1`, paymentID).Scan(&eventID); err != nil {
			t.Fatalf("event id: %v", err)
		}
		if _, err := store.MarkOutboxEventProcessing(ctx, eventID); err != nil {
			t.Fatalf("claim: %v", err)
		}

		recorder := &recordingPublisher{}
		_ = outbox.NewProcessor(store, recorder).ProcessBatch(ctx, 100)
		if status, _ := outboxStatus(t, db, paymentID); status != "PROCESSING" {
			t.Fatalf("outbox status = %s, want PROCESSING", status)
		}
		if n := recorder.count("payment." + paymentID.String()); n != 0 {
			t.Fatalf("interrupted event was delivered %d times", n)
		}
	})
}
