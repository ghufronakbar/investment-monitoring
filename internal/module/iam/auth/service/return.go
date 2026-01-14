// internal/module/iam/auth/service/return.go
package auth_service

import (
	"time"

	"github.com/google/uuid"
)

// EmailStatus represents the status of an email check
type EmailStatus string

const (
	EmailStatusNotFound      EmailStatus = "NOT_FOUND"
	EmailStatusSetupRequired EmailStatus = "SETUP_REQUIRED"
	EmailStatusLoginReady    EmailStatus = "LOGIN_READY"
)

type CheckEmailReturn struct {
	Status EmailStatus `json:"status"`
}

type LoginReturn struct {
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	User         UserReturn   `json:"user"`
	Memberships  []Membership `json:"memberships"`
}

type UserReturn struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	ImageUrl *string   `json:"imageUrl"`
}

type Membership struct {
	ID         uuid.UUID `json:"id"`
	MemberRole string    `json:"memberRole"`
	OrgID      uuid.UUID `json:"orgId"`
	OrgName    string    `json:"orgName"`
	OrgRole    string    `json:"orgRole"`
	OrgLogoUrl *string   `json:"orgLogoUrl"`
	CreatedAt  time.Time `json:"createdAt"`
}

type SetPasswordReturn struct {
	Message string `json:"message"`
}

type RefreshTokenReturn struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}
