-- internal/repository/queries/file.sql
-- SQLC queries for uploaded file metadata

-- name: CreateFileMetadata :one
INSERT INTO files (
    public_id,
    url,
    file_name,
    mime_type,
    size,
    checksum,
    uploaded_by_user_id,
    uploaded_by_org_id,
    entity_type,
    entity_id
)
VALUES (
    sqlc.arg(public_id),
    sqlc.arg(url),
    sqlc.arg(file_name),
    sqlc.arg(mime_type),
    sqlc.arg(size),
    sqlc.arg(checksum),
    sqlc.arg(uploaded_by_user_id),
    sqlc.narg(uploaded_by_org_id),
    sqlc.narg(entity_type),
    sqlc.narg(entity_id)
)
RETURNING *;

-- name: GetFileByID :one
SELECT * FROM files
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListFilesByEntity :many
SELECT * FROM files
WHERE entity_type = $1
  AND entity_id = $2
  AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListFilesByUploader :many
SELECT * FROM files
WHERE uploaded_by_user_id = $1
  AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
