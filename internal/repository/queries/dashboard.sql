-- internal/repository/queries/dashboard.sql
-- SQLC queries for DASHBOARD module (S-Curve, Stats)

-- ===== PROJECT SETTINGS =====

-- name: GetProjectSettingsForDashboard :one
SELECT ps.*, p.name as project_name
FROM project_settings ps
JOIN projects p ON p.id = ps.project_id
WHERE ps.project_id = $1 AND ps.deleted_at IS NULL;

-- ===== S-CURVE DATA =====

-- name: GetTaskItemsWithDeadlines :many
-- Get all leaf task items with their deadlines and weights for plan calculation
SELECT ti.id, ti.title, ti.deadline, ti.weight, ti.parent_id
FROM task_items ti
JOIN contracts c ON c.id = ti.contract_id
WHERE c.project_id = $1 AND ti.deleted_at IS NULL
ORDER BY ti.deadline ASC;

-- name: GetAcceptedProgressData :many
-- Get progress data with validations for actual calculation
SELECT 
    pl.id as log_id,
    pl.task_item_id,
    pl.snapshot_weight,
    pl.created_at as log_date,
    pv.real_progress_percentage,
    pv.updated_at as validation_date
FROM progress_logs pl
JOIN progress_validations pv ON pv.progress_log_id = pl.id
JOIN task_items ti ON ti.id = pl.task_item_id
JOIN contracts c ON c.id = ti.contract_id
WHERE c.project_id = $1 
  AND pv.is_accepted = TRUE 
  AND pl.deleted_at IS NULL
ORDER BY pv.updated_at ASC;

-- ===== STATS DATA =====

-- name: GetDisbursedAmountByProjectID :one
SELECT COALESCE(SUM(pr.amount), 0)::BIGINT as disbursed_amount
FROM payment_requests pr
JOIN contracts c ON c.id = pr.contract_id
WHERE c.project_id = $1 
  AND pr.status = 'PAID' 
  AND pr.deleted_at IS NULL;

-- name: GetTicketStatsByProjectID :one
-- Count tickets by status for a project
SELECT 
    COUNT(*) FILTER (WHERE latest.new_status IN ('OPEN', 'ASSIGNED')) as open_count,
    COUNT(*) FILTER (WHERE latest.new_status = 'IN_PROGRESS') as in_progress_count,
    COUNT(*) FILTER (WHERE latest.new_status = 'RESOLVED') as resolved_count,
    COUNT(*) FILTER (WHERE latest.new_status = 'CLOSED') as closed_count,
    COUNT(*) as total_count
FROM tickets t
JOIN contracts c ON c.id = t.contract_id
LEFT JOIN LATERAL (
    SELECT tl.new_status
    FROM ticket_logs tl 
    WHERE tl.ticket_id = t.id 
    ORDER BY tl.created_at DESC 
    LIMIT 1
) latest ON TRUE
WHERE c.project_id = $1 AND t.deleted_at IS NULL;
