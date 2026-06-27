package finance_service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/internal/testutil"

	"github.com/google/uuid"
)

type fakeStore struct {
	entity.Store

	contract               entity.Contract
	contractErr            error
	contractWithSettings   entity.GetContractWithProjectSettingsRow
	contractSettingsErr    error
	paidAmount             int64
	paidAmountErr          error
	pendingAmount          int64
	pendingAmountErr       error
	hasPending             bool
	hasPendingErr          error
	paymentRequest         entity.GetPaymentRequestByIDRow
	paymentRequestErr      error
	paymentWithContract    entity.GetPaymentRequestWithContractRow
	paymentWithContractErr error
}

func (f *fakeStore) GetContractByProjectID(ctx context.Context, projectID uuid.UUID) (entity.Contract, error) {
	if f.contractErr != nil {
		return entity.Contract{}, f.contractErr
	}
	return f.contract, nil
}

func (f *fakeStore) GetContractWithProjectSettings(ctx context.Context, id uuid.UUID) (entity.GetContractWithProjectSettingsRow, error) {
	if f.contractSettingsErr != nil {
		return entity.GetContractWithProjectSettingsRow{}, f.contractSettingsErr
	}
	return f.contractWithSettings, nil
}

func (f *fakeStore) GetPaidAmountByContractID(ctx context.Context, contractID uuid.UUID) (int64, error) {
	if f.paidAmountErr != nil {
		return 0, f.paidAmountErr
	}
	return f.paidAmount, nil
}

func (f *fakeStore) GetPendingAmountByContractID(ctx context.Context, contractID uuid.UUID) (int64, error) {
	if f.pendingAmountErr != nil {
		return 0, f.pendingAmountErr
	}
	return f.pendingAmount, nil
}

func (f *fakeStore) HasPendingPaymentRequest(ctx context.Context, contractID uuid.UUID) (bool, error) {
	if f.hasPendingErr != nil {
		return false, f.hasPendingErr
	}
	return f.hasPending, nil
}

func (f *fakeStore) GetPaymentRequestByID(ctx context.Context, id uuid.UUID) (entity.GetPaymentRequestByIDRow, error) {
	if f.paymentRequestErr != nil {
		return entity.GetPaymentRequestByIDRow{}, f.paymentRequestErr
	}
	return f.paymentRequest, nil
}

func (f *fakeStore) GetPaymentRequestWithContract(ctx context.Context, id uuid.UUID) (entity.GetPaymentRequestWithContractRow, error) {
	if f.paymentWithContractErr != nil {
		return entity.GetPaymentRequestWithContractRow{}, f.paymentWithContractErr
	}
	return f.paymentWithContract, nil
}

func TestGetPaymentSummaryCalculatesAvailableAndCanSubmit(t *testing.T) {
	projectID := uuid.New()
	contractID := uuid.New()
	store := &fakeStore{
		contract: entity.Contract{ID: contractID, ProjectID: projectID},
		contractWithSettings: entity.GetContractWithProjectSettingsRow{
			ID:            contractID,
			ProjectID:     projectID,
			ContractValue: sql.NullInt64{Int64: 1000, Valid: true},
			Currency:      "IDR",
		},
		paidAmount:    300,
		pendingAmount: 200,
		hasPending:    false,
	}

	got, err := NewService(store, nil, nil).GetPaymentSummary(context.Background(), GetPaymentSummaryParams{ProjectID: projectID})
	if err != nil {
		t.Fatalf("GetPaymentSummary returned error: %v", err)
	}
	if got.ContractValue != "1000" || got.PaidAmount != "300" || got.PendingAmount != "200" || got.AvailableAmount != "500" {
		t.Fatalf("summary = %+v, want calculated amounts", got)
	}
	if !got.CanSubmit {
		t.Fatalf("CanSubmit = false, want true")
	}
	if got.Currency != "IDR" {
		t.Fatalf("currency = %s, want IDR", got.Currency)
	}
}

func TestGetPaymentSummaryClampsNegativeAvailableAndBlocksPending(t *testing.T) {
	projectID := uuid.New()
	contractID := uuid.New()
	store := &fakeStore{
		contract: entity.Contract{ID: contractID, ProjectID: projectID},
		contractWithSettings: entity.GetContractWithProjectSettingsRow{
			ID:            contractID,
			ProjectID:     projectID,
			ContractValue: sql.NullInt64{Int64: 1000, Valid: true},
			Currency:      "IDR",
		},
		paidAmount:    900,
		pendingAmount: 300,
		hasPending:    true,
	}

	got, err := NewService(store, nil, nil).GetPaymentSummary(context.Background(), GetPaymentSummaryParams{ProjectID: projectID})
	if err != nil {
		t.Fatalf("GetPaymentSummary returned error: %v", err)
	}
	if got.AvailableAmount != "0" {
		t.Fatalf("available amount = %s, want 0", got.AvailableAmount)
	}
	if got.CanSubmit {
		t.Fatal("CanSubmit = true, want false when pending exists")
	}
}

func TestGetPaymentSummaryReturnsDomainErrors(t *testing.T) {
	tests := []struct {
		name    string
		store   *fakeStore
		wantMsg string
	}{
		{
			name:    "missing contract",
			store:   &fakeStore{contractErr: sql.ErrNoRows},
			wantMsg: "CONTRACT_NOT_FOUND",
		},
		{
			name: "store error",
			store: &fakeStore{
				contract:            entity.Contract{ID: uuid.New()},
				contractSettingsErr: errors.New("db down"),
			},
			wantMsg: "INTERNAL_SERVER_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewService(tt.store, nil, nil).GetPaymentSummary(context.Background(), GetPaymentSummaryParams{ProjectID: uuid.New()})
			testutil.AssertAppError(t, err, tt.wantMsg)
		})
	}
}

func TestGetPaymentRequestByIDMapsApprovalSignaturesAndNilProofs(t *testing.T) {
	requestID := uuid.New()
	contractID := uuid.New()
	auditorOrgID := uuid.New()
	auditorUserID := uuid.New()
	agencyOrgID := uuid.New()
	agencyUserID := uuid.New()
	loanOrgID := uuid.New()
	loanUserID := uuid.New()
	now := time.Now()

	store := &fakeStore{
		paymentRequest: entity.GetPaymentRequestByIDRow{
			ID:                    requestID,
			Title:                 "Termin 1",
			Amount:                1500,
			Status:                entity.PaymentRequestStatusEnumPAID,
			InvoiceFileUrl:        "https://example.com/invoice.pdf",
			ContractID:            contractID,
			AuditorOrgID:          uuid.NullUUID{UUID: auditorOrgID, Valid: true},
			AuditorUserID:         uuid.NullUUID{UUID: auditorUserID, Valid: true},
			AuditorAt:             sql.NullTime{Time: now, Valid: true},
			AuditorOrgName:        sql.NullString{String: "Auditor", Valid: true},
			ExecutingAgencyOrgID:  uuid.NullUUID{UUID: agencyOrgID, Valid: true},
			ExecutingAgencyUserID: uuid.NullUUID{UUID: agencyUserID, Valid: true},
			ExecutingAgencyAt:     sql.NullTime{Time: now, Valid: true},
			AgencyOrgName:         sql.NullString{String: "Agency", Valid: true},
			LoanProviderOrgID:     uuid.NullUUID{UUID: loanOrgID, Valid: true},
			LoanProviderUserID:    uuid.NullUUID{UUID: loanUserID, Valid: true},
			LoanProviderAt:        sql.NullTime{Time: now, Valid: true},
			LoanProviderOrgName:   sql.NullString{String: "Loan", Valid: true},
			PaymentProofImageUrls: nil,
			CreatedAt:             now,
			UpdatedAt:             now,
		},
	}

	got, err := NewService(store, nil, nil).GetPaymentRequestByID(context.Background(), GetPaymentRequestByIDParams{RequestID: requestID})
	if err != nil {
		t.Fatalf("GetPaymentRequestByID returned error: %v", err)
	}
	if got.Amount != "1500" || got.Status != string(entity.PaymentRequestStatusEnumPAID) {
		t.Fatalf("payment return = %+v, want mapped amount/status", got)
	}
	if got.PaymentProofImageUrls == nil || len(got.PaymentProofImageUrls) != 0 {
		t.Fatalf("payment proof urls = %#v, want empty non-nil array", got.PaymentProofImageUrls)
	}
	if got.Auditor == nil || got.Auditor.OrgName != "Auditor" || got.Auditor.UserID != auditorUserID {
		t.Fatalf("auditor signature = %+v, want mapped auditor", got.Auditor)
	}
	if got.Agency == nil || got.Agency.OrgName != "Agency" || got.Agency.UserID != agencyUserID {
		t.Fatalf("agency signature = %+v, want mapped agency", got.Agency)
	}
	if got.LoanProvider == nil || got.LoanProvider.OrgName != "Loan" || got.LoanProvider.UserID != loanUserID {
		t.Fatalf("loan signature = %+v, want mapped loan provider", got.LoanProvider)
	}
}

func TestFinanceApprovalPreTransactionValidation(t *testing.T) {
	requestID := uuid.New()
	orgID := uuid.New()

	t.Run("auditor must match contract auditor", func(t *testing.T) {
		store := &fakeStore{
			paymentWithContract: entity.GetPaymentRequestWithContractRow{
				ID:                   requestID,
				Status:               entity.PaymentRequestStatusEnumSUBMITTED,
				AuditorParticipantID: uuid.NullUUID{UUID: uuid.New(), Valid: true},
			},
		}

		_, err := NewService(store, nil, nil).ApproveByAuditor(context.Background(), ApprovalParams{
			RequestID: requestID,
			OrgID:     orgID,
			Action:    "APPROVE",
		})
		testutil.AssertAppError(t, err, "NOT_CONTRACT_AUDITOR")
	})

	t.Run("auditor invalid transition stops before transaction", func(t *testing.T) {
		store := &fakeStore{
			paymentWithContract: entity.GetPaymentRequestWithContractRow{
				ID:                   requestID,
				Status:               entity.PaymentRequestStatusEnumPAID,
				AuditorParticipantID: uuid.NullUUID{UUID: orgID, Valid: true},
			},
		}

		_, err := NewService(store, nil, nil).ApproveByAuditor(context.Background(), ApprovalParams{
			RequestID: requestID,
			OrgID:     orgID,
			Action:    "APPROVE",
		})
		testutil.AssertAppError(t, err, "INVALID_STATUS_TRANSITION")
	})

	t.Run("agency must match contract executing agency", func(t *testing.T) {
		store := &fakeStore{
			paymentWithContract: entity.GetPaymentRequestWithContractRow{
				ID:                           requestID,
				Status:                       entity.PaymentRequestStatusEnumAUDITORAPPROVED,
				ExecutingAgencyParticipantID: uuid.NullUUID{UUID: uuid.New(), Valid: true},
			},
		}

		_, err := NewService(store, nil, nil).ApproveByAgency(context.Background(), ApprovalParams{
			RequestID: requestID,
			OrgID:     orgID,
			Action:    "APPROVE",
		})
		testutil.AssertAppError(t, err, "NOT_CONTRACT_EXECUTING_AGENCY")
	})
}
