-- internal/repository/queries/finance.sql
-- SQLC queries for FINANCE module (Disbursement/Payment Requests)
-- Tiered Approval Flow: SUBMITTED -> AUDITOR -> AGENCY -> PAID

-- ===== PAYMENT REQUESTS QUERIES =====

-- name: CreatePaymentRequest :one
INSERT INTO payment_requests (
    title, amount, status, invoice_file_url, contract_id
)
VALUES ($1, $2, 'SUBMITTED', $3, $4)
RETURNING *;

-- name: GetPaymentRequestByID :one
SELECT 
    pr.*,
    o_auditor.name as auditor_org_name,
    o_agency.name as agency_org_name,
    o_loan.name as loan_provider_org_name
FROM payment_requests pr
LEFT JOIN organizations o_auditor ON o_auditor.id = pr.auditor_org_id
LEFT JOIN organizations o_agency ON o_agency.id = pr.executing_agency_org_id
LEFT JOIN organizations o_loan ON o_loan.id = pr.loan_provider_org_id
WHERE pr.id = $1 AND pr.deleted_at IS NULL;

-- name: GetPaymentRequestsByContractID :many
SELECT 
    pr.*,
    o_auditor.name as auditor_org_name,
    o_agency.name as agency_org_name,
    o_loan.name as loan_provider_org_name
FROM payment_requests pr
LEFT JOIN organizations o_auditor ON o_auditor.id = pr.auditor_org_id
LEFT JOIN organizations o_agency ON o_agency.id = pr.executing_agency_org_id
LEFT JOIN organizations o_loan ON o_loan.id = pr.loan_provider_org_id
WHERE pr.contract_id = $1 AND pr.deleted_at IS NULL
ORDER BY pr.created_at DESC;

-- name: ApproveByAuditor :one
UPDATE payment_requests SET
    status = $2,
    auditor_org_id = $3,
    auditor_user_id = $4,
    auditor_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: ApproveByAgency :one
UPDATE payment_requests SET
    status = $2,
    executing_agency_org_id = $3,
    executing_agency_user_id = $4,
    executing_agency_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DisburseByLoanProvider :one
UPDATE payment_requests SET
    status = 'PAID',
    loan_provider_org_id = $2,
    loan_provider_user_id = $3,
    loan_provider_at = NOW(),
    payment_proof_image_urls = $4
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GetPaymentRequestWithContract :one
-- For access control and participant validation
SELECT pr.*, c.id as contract_id, c.project_id,
       c.auditor_participant_id, c.executing_agency_participant_id,
       c.contract_value
FROM payment_requests pr
JOIN contracts c ON c.id = pr.contract_id
WHERE pr.id = $1 AND pr.deleted_at IS NULL;

-- name: GetPendingPaymentRequestsByContractID :many
-- Get payment requests that are in-progress (not PAID, not REJECTED)
SELECT pr.* FROM payment_requests pr
WHERE pr.contract_id = $1 
  AND pr.deleted_at IS NULL
  AND pr.status NOT IN ('PAID', 'AUDITOR_REJECTED', 'AGENCY_REJECTED');

-- name: GetPaidAmountByContractID :one
-- Sum of PAID payment requests
SELECT COALESCE(SUM(pr.amount), 0)::BIGINT as paid_amount
FROM payment_requests pr
WHERE pr.contract_id = $1 
  AND pr.deleted_at IS NULL
  AND pr.status = 'PAID';

-- name: GetPendingAmountByContractID :one
-- Sum of in-progress payment requests (not PAID, not REJECTED)
SELECT COALESCE(SUM(pr.amount), 0)::BIGINT as pending_amount
FROM payment_requests pr
WHERE pr.contract_id = $1 
  AND pr.deleted_at IS NULL
  AND pr.status NOT IN ('PAID', 'AUDITOR_REJECTED', 'AGENCY_REJECTED');

-- name: HasPendingPaymentRequest :one
-- Check if there's any pending payment request
SELECT EXISTS (
    SELECT 1 FROM payment_requests pr
    WHERE pr.contract_id = $1 
      AND pr.deleted_at IS NULL
      AND pr.status NOT IN ('PAID', 'AUDITOR_REJECTED', 'AGENCY_REJECTED')
) as has_pending;

-- name: GetContractWithProjectSettings :one
-- Get contract with project settings for currency
SELECT c.*, p.currency FROM contracts c
JOIN project_settings p ON p.project_id = c.project_id
WHERE c.id = $1;
