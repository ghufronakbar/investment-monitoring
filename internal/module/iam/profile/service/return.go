// internal/module/iam/profile/service/return.go
package profile_service

import (
	"time"

	"github.com/google/uuid"
)

// ProfileReturn represents the user profile response
type ProfileReturn struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Email       string       `json:"email"`
	ImageUrl    *string      `json:"imageUrl"`
	Memberships []Membership `json:"memberships"`
}

// Membership represents organization membership
type Membership struct {
	ID         uuid.UUID `json:"id"`
	MemberRole string    `json:"memberRole"`
	OrgID      uuid.UUID `json:"orgId"`
	OrgName    string    `json:"orgName"`
	OrgRole    string    `json:"orgRole"`
	OrgLogoUrl *string   `json:"orgLogoUrl"`
	CreatedAt  time.Time `json:"createdAt"`
}

// UpdateProfileReturn represents the response after updating profile
type UpdateProfileReturn struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	ImageUrl *string   `json:"imageUrl"`
}

// ChangePasswordReturn represents the response after changing password
type ChangePasswordReturn struct {
	Message string `json:"message"`
}
