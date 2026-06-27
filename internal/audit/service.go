package audit

import (
	"context"
	"database/sql"
	"encoding/json"

	"investment_monitoring/internal/repository/entity"

	"github.com/google/uuid"
)

type Service struct {
	store entity.Store
}

func NewService(store entity.Store) *Service {
	return &Service{store: store}
}

func (s *Service) Record(ctx context.Context, params RecordParams) error {
	_, err := s.store.CreateAuditLog(ctx, s.toCreateParams(ctx, params))
	return err
}

func (s *Service) RecordTx(ctx context.Context, q *entity.Queries, params RecordParams) error {
	_, err := q.CreateAuditLog(ctx, s.toCreateParams(ctx, params))
	return err
}

func (s *Service) toCreateParams(ctx context.Context, params RecordParams) entity.CreateAuditLogParams {
	metadata := MetadataFromContext(ctx)

	return entity.CreateAuditLogParams{
		RequestID:   nullableString(metadata.RequestID),
		ActorUserID: nullableUUID(params.ActorUserID),
		ActorOrgID:  nullableUUID(params.ActorOrgID),
		Action:      params.Action,
		EntityType:  params.EntityType,
		EntityID:    nullableUUID(params.EntityID),
		BeforeData:  marshalData(params.BeforeData),
		AfterData:   marshalData(params.AfterData),
		IpAddress:   nullableString(metadata.IPAddress),
		UserAgent:   nullableString(metadata.UserAgent),
	}
}

func marshalData(data any) []byte {
	if data == nil {
		return []byte("{}")
	}
	b, err := json.Marshal(data)
	if err != nil {
		return []byte(`{"marshal_error":true}`)
	}
	return b
}

func nullableUUID(id uuid.UUID) uuid.NullUUID {
	return uuid.NullUUID{UUID: id, Valid: id != uuid.Nil}
}

func nullableString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}
