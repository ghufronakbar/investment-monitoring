-- internal/repository/queries/project.sql
-- SQLC queries for PROJECT module

-- ===== PROJECT QUERIES =====

-- name: CreateProject :one
INSERT INTO projects (name, description, contract_file_url, image_urls)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetProjectByID :one
SELECT * FROM projects WHERE id = $1 AND deleted_at IS NULL;

-- name: ListProjects :many
SELECT * FROM projects WHERE deleted_at IS NULL ORDER BY created_at DESC;

-- name: UpdateProject :one
UPDATE projects SET name = $2, description = $3, contract_file_url = $4, image_urls = $5
WHERE id = $1 AND deleted_at IS NULL RETURNING *;

-- ===== CONTRACT QUERIES =====

-- name: CreateContract :one
INSERT INTO contracts (description, project_id, gov_supervisor_participant_id, loan_provider_participant_id)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetContractByID :one
SELECT * FROM contracts WHERE id = $1 AND deleted_at IS NULL;

-- name: GetContractByProjectID :one
SELECT * FROM contracts WHERE project_id = $1 AND deleted_at IS NULL;

-- name: GetContractWithDetails :one
SELECT 
    c.*,
    p.name as project_name,
    gs.name as gov_supervisor_name,
    lp.name as loan_provider_name,
    ea.name as executing_agency_name,
    v.name as vendor_name,
    a.name as auditor_name
FROM contracts c
JOIN projects p ON p.id = c.project_id
JOIN organizations gs ON gs.id = c.gov_supervisor_participant_id
JOIN organizations lp ON lp.id = c.loan_provider_participant_id
LEFT JOIN organizations ea ON ea.id = c.executing_agency_participant_id
LEFT JOIN organizations v ON v.id = c.vendor_participant_id
LEFT JOIN organizations a ON a.id = c.auditor_participant_id
WHERE c.project_id = $1 AND c.deleted_at IS NULL;

-- name: AssignExecutingAgency :one
UPDATE contracts SET executing_agency_participant_id = $2 
WHERE id = $1 AND deleted_at IS NULL RETURNING *;

-- name: AssignVendorToContract :one
UPDATE contracts SET 
    vendor_participant_id = $2, 
    contract_value = $3, 
    contract_doc_url = $4, 
    vendor_participant_selected_at = NOW()
WHERE id = $1 AND deleted_at IS NULL RETURNING *;

-- name: AssignAuditor :one
UPDATE contracts SET 
    auditor_participant_id = $2, 
    auditor_participant_selected_at = NOW()
WHERE id = $1 AND deleted_at IS NULL RETURNING *;

-- ===== PROJECT SETTINGS QUERIES =====

-- name: CreateProjectSettings :one
INSERT INTO project_settings (project_id, start_date, baseline_deadline, total_budget, currency)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: GetProjectSettingsByProjectID :one
SELECT * FROM project_settings WHERE project_id = $1 AND deleted_at IS NULL;

-- name: UpdateProjectSettings :one
UPDATE project_settings 
SET start_date = $2, baseline_deadline = $3, total_budget = $4, currency = $5
WHERE project_id = $1 AND deleted_at IS NULL RETURNING *;

-- name: UpsertProjectSettings :one
INSERT INTO project_settings (project_id, start_date, baseline_deadline, total_budget, currency)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (project_id) DO UPDATE SET 
    start_date = EXCLUDED.start_date,
    baseline_deadline = EXCLUDED.baseline_deadline,
    total_budget = EXCLUDED.total_budget,
    currency = EXCLUDED.currency
RETURNING *;

-- ===== GOV PARTICIPANTS QUERIES =====

-- name: AddGovParticipant :one
INSERT INTO gov_participants (contract_id, gov_participant_id) 
VALUES ($1, $2) RETURNING *;

-- name: GetGovParticipantsByContractID :many
SELECT 
    gp.id,
    gp.contract_id,
    gp.gov_participant_id,
    gp.created_at,
    o.name as org_name,
    o.logo_url as org_logo_url
FROM gov_participants gp
JOIN organizations o ON o.id = gp.gov_participant_id
WHERE gp.contract_id = $1 AND gp.deleted_at IS NULL;

-- name: DeleteGovParticipantsByContractID :exec
UPDATE gov_participants SET deleted_at = NOW() WHERE contract_id = $1 AND deleted_at IS NULL;

-- name: CheckGovParticipantExists :one
SELECT EXISTS(
    SELECT 1 FROM gov_participants 
    WHERE contract_id = $1 AND gov_participant_id = $2 AND deleted_at IS NULL
) as exists;

-- ===== PROJECT ACCESS HELPERS =====

-- name: GetUserProjectAccess :one
-- Check if user's organization has access to a project via contract
SELECT 
    c.id as contract_id,
    c.project_id,
    CASE 
        WHEN c.gov_supervisor_participant_id = om.organization_id THEN 'GOV_SUPERVISOR'
        WHEN c.loan_provider_participant_id = om.organization_id THEN 'LOAN_PROVIDER'
        WHEN c.executing_agency_participant_id = om.organization_id THEN 'EXECUTING_AGENCY'
        WHEN c.vendor_participant_id = om.organization_id THEN 'CONTRACTOR'
        WHEN c.auditor_participant_id = om.organization_id THEN 'INDEPENDENT_AUDITOR'
        ELSE NULL
    END as participant_role,
    om.role as member_role,
    om.organization_id,
    o.name as org_name,
    o.role as org_type
FROM contracts c
JOIN organization_members om ON (
    om.organization_id = c.gov_supervisor_participant_id OR
    om.organization_id = c.loan_provider_participant_id OR
    om.organization_id = c.executing_agency_participant_id OR
    om.organization_id = c.vendor_participant_id OR
    om.organization_id = c.auditor_participant_id
)
JOIN organizations o ON o.id = om.organization_id
WHERE c.project_id = $1 
    AND om.user_id = $2 
    AND c.deleted_at IS NULL 
    AND om.deleted_at IS NULL
LIMIT 1;

-- name: ListUserAccessibleProjects :many
-- Get all projects where user's organization is a participant
SELECT DISTINCT p.* FROM projects p
JOIN contracts c ON c.project_id = p.id
JOIN organization_members om ON (
    om.organization_id = c.gov_supervisor_participant_id OR
    om.organization_id = c.loan_provider_participant_id OR
    om.organization_id = c.executing_agency_participant_id OR
    om.organization_id = c.vendor_participant_id OR
    om.organization_id = c.auditor_participant_id
)
LEFT JOIN gov_participants gp ON gp.contract_id = c.id AND gp.gov_participant_id = om.organization_id
WHERE om.user_id = $1 
    AND p.deleted_at IS NULL 
    AND c.deleted_at IS NULL 
    AND om.deleted_at IS NULL
ORDER BY p.created_at DESC;
