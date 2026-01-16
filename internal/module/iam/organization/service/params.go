// internal/module/iam/organization/service/params.go
package organization_service

import "github.com/google/uuid"

type CreateOrganizationParams struct {
	Name       string `json:"name" validate:"required"`
	Role       string `json:"role" validate:"required,oneof=CONTRACTOR GOV_PARTNER INDEPENDENT_AUDITOR LOAN_PROVIDER EXECUTING_AGENCY"`
	LogoUrl    string `json:"logoUrl"`
	AdminName  string `json:"adminName" validate:"required"`
	AdminEmail string `json:"adminEmail" validate:"required,email"`
}

type InviteMemberParams struct {
	OrganizationID uuid.UUID `json:"-"` // from URL param
	Name           string    `json:"name" validate:"required"`
	Email          string    `json:"email" validate:"required,email"`
}

type GetOrganizationParams struct {
	ID uuid.UUID `json:"-"` // from URL param
}

type GetMembersParams struct {
	OrganizationID uuid.UUID `json:"-"` // from URL param
}

type GetMyOrganizationsParams struct {
	UserID uuid.UUID `json:"-"` // from authenticated user
}

type UpdateOrganizationParams struct {
	ID      uuid.UUID `json:"-"` // from URL param
	Name    string    `json:"name" validate:"required,min=2"`
	LogoUrl *string   `json:"logoUrl"`
}
