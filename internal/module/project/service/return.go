// internal/module/project/service/return.go
package project_service

import (
	"time"

	"github.com/google/uuid"
)

type ProjectReturn struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	ContractFileUrl string    `json:"contractFileUrl"`
	ImageUrls       []string  `json:"imageUrls"`
	CreatedAt       time.Time `json:"createdAt"`
}

type ProjectDetailReturn struct {
	Project  ProjectReturn          `json:"project"`
	Contract ContractReturn         `json:"contract"`
	Settings *ProjectSettingsReturn `json:"settings"`
}

type ContractReturn struct {
	ID                uuid.UUID           `json:"id"`
	Description       string              `json:"description"`
	ContractValue     *int64              `json:"contractValue"`
	ContractDocUrl    *string             `json:"contractDocUrl"`
	GovSupervisor     ParticipantReturn   `json:"govSupervisor"`
	LoanProvider      ParticipantReturn   `json:"loanProvider"`
	ExecutingAgency   *ParticipantReturn  `json:"executingAgency"`
	Vendor            *ParticipantReturn  `json:"vendor"`
	VendorSelectedAt  *time.Time          `json:"vendorSelectedAt"`
	Auditor           *ParticipantReturn  `json:"auditor"`
	AuditorSelectedAt *time.Time          `json:"auditorSelectedAt"`
	GovPartners       []ParticipantReturn `json:"govPartners"`
}

type ParticipantReturn struct {
	OrgID   uuid.UUID `json:"orgId"`
	OrgName string    `json:"orgName"`
	OrgRole string    `json:"orgRole"`
	LogoUrl *string   `json:"logoUrl"`
}

type ProjectSettingsReturn struct {
	ID               uuid.UUID `json:"id"`
	StartDate        time.Time `json:"startDate"`
	BaselineDeadline time.Time `json:"baselineDeadline"`
	TotalBudget      int64     `json:"totalBudget"`
	Currency         string    `json:"currency"`
}

type CreateProjectReturn struct {
	Project  ProjectReturn  `json:"project"`
	Contract ContractReturn `json:"contract"`
}

type AssignmentReturn struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
