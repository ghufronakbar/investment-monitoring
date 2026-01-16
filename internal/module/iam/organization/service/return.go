// internal/module/iam/organization/service/return.go
package organization_service

import (
	"time"

	"github.com/google/uuid"
)

type CreateOrganizationReturn struct {
	Organization OrganizationReturn `json:"organization"`
	Admin        AdminReturn        `json:"admin"`
}

type OrganizationReturn struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	LogoUrl   *string   `json:"logoUrl"`
	CreatedAt time.Time `json:"createdAt"`
}

type AdminReturn struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	ImageUrl *string   `json:"imageUrl"`
}

type InviteMemberReturn struct {
	Member MemberReturn `json:"member"`
}

type MemberReturn struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	MemberRole string    `json:"memberRole"`
	CreatedAt  time.Time `json:"createdAt"`
}

type GetOrganizationReturn struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	LogoUrl   *string   `json:"logoUrl"`
	CreatedAt time.Time `json:"createdAt"`
}

type MemberDetailReturn struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"userId"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	ImageUrl   *string   `json:"imageUrl"`
	MemberRole string    `json:"memberRole"`
	CreatedAt  time.Time `json:"createdAt"`
}

type MyOrganizationReturn struct {
	MembershipID uuid.UUID `json:"membershipId"`
	MemberRole   string    `json:"memberRole"`
	OrgID        uuid.UUID `json:"orgId"`
	OrgName      string    `json:"orgName"`
	OrgRole      string    `json:"orgRole"`
	OrgLogoUrl   *string   `json:"orgLogoUrl"`
	JoinedAt     time.Time `json:"joinedAt"`
}

type UpdateOrganizationReturn struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	LogoUrl   *string   `json:"logoUrl"`
	UpdatedAt time.Time `json:"updatedAt"`
}
