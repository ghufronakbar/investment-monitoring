// internal/module/finance/service/return.go
package finance_service

import (
	"time"

	"github.com/google/uuid"
)

// ===== PAYMENT REQUESTS =====

type PaymentRequestReturn struct {
	ID                    uuid.UUID          `json:"id"`
	Title                 string             `json:"title"`
	Amount                string             `json:"amount"` // String to preserve BigInt precision
	Status                string             `json:"status"`
	InvoiceFileUrl        string             `json:"invoiceFileUrl"`
	PaymentProofImageUrls []string           `json:"paymentProofImageUrls"` // Always array, never null
	ContractID            uuid.UUID          `json:"contractId"`
	Auditor               *ApprovalSignature `json:"auditor"`
	Agency                *ApprovalSignature `json:"agency"`
	LoanProvider          *ApprovalSignature `json:"loanProvider"`
	CreatedAt             time.Time          `json:"createdAt"`
	UpdatedAt             time.Time          `json:"updatedAt"`
}

type ApprovalSignature struct {
	OrgID      uuid.UUID `json:"orgId"`
	OrgName    string    `json:"orgName"`
	UserID     uuid.UUID `json:"userId"`
	ApprovedAt time.Time `json:"approvedAt"`
}

// ===== PAYMENT SUMMARY =====

type PaymentSummaryReturn struct {
	ContractValue   string `json:"contractValue"`   // Total contract value
	PaidAmount      string `json:"paidAmount"`      // Sum of PAID requests
	PendingAmount   string `json:"pendingAmount"`   // Sum of in-progress requests
	AvailableAmount string `json:"availableAmount"` // ContractValue - PaidAmount - PendingAmount
	CanSubmit       bool   `json:"canSubmit"`       // False if has pending request
	Currency        string `json:"currency"`        // From project settings
}
