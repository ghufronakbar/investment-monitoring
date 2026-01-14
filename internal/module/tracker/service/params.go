// internal/module/tracker/service/params.go
package tracker_service

import (
	"github.com/google/uuid"
)

// ===== TICKETS =====

type CreateTicketParams struct {
	ProjectID         uuid.UUID
	ContractorID      uuid.UUID  // From JWT context (organization)
	UserID            uuid.UUID  // From JWT context
	Title             string     `json:"title" validate:"required"`
	Description       string     `json:"description" validate:"required"`
	Location          string     `json:"location" validate:"required"`
	EvidenceImageUrls []string   `json:"evidenceImageUrls" validate:"required,min=1,dive,url"`
	TaskItemID        *uuid.UUID `json:"taskItemId"` // Optional link to WBS task
}

type GetTicketsParams struct {
	ProjectID    uuid.UUID
	StatusFilter string // "open", "closed", or empty for all
}

type GetTicketByIDParams struct {
	TicketID uuid.UUID
}

type AssignTicketParams struct {
	TicketID      uuid.UUID
	AssignToOrgID uuid.UUID `json:"assignToOrgId" validate:"required"`
	OrgID         uuid.UUID // Executor org from JWT
	UserID        uuid.UUID // From JWT
}

// ===== TICKET LOGS =====

type AddTicketLogParams struct {
	TicketID      uuid.UUID
	OrgID         uuid.UUID // From JWT
	UserID        uuid.UUID // From JWT
	Description   string    `json:"description" validate:"required"`
	NewStatus     *string   `json:"newStatus"` // Optional status change
	AttachmentUrl *string   `json:"attachmentUrl" validate:"omitempty,url"`
}

type CloseTicketParams struct {
	TicketID uuid.UUID
	OrgID    uuid.UUID // From JWT
	UserID   uuid.UUID // From JWT
}
