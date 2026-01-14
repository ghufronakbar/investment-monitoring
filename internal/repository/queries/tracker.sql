-- internal/repository/queries/tracker.sql
-- SQLC queries for TRACKER module (Ticketing System)
-- Uses Event-Sourced Status: current status = latest log entry

-- ===== TICKETS QUERIES =====

-- name: CreateTicket :one
INSERT INTO tickets (
    title, description, location, evidence_image_urls, 
    task_item_id, contract_id,
    ticket_assigner_contractor_id, ticket_assigner_user_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetTicketByID :one
SELECT 
    t.*,
    o_creator.name as creator_org_name,
    o_assignee.name as assignee_org_name,
    ti.title as task_title,
    (
        SELECT new_status FROM ticket_logs 
        WHERE ticket_id = t.id AND deleted_at IS NULL 
        ORDER BY created_at DESC LIMIT 1
    ) as current_status
FROM tickets t
JOIN organizations o_creator ON o_creator.id = t.ticket_assigner_contractor_id
LEFT JOIN organizations o_assignee ON o_assignee.id = t.ticket_assign_to_id
LEFT JOIN task_items ti ON ti.id = t.task_item_id
WHERE t.id = $1 AND t.deleted_at IS NULL;

-- name: GetTicketsByContractID :many
-- Uses DISTINCT ON to avoid N+1 query for current_status
SELECT 
    t.*,
    o_creator.name as creator_org_name,
    o_assignee.name as assignee_org_name,
    ti.title as task_title,
    latest_log.new_status as current_status,
    latest_log.created_at as last_log_at
FROM tickets t
JOIN organizations o_creator ON o_creator.id = t.ticket_assigner_contractor_id
LEFT JOIN organizations o_assignee ON o_assignee.id = t.ticket_assign_to_id
LEFT JOIN task_items ti ON ti.id = t.task_item_id
LEFT JOIN LATERAL (
    SELECT new_status, created_at 
    FROM ticket_logs 
    WHERE ticket_id = t.id AND deleted_at IS NULL 
    ORDER BY created_at DESC 
    LIMIT 1
) latest_log ON true
WHERE t.contract_id = $1 AND t.deleted_at IS NULL
ORDER BY t.created_at DESC;

-- name: GetOpenTicketsByContractID :many
-- Filter non-CLOSED tickets
SELECT 
    t.*,
    o_creator.name as creator_org_name,
    o_assignee.name as assignee_org_name,
    ti.title as task_title,
    latest_log.new_status as current_status,
    latest_log.created_at as last_log_at
FROM tickets t
JOIN organizations o_creator ON o_creator.id = t.ticket_assigner_contractor_id
LEFT JOIN organizations o_assignee ON o_assignee.id = t.ticket_assign_to_id
LEFT JOIN task_items ti ON ti.id = t.task_item_id
LEFT JOIN LATERAL (
    SELECT new_status, created_at 
    FROM ticket_logs 
    WHERE ticket_id = t.id AND deleted_at IS NULL 
    ORDER BY created_at DESC 
    LIMIT 1
) latest_log ON true
WHERE t.contract_id = $1 
  AND t.deleted_at IS NULL 
  AND latest_log.new_status != 'CLOSED'
ORDER BY t.created_at DESC;

-- name: GetClosedTicketsByContractID :many
-- Filter CLOSED tickets only
SELECT 
    t.*,
    o_creator.name as creator_org_name,
    o_assignee.name as assignee_org_name,
    ti.title as task_title,
    latest_log.new_status as current_status,
    latest_log.created_at as last_log_at
FROM tickets t
JOIN organizations o_creator ON o_creator.id = t.ticket_assigner_contractor_id
LEFT JOIN organizations o_assignee ON o_assignee.id = t.ticket_assign_to_id
LEFT JOIN task_items ti ON ti.id = t.task_item_id
LEFT JOIN LATERAL (
    SELECT new_status, created_at 
    FROM ticket_logs 
    WHERE ticket_id = t.id AND deleted_at IS NULL 
    ORDER BY created_at DESC 
    LIMIT 1
) latest_log ON true
WHERE t.contract_id = $1 
  AND t.deleted_at IS NULL 
  AND latest_log.new_status = 'CLOSED'
ORDER BY t.created_at DESC;

-- name: AssignTicket :one
UPDATE tickets SET ticket_assign_to_id = $2
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- ===== TICKET LOGS QUERIES =====

-- name: CreateTicketLog :one
INSERT INTO ticket_logs (
    action, description, attachment_url,
    previous_status, new_status,
    ticket_id, organization_id, user_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetTicketLogsByTicketID :many
SELECT 
    tl.*,
    o.name as org_name,
    u.name as user_name
FROM ticket_logs tl
JOIN organizations o ON o.id = tl.organization_id
JOIN users u ON u.id = tl.user_id
WHERE tl.ticket_id = $1 AND tl.deleted_at IS NULL
ORDER BY tl.created_at DESC;

-- name: GetLatestTicketStatus :one
SELECT new_status FROM ticket_logs
WHERE ticket_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: GetTicketWithContract :one
-- For access control - get ticket with contract info
SELECT t.*, c.id as contract_id
FROM tickets t
JOIN contracts c ON c.id = t.contract_id
WHERE t.id = $1 AND t.deleted_at IS NULL;
