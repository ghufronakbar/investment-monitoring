package sla_service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"investment_monitoring/internal/repository/entity"

	"github.com/google/uuid"
)

type fakeStore struct {
	entity.Store

	upsertArg entity.UpsertSLARecordParams
	upsertErr error
	record    entity.SlaRecord
}

func (f *fakeStore) UpsertSLARecord(ctx context.Context, arg entity.UpsertSLARecordParams) (entity.SlaRecord, error) {
	f.upsertArg = arg
	if f.upsertErr != nil {
		return entity.SlaRecord{}, f.upsertErr
	}
	f.record = entity.SlaRecord{
		ID:         uuid.New(),
		EntityType: arg.EntityType,
		EntityID:   arg.EntityID,
		State:      arg.State,
		OwnerOrgID: arg.OwnerOrgID,
		EnteredAt:  arg.EnteredAt,
		DueAt:      arg.DueAt,
		Status:     arg.Status,
		ResolvedAt: sql.NullTime{},
	}
	return f.record, nil
}

func TestStartTicketUpsertsSLARecord(t *testing.T) {
	now := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)
	store := &fakeStore{}
	svc := NewService(store)
	svc.now = func() time.Time { return now }

	ticketID := uuid.New()
	ownerOrgID := uuid.New()
	got, err := svc.StartTicket(context.Background(), ticketID, ownerOrgID, "OPEN")
	if err != nil {
		t.Fatalf("StartTicket returned error: %v", err)
	}

	if store.upsertArg.EntityType != EntityTypeTicket {
		t.Fatalf("entity type = %s, want %s", store.upsertArg.EntityType, EntityTypeTicket)
	}
	if store.upsertArg.EntityID != ticketID {
		t.Fatalf("entity id = %s, want %s", store.upsertArg.EntityID, ticketID)
	}
	if !store.upsertArg.OwnerOrgID.Valid || store.upsertArg.OwnerOrgID.UUID != ownerOrgID {
		t.Fatalf("owner org id = %+v, want %s", store.upsertArg.OwnerOrgID, ownerOrgID)
	}
	if !store.upsertArg.EnteredAt.Equal(now) {
		t.Fatalf("entered at = %v, want %v", store.upsertArg.EnteredAt, now)
	}
	if !store.upsertArg.DueAt.Equal(now.Add(DefaultTicketSLADuration)) {
		t.Fatalf("due at = %v, want %v", store.upsertArg.DueAt, now.Add(DefaultTicketSLADuration))
	}
	if got.Status != StatusOnTrack {
		t.Fatalf("status = %s, want %s", got.Status, StatusOnTrack)
	}
	if got.RemainingSeconds != int64(DefaultTicketSLADuration.Seconds()) {
		t.Fatalf("remaining seconds = %d, want %d", got.RemainingSeconds, int64(DefaultTicketSLADuration.Seconds()))
	}
}

func TestEffectiveStatus(t *testing.T) {
	now := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)
	svc := NewService(nil)

	tests := []struct {
		name   string
		record entity.SlaRecord
		want   string
	}{
		{
			name: "on track",
			record: entity.SlaRecord{
				DueAt: now.Add(48 * time.Hour),
			},
			want: StatusOnTrack,
		},
		{
			name: "warning",
			record: entity.SlaRecord{
				DueAt: now.Add(DefaultWarningThreshold),
			},
			want: StatusWarning,
		},
		{
			name: "violated",
			record: entity.SlaRecord{
				DueAt: now.Add(-time.Minute),
			},
			want: StatusViolated,
		},
		{
			name: "resolved before due",
			record: entity.SlaRecord{
				DueAt:      now.Add(time.Hour),
				ResolvedAt: sql.NullTime{Time: now, Valid: true},
			},
			want: StatusResolved,
		},
		{
			name: "resolved after due remains violated",
			record: entity.SlaRecord{
				DueAt:      now.Add(-time.Hour),
				ResolvedAt: sql.NullTime{Time: now, Valid: true},
			},
			want: StatusViolated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := svc.effectiveStatus(tt.record, now); got != tt.want {
				t.Fatalf("status = %s, want %s", got, tt.want)
			}
		})
	}
}
