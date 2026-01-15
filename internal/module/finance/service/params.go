// internal/module/finance/service/params.go
package finance_service

import (
	"github.com/google/uuid"
)

// ===== PAYMENT REQUESTS =====

type CreatePaymentRequestParams struct {
	ProjectID      uuid.UUID
	ContractorID   uuid.UUID // From JWT context (organization)
	UserID         uuid.UUID // From JWT context
	Title          string    `json:"title" validate:"required"`
	Amount         int64     `json:"amount" validate:"required,gt=0"`
	InvoiceFileUrl string    `json:"invoiceFileUrl" validate:"required,url"`
}

type GetPaymentRequestsParams struct {
	ProjectID uuid.UUID
}

type GetPaymentRequestByIDParams struct {
	RequestID uuid.UUID
}

type ApprovalParams struct {
	RequestID uuid.UUID
	OrgID     uuid.UUID // From JWT
	UserID    uuid.UUID // From JWT
	Action    string    `json:"action" validate:"required,oneof=APPROVE REJECT"`
}

type DisburseParams struct {
	RequestID             uuid.UUID
	OrgID                 uuid.UUID // From JWT
	UserID                uuid.UUID // From JWT
	PaymentProofImageUrls []string  `json:"paymentProofImageUrls" validate:"required,min=1,dive,url"`
}

type GetPaymentSummaryParams struct {
	ProjectID uuid.UUID
}
