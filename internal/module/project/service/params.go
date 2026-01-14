// internal/module/project/service/params.go
package project_service

import (
	"time"

	"github.com/google/uuid"
)

type CreateProjectParams struct {
	Name            string    `json:"name" validate:"required"`
	Description     string    `json:"description" validate:"required"`
	ContractFileUrl string    `json:"contractFileUrl" validate:"required,url"`
	LoanProviderID  uuid.UUID `json:"loanProviderId" validate:"required"`
	ImageUrls       []string  `json:"imageUrls" validate:"required,min=1,dive,url"`
}

type GetProjectParams struct {
	ProjectID uuid.UUID
	UserID    uuid.UUID // for access check
}

type ConfigureSettingsParams struct {
	ProjectID        uuid.UUID
	StartDate        time.Time `json:"startDate" validate:"required"`
	BaselineDeadline time.Time `json:"baselineDeadline" validate:"required"`
	TotalBudget      int64     `json:"totalBudget" validate:"required,gt=0"`
	Currency         string    `json:"currency" validate:"required"`
}

type AssignExecutingAgencyParams struct {
	ProjectID         uuid.UUID
	ExecutingAgencyID uuid.UUID `json:"executingAgencyId" validate:"required"`
}

type AssignVendorParams struct {
	ProjectID      uuid.UUID
	VendorOrgID    uuid.UUID `json:"vendorOrgId" validate:"required"`
	ContractValue  int64     `json:"contractValue" validate:"required,gt=0"`
	ContractDocUrl string    `json:"contractDocUrl" validate:"required,url"`
}

type AssignAuditorParams struct {
	ProjectID    uuid.UUID
	AuditorOrgID uuid.UUID `json:"auditorOrgId" validate:"required"`
}

type AssignGovPartnersParams struct {
	ProjectID     uuid.UUID
	GovPartnerIDs []uuid.UUID `json:"govPartnerIds" validate:"required,min=1"`
}
