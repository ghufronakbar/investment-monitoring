-- internal/repository/queries/audit.sql
-- SQLC queries for global audit logging

-- name: CreateAuditLog :one
INSERT INTO audit_logs (
    request_id,
    actor_user_id,
    actor_org_id,
    action,
    entity_type,
    entity_id,
    before_data,
    after_data,
    ip_address,
    user_agent
)
VALUES (
    sqlc.narg(request_id),
    sqlc.narg(actor_user_id),
    sqlc.narg(actor_org_id),
    sqlc.arg(action),
    sqlc.arg(entity_type),
    sqlc.narg(entity_id),
    sqlc.arg(before_data)::jsonb,
    sqlc.arg(after_data)::jsonb,
    sqlc.narg(ip_address),
    sqlc.narg(user_agent)
)
RETURNING *;

-- name: ListAuditLogsByEntity :many
SELECT * FROM audit_logs
WHERE entity_type = $1
  AND entity_id = $2
ORDER BY created_at DESC;

-- name: ListAuditLogsByActor :many
SELECT * FROM audit_logs
WHERE actor_user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
