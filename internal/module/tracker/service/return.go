// internal/module/tracker/service/return.go
package tracker_service

import (
	sla_service "investment_monitoring/internal/module/support/sla/service"
	"time"

	"github.com/google/uuid"
)

// ===== TICKETS =====

type TicketReturn struct {
	ID                uuid.UUID              `json:"id"`
	Title             string                 `json:"title"`
	Description       string                 `json:"description"`
	Location          string                 `json:"location"`
	EvidenceImageUrls []string               `json:"evidenceImageUrls"`
	TaskItemID        *uuid.UUID             `json:"taskItemId"`
	TaskTitle         *string                `json:"taskTitle"`
	ContractID        uuid.UUID              `json:"contractId"`
	CurrentStatus     string                 `json:"currentStatus"`
	Creator           TicketActorReturn      `json:"creator"`
	AssignedTo        *TicketActorReturn     `json:"assignedTo"`
	Logs              []TicketLogReturn      `json:"logs,omitempty"`
	SLA               *sla_service.SLAReturn `json:"sla,omitempty"`
	CreatedAt         time.Time              `json:"createdAt"`
	LastLogAt         *time.Time             `json:"lastLogAt,omitempty"`
}

type TicketActorReturn struct {
	OrgID   uuid.UUID `json:"orgId"`
	OrgName string    `json:"orgName"`
}

// ===== TICKET LOGS =====

type TicketLogReturn struct {
	ID             uuid.UUID `json:"id"`
	Action         string    `json:"action"`
	Description    string    `json:"description"`
	AttachmentUrl  *string   `json:"attachmentUrl"`
	PreviousStatus *string   `json:"previousStatus"`
	NewStatus      string    `json:"newStatus"`
	OrgID          uuid.UUID `json:"orgId"`
	OrgName        string    `json:"orgName"`
	UserID         uuid.UUID `json:"userId"`
	UserName       string    `json:"userName"`
	CreatedAt      time.Time `json:"createdAt"`
}
