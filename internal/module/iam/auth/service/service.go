// internal/module/iam/auth/service/service.go
package auth_service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"investment_monitoring/config"
	token_service "investment_monitoring/internal/module/support/token/service"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/utils"

	"github.com/redis/go-redis/v9"
)

type AuthService struct {
	store    entity.Store
	tokenSvc *token_service.TokenMaker
	rdb      *redis.Client
	cfg      *config.Config
}

func NewService(store entity.Store, tokenSvc *token_service.TokenMaker, rdb *redis.Client, cfg *config.Config) *AuthService {
	return &AuthService{
		store:    store,
		tokenSvc: tokenSvc,
		rdb:      rdb,
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
	if err := s.ensureRefreshTokenActive(ctx, claims); err != nil {
		return RefreshTokenReturn{}, err
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

	if s.rdb != nil {
		if err := s.blacklistRefreshToken(ctx, claims); err != nil {
			return RefreshTokenReturn{}, err
		}
	}

	return RefreshTokenReturn{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, params LogoutParams) (LogoutReturn, error) {
	claims, err := s.tokenSvc.ValidateRefreshToken(params.RefreshToken)
	if err != nil {
		return LogoutReturn{}, errs.NewUnauthorized("INVALID_REFRESH_TOKEN")
	}
	if err := s.ensureRefreshTokenActive(ctx, claims); err != nil {
		return LogoutReturn{}, err
	}
	if err := s.blacklistRefreshToken(ctx, claims); err != nil {
		return LogoutReturn{}, err
	}

	return LogoutReturn{Message: "LOGOUT_SUCCESSFULLY"}, nil
}

func (s *AuthService) RevokeAll(ctx context.Context, params RevokeAllParams) (RevokeAllReturn, error) {
	claims, err := s.tokenSvc.ValidateRefreshToken(params.RefreshToken)
	if err != nil {
		return RevokeAllReturn{}, errs.NewUnauthorized("INVALID_REFRESH_TOKEN")
	}
	if err := s.ensureRefreshTokenActive(ctx, claims); err != nil {
		return RevokeAllReturn{}, err
	}
	if err := s.revokeAllRefreshTokens(ctx, claims.ID.String()); err != nil {
		return RevokeAllReturn{}, err
	}
	if err := s.blacklistRefreshToken(ctx, claims); err != nil {
		return RevokeAllReturn{}, err
	}

	return RevokeAllReturn{Message: "REFRESH_TOKENS_REVOKED_SUCCESSFULLY"}, nil
}

func (s *AuthService) ensureRefreshTokenActive(ctx context.Context, claims *token_service.RefreshTokenClaims) error {
	if s.rdb == nil {
		return nil
	}

	blacklisted, err := s.rdb.Exists(ctx, s.refreshBlacklistKey(claims.RegisteredClaims.ID)).Result()
	if err != nil {
		return errs.NewInternalServerError(err)
	}
	if blacklisted > 0 {
		return errs.NewUnauthorized("REFRESH_TOKEN_REVOKED")
	}

	cutoffValue, err := s.rdb.Get(ctx, s.refreshUserRevokeAllKey(claims.ID.String())).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return errs.NewInternalServerError(err)
	}
	if errors.Is(err, redis.Nil) {
		return nil
	}

	cutoffUnixNano, err := strconv.ParseInt(cutoffValue, 10, 64)
	if err != nil {
		return errs.NewInternalServerError(err)
	}
	if claims.IssuedAt == nil || !claims.IssuedAt.Time.After(time.Unix(0, cutoffUnixNano)) {
		return errs.NewUnauthorized("REFRESH_TOKEN_REVOKED")
	}
	return nil
}

func (s *AuthService) blacklistRefreshToken(ctx context.Context, claims *token_service.RefreshTokenClaims) error {
	if s.rdb == nil {
		return errs.NewInternalServerError(errors.New("redis is required for refresh token revocation"))
	}
	ttl := s.refreshTokenTTL(claims)
	if ttl <= 0 {
		return nil
	}
	if err := s.rdb.Set(ctx, s.refreshBlacklistKey(claims.RegisteredClaims.ID), claims.ID.String(), ttl).Err(); err != nil {
		return errs.NewInternalServerError(err)
	}
	return nil
}

func (s *AuthService) revokeAllRefreshTokens(ctx context.Context, userID string) error {
	if s.rdb == nil {
		return errs.NewInternalServerError(errors.New("redis is required for refresh token revocation"))
	}
	key := s.refreshUserRevokeAllKey(userID)
	if err := s.rdb.Set(ctx, key, time.Now().UnixNano(), s.cfg.JWT_REFRESH_TOKEN_EXPIRED).Err(); err != nil {
		return errs.NewInternalServerError(err)
	}
	return nil
}

func (s *AuthService) refreshTokenTTL(claims *token_service.RefreshTokenClaims) time.Duration {
	if claims.ExpiresAt == nil {
		return 0
	}
	return time.Until(claims.ExpiresAt.Time)
}

func (s *AuthService) refreshBlacklistKey(jti string) string {
	return fmt.Sprintf("%s:blacklist:refresh:%s", s.cfg.REDIS_PREFIX, jti)
}

func (s *AuthService) refreshUserRevokeAllKey(userID string) string {
	return fmt.Sprintf("%s:revoke-all:refresh:%s", s.cfg.REDIS_PREFIX, userID)
}
