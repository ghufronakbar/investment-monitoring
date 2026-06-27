-- internal/repository/queries/sla.sql
-- SQLC queries for generic SLA tracking

-- name: UpsertSLARecord :one
INSERT INTO sla_records (
    entity_type,
    entity_id,
    state,
    owner_org_id,
    entered_at,
    due_at,
    status
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (entity_type, entity_id) DO UPDATE SET
    state = EXCLUDED.state,
    owner_org_id = EXCLUDED.owner_org_id,
    entered_at = EXCLUDED.entered_at,
    due_at = EXCLUDED.due_at,
    resolved_at = NULL,
    status = EXCLUDED.status,
    deleted_at = NULL
RETURNING *;

-- name: ResolveSLARecord :one
UPDATE sla_records
SET resolved_at = $3,
    status = CASE
        WHEN $3 > due_at THEN 'SLA_VIOLATED'
        ELSE 'RESOLVED'
    END
WHERE entity_type = $1
  AND entity_id = $2
  AND deleted_at IS NULL
RETURNING *;

-- name: GetSLARecordByEntity :one
SELECT * FROM sla_records
WHERE entity_type = $1
  AND entity_id = $2
  AND deleted_at IS NULL;

-- name: TouchSLAStatus :one
UPDATE sla_records
SET status = $3
WHERE entity_type = $1
  AND entity_id = $2
  AND deleted_at IS NULL
RETURNING *;
