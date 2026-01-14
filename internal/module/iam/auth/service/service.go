// internal/module/iam/auth/service/service.go
package auth_service

import (
	"context"
	"database/sql"

	"investment_monitoring/config"
	token_service "investment_monitoring/internal/module/support/token/service"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/utils"
)

type AuthService struct {
	store    entity.Store
	tokenSvc *token_service.TokenMaker
	cfg      *config.Config
}

func NewService(store entity.Store, tokenSvc *token_service.TokenMaker, cfg *config.Config) *AuthService {
	return &AuthService{
		store:    store,
		tokenSvc: tokenSvc,
		cfg:      cfg,
	}
}

// CheckEmail determines the status of an email in the system
// Returns: NOT_FOUND, SETUP_REQUIRED, or LOGIN_READY
func (s *AuthService) CheckEmail(ctx context.Context, params CheckEmailParams) (CheckEmailReturn, error) {
	user, err := s.store.GetUserByEmailFull(ctx, params.Email)
	if err == sql.ErrNoRows {
		return CheckEmailReturn{Status: EmailStatusNotFound}, nil
	}
	if err != nil {
		return CheckEmailReturn{}, errs.NewInternalServerError(err)
	}

	// Check if password is set
	if !user.Password.Valid || user.Password.String == "" {
		return CheckEmailReturn{Status: EmailStatusSetupRequired}, nil
	}

	return CheckEmailReturn{Status: EmailStatusLoginReady}, nil
}

// Login authenticates user and returns JWT tokens
func (s *AuthService) Login(ctx context.Context, params LoginParams) (LoginReturn, error) {
	// Get user by email
	user, err := s.store.GetUserByEmailFull(ctx, params.Email)
	if err == sql.ErrNoRows {
		return LoginReturn{}, errs.NewUnauthorized("INVALID_CREDENTIALS")
	}
	if err != nil {
		return LoginReturn{}, errs.NewInternalServerError(err)
	}

	// Check if password is set
	if !user.Password.Valid || user.Password.String == "" {
		return LoginReturn{}, errs.NewBadRequest("PASSWORD_NOT_SET")
	}

	// Verify password
	if !utils.ComparePassword(user.Password.String, params.Password) {
		return LoginReturn{}, errs.NewUnauthorized("INVALID_CREDENTIALS")
	}

	// Generate tokens
	accessToken, err := s.tokenSvc.GenerateAccessToken(token_service.GenerateAccessTokenInput{
		ID:    user.ID,
		Email: user.Email,
	})
	if err != nil {
		return LoginReturn{}, errs.NewInternalServerError(err)
	}

	refreshToken, err := s.tokenSvc.GenerateRefreshToken(token_service.GenerateRefreshTokenInput{
		ID:    user.ID,
		Email: user.Email,
	})
	if err != nil {
		return LoginReturn{}, errs.NewInternalServerError(err)
	}

	// Get user memberships
	memberships, err := s.store.GetUserMemberships(ctx, user.ID)
	if err != nil {
		return LoginReturn{}, errs.NewInternalServerError(err)
	}

	// Map memberships to return type
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

	// Map user image URL
	var imageUrl *string
	if user.ImageUrl.Valid {
		imageUrl = &user.ImageUrl.String
	}

	return LoginReturn{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: UserReturn{
			ID:       user.ID,
			Name:     user.Name,
			Email:    user.Email,
			ImageUrl: imageUrl,
		},
		Memberships: membershipReturns,
	}, nil
}

// SetPassword sets password for a user who was invited (password is null)
func (s *AuthService) SetPassword(ctx context.Context, params SetPasswordParams) (SetPasswordReturn, error) {
	// Get user by email
	user, err := s.store.GetUserByEmailFull(ctx, params.Email)
	if err == sql.ErrNoRows {
		return SetPasswordReturn{}, errs.NewNotFound("USER_NOT_FOUND")
	}
	if err != nil {
		return SetPasswordReturn{}, errs.NewInternalServerError(err)
	}

	// Check if password is already set
	if user.Password.Valid && user.Password.String != "" {
		return SetPasswordReturn{}, errs.NewBadRequest("PASSWORD_ALREADY_SET")
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(params.Password)
	if err != nil {
		return SetPasswordReturn{}, errs.NewInternalServerError(err)
	}

	// Update password
	_, err = s.store.SetUserPassword(ctx, entity.SetUserPasswordParams{
		ID:       user.ID,
		Password: sql.NullString{String: hashedPassword, Valid: true},
	})
	if err != nil {
		return SetPasswordReturn{}, errs.NewInternalServerError(err)
	}

	return SetPasswordReturn{
		Message: "PASSWORD_SET_SUCCESSFULLY",
	}, nil
}

// RefreshToken generates new access token using refresh token
func (s *AuthService) RefreshToken(ctx context.Context, params RefreshTokenParams) (RefreshTokenReturn, error) {
	// Validate refresh token
	claims, err := s.tokenSvc.ValidateRefreshToken(params.RefreshToken)
	if err != nil {
		return RefreshTokenReturn{}, errs.NewUnauthorized("INVALID_REFRESH_TOKEN")
	}

	// Verify user still exists
	user, err := s.store.GetUserByID(ctx, claims.ID)
	if err == sql.ErrNoRows {
		return RefreshTokenReturn{}, errs.NewUnauthorized("USER_NOT_FOUND")
	}
	if err != nil {
		return RefreshTokenReturn{}, errs.NewInternalServerError(err)
	}

	// Generate new tokens
	accessToken, err := s.tokenSvc.GenerateAccessToken(token_service.GenerateAccessTokenInput{
		ID:    user.ID,
		Email: user.Email,
	})
	if err != nil {
		return RefreshTokenReturn{}, errs.NewInternalServerError(err)
	}

	refreshToken, err := s.tokenSvc.GenerateRefreshToken(token_service.GenerateRefreshTokenInput{
		ID:    user.ID,
		Email: user.Email,
	})
	if err != nil {
		return RefreshTokenReturn{}, errs.NewInternalServerError(err)
	}

	return RefreshTokenReturn{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
