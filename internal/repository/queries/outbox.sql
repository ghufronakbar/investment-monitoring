-- internal/repository/queries/outbox.sql
-- SQLC queries for transactional domain events / outbox pattern

-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (
    event_type,
    aggregate_type,
    aggregate_id,
    payload,
    status
)
VALUES (
    sqlc.arg(event_type),
    sqlc.arg(aggregate_type),
    sqlc.arg(aggregate_id),
    sqlc.arg(payload)::jsonb,
    'PENDING'
)
RETURNING *;

-- name: ListPendingOutboxEvents :many
SELECT * FROM outbox_events
WHERE status = 'PENDING'
ORDER BY created_at
LIMIT $1;

-- name: MarkOutboxEventProcessing :one
UPDATE outbox_events
SET status = 'PROCESSING'
WHERE id = $1
  AND status = 'PENDING'
RETURNING *;

-- name: MarkOutboxEventProcessed :one
UPDATE outbox_events
SET status = 'PROCESSED',
    processed_at = NOW()
WHERE id = $1
RETURNING *;

-- name: MarkOutboxEventFailed :one
UPDATE outbox_events
SET status = 'FAILED',
    processed_at = NOW()
WHERE id = $1
RETURNING *;
