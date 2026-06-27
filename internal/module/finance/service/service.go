// internal/module/finance/service/service.go
package finance_service

import (
	"context"
	"database/sql"
	"strconv"

	"investment_monitoring/config"
	"investment_monitoring/internal/audit"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/internal/workflow"
	"investment_monitoring/pkg/errs"

	"github.com/google/uuid"
)

type FinanceService struct {
	store *entity.SQLStore
	audit *audit.Service
	cfg   *config.Config
}

func NewService(store *entity.SQLStore, cfg *config.Config) *FinanceService {
	return &FinanceService{
		store: store,
		audit: audit.NewService(store),
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

	var pr entity.PaymentRequest
	var validationErr error
	err = s.store.ExecTx(ctx, func(q *entity.Queries) error {
		var txErr error
		lockedContract, txErr := q.GetContractByIDForUpdate(ctx, contract.ID)
		if txErr != nil {
			return txErr
		}

		// Verify user's org is the contractor while holding the contract lock.
		if lockedContract.VendorParticipantID.Valid && lockedContract.VendorParticipantID.UUID != params.ContractorID {
			validationErr = errs.NewForbidden("NOT_CONTRACT_VENDOR")
			return validationErr
		}

		// Check contract_value is set while holding the contract lock.
		if !lockedContract.ContractValue.Valid || lockedContract.ContractValue.Int64 == 0 {
			validationErr = errs.NewBadRequest("CONTRACT_VALUE_NOT_SET")
			return validationErr
		}

		hasPending, txErr := q.HasPendingPaymentRequest(ctx, lockedContract.ID)
		if txErr != nil {
			return txErr
		}
		if hasPending {
			validationErr = errs.NewBadRequest("PENDING_REQUEST_EXISTS")
			return validationErr
		}

		paidAmount, txErr := q.GetPaidAmountByContractID(ctx, lockedContract.ID)
		if txErr != nil {
			return txErr
		}
		pendingAmount, txErr := q.GetPendingAmountByContractID(ctx, lockedContract.ID)
		if txErr != nil {
			return txErr
		}

		availableAmount := lockedContract.ContractValue.Int64 - paidAmount - pendingAmount
		if params.Amount > lockedContract.ContractValue.Int64 {
			validationErr = errs.NewBadRequest("AMOUNT_EXCEEDS_CONTRACT_VALUE")
			return validationErr
		}
		if params.Amount > availableAmount {
			validationErr = errs.NewBadRequest("AMOUNT_EXCEEDS_AVAILABLE")
			return validationErr
		}

		pr, txErr = q.CreatePaymentRequest(ctx, entity.CreatePaymentRequestParams{
			Title:          params.Title,
			Amount:         params.Amount,
			InvoiceFileUrl: params.InvoiceFileUrl,
			ContractID:     lockedContract.ID,
		})
		if txErr != nil {
			return txErr
		}

		return s.audit.RecordTx(ctx, q, audit.RecordParams{
			ActorUserID: params.UserID,
			ActorOrgID:  params.ContractorID,
			Action:      audit.ActionCreatePaymentRequest,
			EntityType:  audit.EntityPaymentRequest,
			EntityID:    pr.ID,
			AfterData: map[string]any{
				"paymentRequestId": pr.ID,
				"contractId":       lockedContract.ID,
				"title":            pr.Title,
				"amount":           pr.Amount,
				"availableBefore":  availableAmount,
				"status":           pr.Status,
			},
		})
	})
	if validationErr != nil {
		return result, validationErr
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

	newStatus, ok := workflow.FinanceNextStatus(
		prWithContract.Status,
		workflow.FinanceApprovalInput(params.Action, true),
	)
	if !ok {
		return PaymentRequestReturn{}, errs.NewBadRequest("INVALID_STATUS_TRANSITION")
	}

	err = s.store.ExecTx(ctx, func(q *entity.Queries) error {
		_, txErr := q.ApproveByAuditor(ctx, entity.ApproveByAuditorParams{
			ID:            params.RequestID,
			Status:        newStatus,
			AuditorOrgID:  uuid.NullUUID{UUID: params.OrgID, Valid: true},
			AuditorUserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
		})
		if txErr != nil {
			return txErr
		}

		return s.audit.RecordTx(ctx, q, audit.RecordParams{
			ActorUserID: params.UserID,
			ActorOrgID:  params.OrgID,
			Action:      audit.ActionApprovePaymentAuditor,
			EntityType:  audit.EntityPaymentRequest,
			EntityID:    params.RequestID,
			BeforeData: map[string]any{
				"status": prWithContract.Status,
			},
			AfterData: map[string]any{
				"status":       newStatus,
				"action":       params.Action,
				"auditorOrgId": params.OrgID,
			},
		})
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

	newStatus, ok := workflow.FinanceNextStatus(
		prWithContract.Status,
		workflow.FinanceApprovalInput(params.Action, false),
	)
	if !ok {
		return PaymentRequestReturn{}, errs.NewBadRequest("INVALID_STATUS_TRANSITION")
	}

	err = s.store.ExecTx(ctx, func(q *entity.Queries) error {
		_, txErr := q.ApproveByAgency(ctx, entity.ApproveByAgencyParams{
			ID:                    params.RequestID,
			Status:                newStatus,
			ExecutingAgencyOrgID:  uuid.NullUUID{UUID: params.OrgID, Valid: true},
			ExecutingAgencyUserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
		})
		if txErr != nil {
			return txErr
		}

		return s.audit.RecordTx(ctx, q, audit.RecordParams{
			ActorUserID: params.UserID,
			ActorOrgID:  params.OrgID,
			Action:      audit.ActionApprovePaymentAgency,
			EntityType:  audit.EntityPaymentRequest,
			EntityID:    params.RequestID,
			BeforeData: map[string]any{
				"status": prWithContract.Status,
			},
			AfterData: map[string]any{
				"status":               newStatus,
				"action":               params.Action,
				"executingAgencyOrgId": params.OrgID,
			},
		})
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

	_, ok := workflow.FinanceNextStatus(prWithContract.Status, workflow.FinanceInputDisburse)
	if !ok {
		return PaymentRequestReturn{}, errs.NewBadRequest("INVALID_STATUS_TRANSITION")
	}

	// Update with payment proof
	err = s.store.ExecTx(ctx, func(q *entity.Queries) error {
		_, txErr := q.DisburseByLoanProvider(ctx, entity.DisburseByLoanProviderParams{
			ID:                    params.RequestID,
			LoanProviderOrgID:     uuid.NullUUID{UUID: params.OrgID, Valid: true},
			LoanProviderUserID:    uuid.NullUUID{UUID: params.UserID, Valid: true},
			PaymentProofImageUrls: params.PaymentProofImageUrls,
		})
		if txErr != nil {
			return txErr
		}

		return s.audit.RecordTx(ctx, q, audit.RecordParams{
			ActorUserID: params.UserID,
			ActorOrgID:  params.OrgID,
			Action:      audit.ActionDisbursePayment,
			EntityType:  audit.EntityPaymentRequest,
			EntityID:    params.RequestID,
			BeforeData: map[string]any{
				"status": prWithContract.Status,
			},
			AfterData: map[string]any{
				"status":                entity.PaymentRequestStatusEnumPAID,
				"loanProviderOrgId":     params.OrgID,
				"paymentProofImageUrls": params.PaymentProofImageUrls,
			},
		})
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
