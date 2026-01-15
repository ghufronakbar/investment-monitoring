// internal/module/finance/service/service.go
package finance_service

import (
	"context"
	"database/sql"
	"strconv"

	"investment_monitoring/config"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"

	"github.com/google/uuid"
)

type FinanceService struct {
	store *entity.SQLStore
	cfg   *config.Config
}

func NewService(store *entity.SQLStore, cfg *config.Config) *FinanceService {
	return &FinanceService{
		store: store,
		cfg:   cfg,
	}
}

// ===== PAYMENT REQUESTS =====

// CreatePaymentRequest creates a new payment request with SUBMITTED status
func (s *FinanceService) CreatePaymentRequest(ctx context.Context, params CreatePaymentRequestParams) (PaymentRequestReturn, error) {
	var result PaymentRequestReturn

	// Lookup ContractID from ProjectID
	contract, err := s.store.GetContractByProjectID(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Verify user's org is the contractor
	if contract.VendorParticipantID.Valid && contract.VendorParticipantID.UUID != params.ContractorID {
		return result, errs.NewForbidden("NOT_CONTRACT_VENDOR")
	}

	// Check contract_value is set
	if !contract.ContractValue.Valid || contract.ContractValue.Int64 == 0 {
		return result, errs.NewBadRequest("CONTRACT_VALUE_NOT_SET")
	}

	// Validation: Check if there's any pending payment request
	hasPending, err := s.store.HasPendingPaymentRequest(ctx, contract.ID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}
	if hasPending {
		return result, errs.NewBadRequest("PENDING_REQUEST_EXISTS")
	}

	// Calculate available amount
	paidAmount, err := s.store.GetPaidAmountByContractID(ctx, contract.ID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}
	pendingAmount, err := s.store.GetPendingAmountByContractID(ctx, contract.ID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	availableAmount := contract.ContractValue.Int64 - paidAmount - pendingAmount

	// Validation: Amount cannot exceed contract_value
	if params.Amount > contract.ContractValue.Int64 {
		return result, errs.NewBadRequest("AMOUNT_EXCEEDS_CONTRACT_VALUE")
	}

	// Validation: Amount cannot exceed available amount
	if params.Amount > availableAmount {
		return result, errs.NewBadRequest("AMOUNT_EXCEEDS_AVAILABLE")
	}

	// Create payment request
	pr, err := s.store.CreatePaymentRequest(ctx, entity.CreatePaymentRequestParams{
		Title:          params.Title,
		Amount:         params.Amount,
		InvoiceFileUrl: params.InvoiceFileUrl,
		ContractID:     contract.ID,
	})
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	result = PaymentRequestReturn{
		ID:                    pr.ID,
		Title:                 pr.Title,
		Amount:                strconv.FormatInt(pr.Amount, 10),
		Status:                string(pr.Status),
		InvoiceFileUrl:        pr.InvoiceFileUrl,
		PaymentProofImageUrls: ensureStringArray(pr.PaymentProofImageUrls),
		ContractID:            pr.ContractID,
		CreatedAt:             pr.CreatedAt,
		UpdatedAt:             pr.UpdatedAt,
	}

	return result, nil
}

// GetPaymentRequests retrieves all payment requests for a project
func (s *FinanceService) GetPaymentRequests(ctx context.Context, params GetPaymentRequestsParams) ([]PaymentRequestReturn, error) {
	var result []PaymentRequestReturn

	// Lookup ContractID from ProjectID
	contract, err := s.store.GetContractByProjectID(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	requests, err := s.store.GetPaymentRequestsByContractID(ctx, contract.ID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	result = make([]PaymentRequestReturn, 0, len(requests))
	for _, pr := range requests {
		item := mapPaymentRequestRowToReturn(pr)
		result = append(result, item)
	}

	return result, nil
}

// GetPaymentRequestByID retrieves a single payment request
func (s *FinanceService) GetPaymentRequestByID(ctx context.Context, params GetPaymentRequestByIDParams) (PaymentRequestReturn, error) {
	var result PaymentRequestReturn

	pr, err := s.store.GetPaymentRequestByID(ctx, params.RequestID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("PAYMENT_REQUEST_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	result = PaymentRequestReturn{
		ID:                    pr.ID,
		Title:                 pr.Title,
		Amount:                strconv.FormatInt(pr.Amount, 10),
		Status:                string(pr.Status),
		InvoiceFileUrl:        pr.InvoiceFileUrl,
		PaymentProofImageUrls: ensureStringArray(pr.PaymentProofImageUrls),
		ContractID:            pr.ContractID,
		CreatedAt:             pr.CreatedAt,
		UpdatedAt:             pr.UpdatedAt,
	}

	// Add approval signatures
	if pr.AuditorOrgID.Valid && pr.AuditorAt.Valid {
		result.Auditor = &ApprovalSignature{
			OrgID:      pr.AuditorOrgID.UUID,
			OrgName:    pr.AuditorOrgName.String,
			UserID:     pr.AuditorUserID.UUID,
			ApprovedAt: pr.AuditorAt.Time,
		}
	}
	if pr.ExecutingAgencyOrgID.Valid && pr.ExecutingAgencyAt.Valid {
		result.Agency = &ApprovalSignature{
			OrgID:      pr.ExecutingAgencyOrgID.UUID,
			OrgName:    pr.AgencyOrgName.String,
			UserID:     pr.ExecutingAgencyUserID.UUID,
			ApprovedAt: pr.ExecutingAgencyAt.Time,
		}
	}
	if pr.LoanProviderOrgID.Valid && pr.LoanProviderAt.Valid {
		result.LoanProvider = &ApprovalSignature{
			OrgID:      pr.LoanProviderOrgID.UUID,
			OrgName:    pr.LoanProviderOrgName.String,
			UserID:     pr.LoanProviderUserID.UUID,
			ApprovedAt: pr.LoanProviderAt.Time,
		}
	}

	return result, nil
}

// GetPaymentSummary returns payment summary with available amount for reimbursement
func (s *FinanceService) GetPaymentSummary(ctx context.Context, params GetPaymentSummaryParams) (PaymentSummaryReturn, error) {
	var result PaymentSummaryReturn

	// Lookup ContractID from ProjectID
	contract, err := s.store.GetContractByProjectID(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Get contract with project settings for currency
	contractWithSettings, err := s.store.GetContractWithProjectSettings(ctx, contract.ID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Get paid and pending amounts
	paidAmount, err := s.store.GetPaidAmountByContractID(ctx, contract.ID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}
	pendingAmount, err := s.store.GetPendingAmountByContractID(ctx, contract.ID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Check if can submit
	hasPending, err := s.store.HasPendingPaymentRequest(ctx, contract.ID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	contractValue := int64(0)
	if contractWithSettings.ContractValue.Valid {
		contractValue = contractWithSettings.ContractValue.Int64
	}
	availableAmount := contractValue - paidAmount - pendingAmount
	if availableAmount < 0 {
		availableAmount = 0
	}

	result = PaymentSummaryReturn{
		ContractValue:   strconv.FormatInt(contractValue, 10),
		PaidAmount:      strconv.FormatInt(paidAmount, 10),
		PendingAmount:   strconv.FormatInt(pendingAmount, 10),
		AvailableAmount: strconv.FormatInt(availableAmount, 10),
		CanSubmit:       !hasPending && availableAmount > 0,
		Currency:        contractWithSettings.Currency,
	}

	return result, nil
}

// ApproveByAuditor handles auditor approval/rejection
func (s *FinanceService) ApproveByAuditor(ctx context.Context, params ApprovalParams) (PaymentRequestReturn, error) {
	// Get request with contract for validation
	prWithContract, err := s.store.GetPaymentRequestWithContract(ctx, params.RequestID)
	if err == sql.ErrNoRows {
		return PaymentRequestReturn{}, errs.NewNotFound("PAYMENT_REQUEST_NOT_FOUND")
	}
	if err != nil {
		return PaymentRequestReturn{}, errs.NewInternalServerError(err)
	}

	// Validate: org must be the auditor_participant_id in contract
	if !prWithContract.AuditorParticipantID.Valid || prWithContract.AuditorParticipantID.UUID != params.OrgID {
		return PaymentRequestReturn{}, errs.NewForbidden("NOT_CONTRACT_AUDITOR")
	}

	// Validate: status must be SUBMITTED
	if prWithContract.Status != entity.PaymentRequestStatusEnumSUBMITTED {
		return PaymentRequestReturn{}, errs.NewBadRequest("INVALID_STATUS_TRANSITION")
	}

	// Determine new status
	var newStatus entity.PaymentRequestStatusEnum
	if params.Action == "APPROVE" {
		newStatus = entity.PaymentRequestStatusEnumAUDITORAPPROVED
	} else {
		newStatus = entity.PaymentRequestStatusEnumAUDITORREJECTED
	}

	// Update
	_, err = s.store.ApproveByAuditor(ctx, entity.ApproveByAuditorParams{
		ID:            params.RequestID,
		Status:        newStatus,
		AuditorOrgID:  uuid.NullUUID{UUID: params.OrgID, Valid: true},
		AuditorUserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
	})
	if err != nil {
		return PaymentRequestReturn{}, errs.NewInternalServerError(err)
	}

	// Return updated request
	return s.GetPaymentRequestByID(ctx, GetPaymentRequestByIDParams{RequestID: params.RequestID})
}

// ApproveByAgency handles agency approval/rejection
func (s *FinanceService) ApproveByAgency(ctx context.Context, params ApprovalParams) (PaymentRequestReturn, error) {
	// Get request with contract for validation
	prWithContract, err := s.store.GetPaymentRequestWithContract(ctx, params.RequestID)
	if err == sql.ErrNoRows {
		return PaymentRequestReturn{}, errs.NewNotFound("PAYMENT_REQUEST_NOT_FOUND")
	}
	if err != nil {
		return PaymentRequestReturn{}, errs.NewInternalServerError(err)
	}

	// Validate: org must be the executing_agency_participant_id in contract
	if !prWithContract.ExecutingAgencyParticipantID.Valid || prWithContract.ExecutingAgencyParticipantID.UUID != params.OrgID {
		return PaymentRequestReturn{}, errs.NewForbidden("NOT_CONTRACT_EXECUTING_AGENCY")
	}

	// Validate: status must be AUDITOR_APPROVED
	if prWithContract.Status != entity.PaymentRequestStatusEnumAUDITORAPPROVED {
		return PaymentRequestReturn{}, errs.NewBadRequest("INVALID_STATUS_TRANSITION")
	}

	// Determine new status
	var newStatus entity.PaymentRequestStatusEnum
	if params.Action == "APPROVE" {
		newStatus = entity.PaymentRequestStatusEnumAGENCYAPPROVED
	} else {
		newStatus = entity.PaymentRequestStatusEnumAGENCYREJECTED
	}

	// Update
	_, err = s.store.ApproveByAgency(ctx, entity.ApproveByAgencyParams{
		ID:                    params.RequestID,
		Status:                newStatus,
		ExecutingAgencyOrgID:  uuid.NullUUID{UUID: params.OrgID, Valid: true},
		ExecutingAgencyUserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
	})
	if err != nil {
		return PaymentRequestReturn{}, errs.NewInternalServerError(err)
	}

	// Return updated request
	return s.GetPaymentRequestByID(ctx, GetPaymentRequestByIDParams{RequestID: params.RequestID})
}

// Disburse handles loan provider disbursement (marking as paid)
func (s *FinanceService) Disburse(ctx context.Context, params DisburseParams) (PaymentRequestReturn, error) {
	// Get request with contract for validation
	prWithContract, err := s.store.GetPaymentRequestWithContract(ctx, params.RequestID)
	if err == sql.ErrNoRows {
		return PaymentRequestReturn{}, errs.NewNotFound("PAYMENT_REQUEST_NOT_FOUND")
	}
	if err != nil {
		return PaymentRequestReturn{}, errs.NewInternalServerError(err)
	}

	// Validate: status must be AGENCY_APPROVED
	if prWithContract.Status != entity.PaymentRequestStatusEnumAGENCYAPPROVED {
		return PaymentRequestReturn{}, errs.NewBadRequest("INVALID_STATUS_TRANSITION")
	}

	// Update with payment proof
	_, err = s.store.DisburseByLoanProvider(ctx, entity.DisburseByLoanProviderParams{
		ID:                    params.RequestID,
		LoanProviderOrgID:     uuid.NullUUID{UUID: params.OrgID, Valid: true},
		LoanProviderUserID:    uuid.NullUUID{UUID: params.UserID, Valid: true},
		PaymentProofImageUrls: params.PaymentProofImageUrls,
	})
	if err != nil {
		return PaymentRequestReturn{}, errs.NewInternalServerError(err)
	}

	// Return updated request
	return s.GetPaymentRequestByID(ctx, GetPaymentRequestByIDParams{RequestID: params.RequestID})
}

// ===== HELPER FUNCTIONS =====

func ensureStringArray(arr []string) []string {
	if arr == nil {
		return []string{}
	}
	return arr
}

func mapPaymentRequestRowToReturn(pr entity.GetPaymentRequestsByContractIDRow) PaymentRequestReturn {
	result := PaymentRequestReturn{
		ID:                    pr.ID,
		Title:                 pr.Title,
		Amount:                strconv.FormatInt(pr.Amount, 10),
		Status:                string(pr.Status),
		InvoiceFileUrl:        pr.InvoiceFileUrl,
		PaymentProofImageUrls: ensureStringArray(pr.PaymentProofImageUrls),
		ContractID:            pr.ContractID,
		CreatedAt:             pr.CreatedAt,
		UpdatedAt:             pr.UpdatedAt,
	}

	if pr.AuditorOrgID.Valid && pr.AuditorAt.Valid {
		result.Auditor = &ApprovalSignature{
			OrgID:      pr.AuditorOrgID.UUID,
			OrgName:    pr.AuditorOrgName.String,
			UserID:     pr.AuditorUserID.UUID,
			ApprovedAt: pr.AuditorAt.Time,
		}
	}
	if pr.ExecutingAgencyOrgID.Valid && pr.ExecutingAgencyAt.Valid {
		result.Agency = &ApprovalSignature{
			OrgID:      pr.ExecutingAgencyOrgID.UUID,
			OrgName:    pr.AgencyOrgName.String,
			UserID:     pr.ExecutingAgencyUserID.UUID,
			ApprovedAt: pr.ExecutingAgencyAt.Time,
		}
	}
	if pr.LoanProviderOrgID.Valid && pr.LoanProviderAt.Valid {
		result.LoanProvider = &ApprovalSignature{
			OrgID:      pr.LoanProviderOrgID.UUID,
			OrgName:    pr.LoanProviderOrgName.String,
			UserID:     pr.LoanProviderUserID.UUID,
			ApprovedAt: pr.LoanProviderAt.Time,
		}
	}

	return result
}
