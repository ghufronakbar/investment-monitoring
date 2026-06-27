package sla_service

import (
	"context"
	"database/sql"
	"time"

	"investment_monitoring/internal/repository/entity"

	"github.com/google/uuid"
)

const (
	EntityTypeTicket = "ticket"

	StatusOnTrack  = "ON_TRACK"
	StatusWarning  = "SLA_WARNING"
	StatusViolated = "SLA_VIOLATED"
	StatusResolved = "RESOLVED"
)

const (
	DefaultTicketSLADuration = 7 * 24 * time.Hour
	DefaultWarningThreshold  = 24 * time.Hour
)

type Service struct {
	store            entity.Store
	now              func() time.Time
	ticketDuration   time.Duration
	warningThreshold time.Duration
}

func NewService(store entity.Store) *Service {
	return &Service{
		store:            store,
		now:              time.Now,
		ticketDuration:   DefaultTicketSLADuration,
		warningThreshold: DefaultWarningThreshold,
	}
}

func (s *Service) StartTicket(ctx context.Context, ticketID uuid.UUID, ownerOrgID uuid.UUID, state string) (*SLAReturn, error) {
	return s.Upsert(ctx, EntityTypeTicket, ticketID, ownerOrgID, state, s.ticketDuration)
}

func (s *Service) MoveTicketState(ctx context.Context, ticketID uuid.UUID, ownerOrgID uuid.UUID, state string) (*SLAReturn, error) {
	return s.Upsert(ctx, EntityTypeTicket, ticketID, ownerOrgID, state, s.ticketDuration)
}

func (s *Service) ResolveTicket(ctx context.Context, ticketID uuid.UUID) (*SLAReturn, error) {
	resolvedAt := s.now()
	record, err := s.store.ResolveSLARecord(ctx, entity.ResolveSLARecordParams{
		EntityType: EntityTypeTicket,
		EntityID:   ticketID,
		ResolvedAt: sql.NullTime{Time: resolvedAt, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return s.mapRecord(record), nil
}

func (s *Service) GetTicket(ctx context.Context, ticketID uuid.UUID) (*SLAReturn, error) {
	record, err := s.store.GetSLARecordByEntity(ctx, entity.GetSLARecordByEntityParams{
		EntityType: EntityTypeTicket,
		EntityID:   ticketID,
	})
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.mapRecord(record), nil
}

func (s *Service) Upsert(ctx context.Context, entityType string, entityID uuid.UUID, ownerOrgID uuid.UUID, state string, duration time.Duration) (*SLAReturn, error) {
	enteredAt := s.now()
	record, err := s.store.UpsertSLARecord(ctx, entity.UpsertSLARecordParams{
		EntityType: entityType,
		EntityID:   entityID,
		State:      state,
		OwnerOrgID: uuid.NullUUID{UUID: ownerOrgID, Valid: ownerOrgID != uuid.Nil},
		EnteredAt:  enteredAt,
		DueAt:      enteredAt.Add(duration),
		Status:     StatusOnTrack,
	})
	if err != nil {
		return nil, err
	}
	return s.mapRecord(record), nil
}

func (s *Service) mapRecord(record entity.SlaRecord) *SLAReturn {
	now := s.now()
	status := s.effectiveStatus(record, now)
	remaining := int64(0)
	if !record.ResolvedAt.Valid {
		remaining = int64(record.DueAt.Sub(now).Seconds())
	}

	var ownerOrgID *uuid.UUID
	if record.OwnerOrgID.Valid {
		ownerOrgID = &record.OwnerOrgID.UUID
	}

	var resolvedAt *time.Time
	if record.ResolvedAt.Valid {
		resolvedAt = &record.ResolvedAt.Time
	}

	return &SLAReturn{
		ID:               record.ID,
		EntityType:       record.EntityType,
		EntityID:         record.EntityID,
		State:            record.State,
		OwnerOrgID:       ownerOrgID,
		EnteredAt:        record.EnteredAt,
		DueAt:            record.DueAt,
		ResolvedAt:       resolvedAt,
		Status:           status,
		RemainingSeconds: remaining,
		IsViolated:       status == StatusViolated,
	}
}

func (s *Service) effectiveStatus(record entity.SlaRecord, now time.Time) string {
	if record.ResolvedAt.Valid {
		if record.ResolvedAt.Time.After(record.DueAt) {
			return StatusViolated
		}
		return StatusResolved
	}
	if now.After(record.DueAt) {
		return StatusViolated
	}
	if record.DueAt.Sub(now) <= s.warningThreshold {
		return StatusWarning
	}
	return StatusOnTrack
}
