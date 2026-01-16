// internal/module/iam/profile/service/service.go
package profile_service

import (
	"context"
	"database/sql"

	"investment_monitoring/config"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/utils"

	"github.com/google/uuid"
)

type ProfileService struct {
	store entity.Store
	cfg   *config.Config
}

func NewService(store entity.Store, cfg *config.Config) *ProfileService {
	return &ProfileService{
		store: store,
		cfg:   cfg,
	}
}

// GetProfile retrieves current user's profile with memberships
func (s *ProfileService) GetProfile(ctx context.Context, userID uuid.UUID) (ProfileReturn, error) {
	// Get user by ID
	user, err := s.store.GetUserByID(ctx, userID)
	if err == sql.ErrNoRows {
		return ProfileReturn{}, errs.NewNotFound("USER_NOT_FOUND")
	}
	if err != nil {
		return ProfileReturn{}, errs.NewInternalServerError(err)
	}

	// Get memberships
	memberships, err := s.store.GetUserMemberships(ctx, userID)
	if err != nil {
		return ProfileReturn{}, errs.NewInternalServerError(err)
	}

	// Map memberships
	membershipReturns := make([]Membership, 0, len(memberships))
	for _, m := range memberships {
		var logoUrl *string
		if m.OrgLogoUrl.Valid {
			logoUrl = &m.OrgLogoUrl.String
		}
		membershipReturns = append(membershipReturns, Membership{
			ID:         m.ID,
			MemberRole: string(m.MemberRole),
			OrgID:      m.OrgID,
			OrgName:    m.OrgName,
			OrgRole:    string(m.OrgRole),
			OrgLogoUrl: logoUrl,
			CreatedAt:  m.CreatedAt,
		})
	}

	// Map image URL
	var imageUrl *string
	if user.ImageUrl.Valid {
		imageUrl = &user.ImageUrl.String
	}

	return ProfileReturn{
		ID:          user.ID,
		Name:        user.Name,
		Email:       user.Email,
		ImageUrl:    imageUrl,
		Memberships: membershipReturns,
	}, nil
}

// UpdateProfile updates user's name and image URL
func (s *ProfileService) UpdateProfile(ctx context.Context, userID uuid.UUID, params UpdateProfileParams) (UpdateProfileReturn, error) {
	// Prepare image URL
	var imageUrl sql.NullString
	if params.ImageUrl != nil && *params.ImageUrl != "" {
		imageUrl = sql.NullString{String: *params.ImageUrl, Valid: true}
	}

	// Update user
	user, err := s.store.UpdateUserProfile(ctx, entity.UpdateUserProfileParams{
		ID:       userID,
		Name:     params.Name,
		ImageUrl: imageUrl,
	})
	if err == sql.ErrNoRows {
		return UpdateProfileReturn{}, errs.NewNotFound("USER_NOT_FOUND")
	}
	if err != nil {
		return UpdateProfileReturn{}, errs.NewInternalServerError(err)
	}

	// Map response
	var returnImageUrl *string
	if user.ImageUrl.Valid {
		returnImageUrl = &user.ImageUrl.String
	}

	return UpdateProfileReturn{
		ID:       user.ID,
		Name:     user.Name,
		Email:    user.Email,
		ImageUrl: returnImageUrl,
	}, nil
}

// ChangePassword changes user's password (requires current password)
func (s *ProfileService) ChangePassword(ctx context.Context, userID uuid.UUID, params ChangePasswordParams) (ChangePasswordReturn, error) {
	// Get user to verify current password
	user, err := s.store.GetUserByEmailFull(ctx, "")
	// Use GetUserByID for full data
	user, err = s.store.GetUserByID(ctx, userID)
	if err == sql.ErrNoRows {
		return ChangePasswordReturn{}, errs.NewNotFound("USER_NOT_FOUND")
	}
	if err != nil {
		return ChangePasswordReturn{}, errs.NewInternalServerError(err)
	}

	// We need to get password to verify, but GetUserByID doesn't return password
	// So we need to use GetUserByEmailFull
	userFull, err := s.store.GetUserByEmailFull(ctx, user.Email)
	if err != nil {
		return ChangePasswordReturn{}, errs.NewInternalServerError(err)
	}

	// Check if password is set
	if !userFull.Password.Valid || userFull.Password.String == "" {
		return ChangePasswordReturn{}, errs.NewBadRequest("PASSWORD_NOT_SET")
	}

	// Verify current password
	if !utils.ComparePassword(userFull.Password.String, params.CurrentPassword) {
		return ChangePasswordReturn{}, errs.NewBadRequest("CURRENT_PASSWORD_INCORRECT")
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(params.NewPassword)
	if err != nil {
		return ChangePasswordReturn{}, errs.NewInternalServerError(err)
	}

	// Update password
	_, err = s.store.UpdateUserPasswordByID(ctx, entity.UpdateUserPasswordByIDParams{
		ID:       userID,
		Password: sql.NullString{String: hashedPassword, Valid: true},
	})
	if err != nil {
		return ChangePasswordReturn{}, errs.NewInternalServerError(err)
	}

	return ChangePasswordReturn{
		Message: "PASSWORD_CHANGED_SUCCESSFULLY",
	}, nil
}
