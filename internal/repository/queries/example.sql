-- name: GetListExample :many
SELECT
    id,
    name,
    price,
    created_at,
    updated_at,
    deleted_at
FROM examples
WHERE
    deleted_at IS NULL
    AND (
        -- Trik agar search TIDAK jadi pointer:
        -- Cek apakah string kosong ('') ATAU match pattern.
        -- sqlc akan membaca ini sebagai string wajib (bukan *string).
        sqlc.arg(search)::text = ''
        OR name ILIKE '%' || sqlc.arg(search)::text || '%'
    )
ORDER BY
    -- sortBy = name
    CASE
        WHEN sqlc.arg(sort_by)::text = 'name' AND sqlc.arg(sort_dir)::text = 'asc'
        THEN name
    END ASC,
    CASE
        WHEN sqlc.arg(sort_by)::text = 'name' AND sqlc.arg(sort_dir)::text = 'desc'
        THEN name
    END DESC,

    -- sortBy = price
    CASE
        WHEN sqlc.arg(sort_by)::text = 'price' AND sqlc.arg(sort_dir)::text = 'asc'
        THEN price
    END ASC,
    CASE
        WHEN sqlc.arg(sort_by)::text = 'price' AND sqlc.arg(sort_dir)::text = 'desc'
        THEN price
    END DESC,

    -- sortBy = created_at
    CASE
        WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_dir)::text = 'asc'
        THEN created_at
    END ASC,
    CASE
        WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_dir)::text = 'desc'
        THEN created_at
    END DESC,

    -- sortBy = updated_at
    CASE
        WHEN sqlc.arg(sort_by)::text = 'updated_at' AND sqlc.arg(sort_dir)::text = 'asc'
        THEN updated_at
    END ASC,
    CASE
        WHEN sqlc.arg(sort_by)::text = 'updated_at' AND sqlc.arg(sort_dir)::text = 'desc'
        THEN updated_at
    END DESC,

    -- deterministic fallback
    created_at DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: CountExample :one
SELECT COUNT(*)
FROM examples
WHERE
    deleted_at IS NULL
    AND (
        sqlc.arg(search)::text = ''
        OR name ILIKE '%' || sqlc.arg(search)::text || '%'
    );

-- name: GetExampleByID :one
SELECT * from examples where id = sqlc.arg('id') AND deleted_at IS NULL;

-- name: CreateExample :one
INSERT INTO examples (
    name,
    price
) VALUES (
    sqlc.arg('name'),
    sqlc.arg('price')
)
RETURNING *;

-- name: UpdateExampleByID :one
UPDATE examples
SET name = sqlc.arg('name'),
    price = sqlc.arg('price')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING * ;

-- name: SoftDeleteExampleByID :one
UPDATE examples
SET deleted_at = now()
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING * ;