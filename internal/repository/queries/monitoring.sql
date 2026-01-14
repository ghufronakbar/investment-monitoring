-- internal/repository/queries/monitoring.sql
-- SQLC queries for MONITORING module (WBS & Progress)

-- ===== TASK ITEMS (WBS) QUERIES =====

-- name: CreateTask :one
INSERT INTO task_items (title, description, deadline, weight, parent_id, contract_id)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: GetTaskByID :one
SELECT * FROM task_items WHERE id = $1 AND deleted_at IS NULL;

-- name: GetTasksByContractID :many
SELECT * FROM task_items 
WHERE contract_id = $1 AND deleted_at IS NULL 
ORDER BY created_at;

-- name: GetRootTasksByContractID :many
SELECT * FROM task_items 
WHERE contract_id = $1 AND parent_id IS NULL AND deleted_at IS NULL 
ORDER BY created_at;

-- name: GetChildTasks :many
SELECT * FROM task_items 
WHERE parent_id = $1 AND deleted_at IS NULL 
ORDER BY created_at;

-- name: GetLeafTasksByContractID :many
-- Leaf tasks are tasks that don't have any children
SELECT t.* FROM task_items t
WHERE t.contract_id = $1 
  AND t.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM task_items child 
    WHERE child.parent_id = t.id AND child.deleted_at IS NULL
  )
ORDER BY t.created_at;

-- name: GetTotalRootWeight :one
-- Sum of all root task weights for a contract
SELECT COALESCE(SUM(weight), 0)::INT as total_weight
FROM task_items 
WHERE contract_id = $1 AND parent_id IS NULL AND deleted_at IS NULL;

-- name: UpdateTask :one
UPDATE task_items 
SET title = $2, description = $3, deadline = $4, weight = $5, parent_id = $6
WHERE id = $1 AND deleted_at IS NULL 
RETURNING *;

-- name: DeleteTask :exec
UPDATE task_items SET deleted_at = NOW() WHERE id = $1;

-- name: GetTaskWithParent :one
SELECT 
    t.*,
    p.deadline as parent_deadline
FROM task_items t
LEFT JOIN task_items p ON p.id = t.parent_id
WHERE t.id = $1 AND t.deleted_at IS NULL;

-- ===== PROGRESS LOGS QUERIES =====

-- name: CreateProgressLog :one
-- CRITICAL: Use INSERT...SELECT to atomically snapshot task data
INSERT INTO progress_logs (
    report, 
    progress_percentage, 
    evidence_image_urls,
    snapshot_title,
    snapshot_description,
    snapshot_weight,
    task_item_id,
    vendor_id
)
SELECT 
    $1,          -- report
    $2,          -- progress_percentage
    $3,          -- evidence_image_urls
    t.title,     -- snapshot from task_items
    t.description,
    t.weight,
    t.id,        -- task_item_id
    $4           -- vendor_id
FROM task_items t
WHERE t.id = $5 AND t.deleted_at IS NULL
RETURNING *;

-- name: GetProgressLogByID :one
SELECT * FROM progress_logs WHERE id = $1 AND deleted_at IS NULL;

-- name: GetProgressLogsByTaskID :many
SELECT pl.*, pv.id as validation_id, pv.is_accepted, pv.real_progress_percentage, pv.comment
FROM progress_logs pl
LEFT JOIN progress_validations pv ON pv.progress_log_id = pl.id AND pv.deleted_at IS NULL
WHERE pl.task_item_id = $1 AND pl.deleted_at IS NULL
ORDER BY pl.created_at DESC;

-- name: GetPendingValidationLogs :many
-- Logs that don't have a validation entry yet
SELECT pl.*, t.contract_id
FROM progress_logs pl
JOIN task_items t ON t.id = pl.task_item_id
LEFT JOIN progress_validations pv ON pv.progress_log_id = pl.id AND pv.deleted_at IS NULL
WHERE t.contract_id = $1 
  AND pl.deleted_at IS NULL 
  AND pv.id IS NULL
ORDER BY pl.created_at;

-- name: GetProgressLogWithDetails :one
SELECT 
    pl.*,
    t.contract_id,
    o.name as vendor_name,
    pv.id as validation_id,
    pv.is_accepted,
    pv.real_progress_percentage,
    pv.comment as validation_comment,
    pv.auditor_id
FROM progress_logs pl
JOIN task_items t ON t.id = pl.task_item_id
JOIN organizations o ON o.id = pl.vendor_id
LEFT JOIN progress_validations pv ON pv.progress_log_id = pl.id AND pv.deleted_at IS NULL
WHERE pl.id = $1 AND pl.deleted_at IS NULL;

-- ===== PROGRESS VALIDATIONS QUERIES =====

-- name: CreateProgressValidation :one
INSERT INTO progress_validations (is_accepted, comment, real_progress_percentage, progress_log_id, auditor_id)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: GetValidationByProgressLogID :one
SELECT * FROM progress_validations WHERE progress_log_id = $1 AND deleted_at IS NULL;

-- name: CheckProgressLogValidated :one
SELECT EXISTS(
    SELECT 1 FROM progress_validations 
    WHERE progress_log_id = $1 AND deleted_at IS NULL
) as is_validated;

-- name: CheckTaskHasPendingLog :one
-- Check if task has any pending progress log (not yet validated by auditor)
SELECT EXISTS(
    SELECT 1 FROM progress_logs pl
    LEFT JOIN progress_validations pv ON pv.progress_log_id = pl.id AND pv.deleted_at IS NULL
    WHERE pl.task_item_id = $1 AND pl.deleted_at IS NULL AND pv.id IS NULL
) as has_pending;

-- ===== S-CURVE DATA QUERIES =====


-- name: GetSCurveData :one
-- Aggregate actual progress based on validated logs
SELECT 
    COALESCE(SUM(
        CASE WHEN pv.is_accepted = true 
        THEN (pl.snapshot_weight::DECIMAL * pv.real_progress_percentage::DECIMAL / 100) 
        ELSE 0 END
    ), 0)::DECIMAL as actual_progress,
    COALESCE(SUM(pl.snapshot_weight), 0)::INT as total_snapshot_weight,
    COUNT(DISTINCT pl.id)::INT as total_logs,
    COUNT(DISTINCT CASE WHEN pv.is_accepted = true THEN pv.id END)::INT as approved_logs,
    COUNT(DISTINCT CASE WHEN pv.is_accepted = false THEN pv.id END)::INT as rejected_logs
FROM progress_logs pl
JOIN task_items t ON t.id = pl.task_item_id
LEFT JOIN progress_validations pv ON pv.progress_log_id = pl.id AND pv.deleted_at IS NULL
WHERE t.contract_id = $1 AND pl.deleted_at IS NULL;

-- name: GetSCurveDataByTask :many
-- Per-task progress for detailed S-Curve
SELECT 
    t.id as task_id,
    t.title as task_title,
    t.weight as planned_weight,
    t.deadline,
    COALESCE(MAX(CASE WHEN pv.is_accepted = true THEN pv.real_progress_percentage ELSE 0 END), 0)::INT as actual_percentage,
    COALESCE(MAX(pl.progress_percentage), 0)::INT as claimed_percentage
FROM task_items t
LEFT JOIN progress_logs pl ON pl.task_item_id = t.id AND pl.deleted_at IS NULL
LEFT JOIN progress_validations pv ON pv.progress_log_id = pl.id AND pv.deleted_at IS NULL AND pv.is_accepted = true
WHERE t.contract_id = $1 AND t.deleted_at IS NULL
GROUP BY t.id, t.title, t.weight, t.deadline
ORDER BY t.deadline;

-- name: GetAllProgressLogsByContractID :many
SELECT 
    pl.*,
    t.title as task_title,
    o.name as vendor_name,
    pv.id as validation_id,
    pv.is_accepted,
    pv.real_progress_percentage,
    pv.comment as validation_comment,
    pv.auditor_id,
    pv.created_at as validation_created_at
FROM progress_logs pl
JOIN task_items t ON t.id = pl.task_item_id
JOIN organizations o ON o.id = pl.vendor_id
LEFT JOIN progress_validations pv ON pv.progress_log_id = pl.id AND pv.deleted_at IS NULL
WHERE t.contract_id = $1 AND pl.deleted_at IS NULL
ORDER BY pl.created_at DESC;
