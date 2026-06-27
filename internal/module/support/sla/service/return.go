package sla_service

import (
	"time"

	"github.com/google/uuid"
)

type SLAReturn struct {
	ID               uuid.UUID  `json:"id"`
	EntityType       string     `json:"entityType"`
	EntityID         uuid.UUID  `json:"entityId"`
	State            string     `json:"state"`
	OwnerOrgID       *uuid.UUID `json:"ownerOrgId"`
	EnteredAt        time.Time  `json:"enteredAt"`
	DueAt            time.Time  `json:"dueAt"`
	ResolvedAt       *time.Time `json:"resolvedAt"`
	Status           string     `json:"status"`
	RemainingSeconds int64      `json:"remainingSeconds"`
	IsViolated       bool       `json:"isViolated"`
}
