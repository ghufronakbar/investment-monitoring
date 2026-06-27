package auth_service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"investment_monitoring/config"
	token_service "investment_monitoring/internal/module/support/token/service"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/utils"

	"github.com/google/uuid"
)

type fakeStore struct {
	entity.Store

	user           entity.User
	userErr        error
	userByID       entity.User
	userByIDErr    error
	memberships    []entity.GetUserMembershipsRow
	membershipsErr error
	setPasswordArg entity.SetUserPasswordParams
	setPasswordErr error
}

func (f *fakeStore) GetUserByEmailFull(ctx context.Context, email string) (entity.User, error) {
	if f.userErr != nil {
		return entity.User{}, f.userErr
	}
	return f.user, nil
}

func (f *fakeStore) GetUserByID(ctx context.Context, id uuid.UUID) (entity.User, error) {
	if f.userByIDErr != nil {
		return entity.User{}, f.userByIDErr
	}
	if f.userByID.ID != uuid.Nil {
		return f.userByID, nil
	}
	return f.user, nil
}

func (f *fakeStore) GetUserMemberships(ctx context.Context, userID uuid.UUID) ([]entity.GetUserMembershipsRow, error) {
	if f.membershipsErr != nil {
		return nil, f.membershipsErr
	}
	return f.memberships, nil
}

func (f *fakeStore) SetUserPassword(ctx context.Context, arg entity.SetUserPasswordParams) (entity.User, error) {
	f.setPasswordArg = arg
	if f.setPasswordErr != nil {
		return entity.User{}, f.setPasswordErr
	}
	f.user.Password = arg.Password
	return f.user, nil
}

func TestCheckEmailStatuses(t *testing.T) {
	tests := []struct {
		name string
		user entity.User
		err  error
		want EmailStatus
	}{
		{name: "not found", err: sql.ErrNoRows, want: EmailStatusNotFound},
		{name: "setup required", user: entity.User{Password: sql.NullString{}}, want: EmailStatusSetupRequired},
		{name: "login ready", user: entity.User{Password: sql.NullString{String: "hash", Valid: true}}, want: EmailStatusLoginReady},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTestAuthService(&fakeStore{user: tt.user, userErr: tt.err}).CheckEmail(context.Background(), CheckEmailParams{Email: "user@example.com"})
			if err != nil {
				t.Fatalf("CheckEmail returned error: %v", err)
			}
			if got.Status != tt.want {
				t.Fatalf("status = %s, want %s", got.Status, tt.want)
			}
		})
	}
}

func TestLoginSuccessMapsTokensUserAndMemberships(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	logo := "https://example.com/logo.png"
	image := "https://example.com/user.png"
	hashed, err := utils.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	store := &fakeStore{
		user: entity.User{
			ID:       userID,
			Name:     "User",
			Email:    "user@example.com",
			ImageUrl: sql.NullString{String: image, Valid: true},
			Password: sql.NullString{String: hashed, Valid: true},
		},
		memberships: []entity.GetUserMembershipsRow{{
			ID:         uuid.New(),
			MemberRole: entity.MemberRoleEnumADMIN,
			OrgID:      orgID,
			OrgName:    "Agency",
			OrgLogoUrl: sql.NullString{String: logo, Valid: true},
			OrgRole:    entity.OrganizationRoleEnumEXECUTINGAGENCY,
			CreatedAt:  time.Now(),
		}},
	}

	got, err := newTestAuthService(store).Login(context.Background(), LoginParams{Email: "user@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if got.AccessToken == "" || got.RefreshToken == "" {
		t.Fatalf("tokens should be present: %+v", got)
	}
	if got.User.ID != userID || got.User.ImageUrl == nil || *got.User.ImageUrl != image {
		t.Fatalf("user = %+v, want mapped user", got.User)
	}
	if len(got.Memberships) != 1 || got.Memberships[0].OrgID != orgID || got.Memberships[0].OrgLogoUrl == nil || *got.Memberships[0].OrgLogoUrl != logo {
		t.Fatalf("memberships = %+v, want mapped membership", got.Memberships)
	}
}

func TestLoginFailurePaths(t *testing.T) {
	hashed, err := utils.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	tests := []struct {
		name     string
		store    *fakeStore
		password string
		want     string
	}{
		{name: "missing user", store: &fakeStore{userErr: sql.ErrNoRows}, password: "password123", want: "INVALID_CREDENTIALS"},
		{name: "password not set", store: &fakeStore{user: entity.User{Email: "user@example.com"}}, password: "password123", want: "PASSWORD_NOT_SET"},
		{name: "wrong password", store: &fakeStore{user: entity.User{Email: "user@example.com", Password: sql.NullString{String: hashed, Valid: true}}}, password: "wrongpassword", want: "INVALID_CREDENTIALS"},
		{name: "membership error", store: &fakeStore{user: entity.User{ID: uuid.New(), Email: "user@example.com", Password: sql.NullString{String: hashed, Valid: true}}, membershipsErr: errors.New("db down")}, password: "password123", want: "INTERNAL_SERVER_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newTestAuthService(tt.store).Login(context.Background(), LoginParams{Email: "user@example.com", Password: tt.password})
			assertAppError(t, err, tt.want)
		})
	}
}

func TestSetPassword(t *testing.T) {
	userID := uuid.New()

	t.Run("sets invited user password", func(t *testing.T) {
		store := &fakeStore{user: entity.User{ID: userID, Email: "user@example.com"}}
		got, err := newTestAuthService(store).SetPassword(context.Background(), SetPasswordParams{
			Email:           "user@example.com",
			Password:        "password123",
			ConfirmPassword: "password123",
		})
		if err != nil {
			t.Fatalf("SetPassword returned error: %v", err)
		}
		if got.Message != "PASSWORD_SET_SUCCESSFULLY" {
			t.Fatalf("message = %s, want PASSWORD_SET_SUCCESSFULLY", got.Message)
		}
		if store.setPasswordArg.ID != userID {
			t.Fatalf("set password id = %s, want %s", store.setPasswordArg.ID, userID)
		}
		if !utils.ComparePassword(store.setPasswordArg.Password.String, "password123") {
			t.Fatal("stored password hash does not match password")
		}
	})

	t.Run("rejects already set password", func(t *testing.T) {
		store := &fakeStore{user: entity.User{ID: userID, Password: sql.NullString{String: "hash", Valid: true}}}
		_, err := newTestAuthService(store).SetPassword(context.Background(), SetPasswordParams{Email: "user@example.com", Password: "password123"})
		assertAppError(t, err, "PASSWORD_ALREADY_SET")
	})
}

func TestRefreshTokenWithoutRedisRotatesTokensAndAllowsActiveToken(t *testing.T) {
	userID := uuid.New()
	svc := newTestAuthService(&fakeStore{
		user:     entity.User{ID: userID, Email: "user@example.com"},
		userByID: entity.User{ID: userID, Email: "user@example.com"},
	})
	refreshToken, err := svc.tokenSvc.GenerateRefreshToken(token_service.GenerateRefreshTokenInput{ID: userID, Email: "user@example.com"})
	if err != nil {
		t.Fatalf("GenerateRefreshToken returned error: %v", err)
	}

	got, err := svc.RefreshToken(context.Background(), RefreshTokenParams{RefreshToken: refreshToken})
	if err != nil {
		t.Fatalf("RefreshToken returned error: %v", err)
	}
	if got.AccessToken == "" || got.RefreshToken == "" || got.RefreshToken == refreshToken {
		t.Fatalf("rotated tokens = %+v, want new access and refresh token", got)
	}
}

func TestLogoutAndRevokeAllRequireRedis(t *testing.T) {
	userID := uuid.New()
	svc := newTestAuthService(&fakeStore{user: entity.User{ID: userID, Email: "user@example.com"}})
	refreshToken, err := svc.tokenSvc.GenerateRefreshToken(token_service.GenerateRefreshTokenInput{ID: userID, Email: "user@example.com"})
	if err != nil {
		t.Fatalf("GenerateRefreshToken returned error: %v", err)
	}

	_, err = svc.Logout(context.Background(), LogoutParams{RefreshToken: refreshToken})
	assertAppError(t, err, "INTERNAL_SERVER_ERROR")

	_, err = svc.RevokeAll(context.Background(), RevokeAllParams{RefreshToken: refreshToken})
	assertAppError(t, err, "INTERNAL_SERVER_ERROR")
}

func TestRefreshTokenKeyHelpersAndTTL(t *testing.T) {
	svc := newTestAuthService(&fakeStore{})
	if got, want := svc.refreshBlacklistKey("jti-1"), "test:blacklist:refresh:jti-1"; got != want {
		t.Fatalf("blacklist key = %s, want %s", got, want)
	}
	if got, want := svc.refreshUserRevokeAllKey("user-1"), "test:revoke-all:refresh:user-1"; got != want {
		t.Fatalf("revoke all key = %s, want %s", got, want)
	}
	if ttl := svc.refreshTokenTTL(&token_service.RefreshTokenClaims{}); ttl != 0 {
		t.Fatalf("ttl = %s, want 0 without expiry", ttl)
	}
}

func newTestAuthService(store entity.Store) *AuthService {
	cfg := &config.Config{
		JWT_ACCESS_TOKEN_SECRET:   "access-secret",
		JWT_REFRESH_TOKEN_SECRET:  "refresh-secret",
		JWT_ACCESS_TOKEN_EXPIRED:  time.Minute,
		JWT_REFRESH_TOKEN_EXPIRED: time.Hour,
		REDIS_PREFIX:              "test",
	}
	return NewService(store, token_service.NewService(cfg), nil, cfg)
}

func assertAppError(t *testing.T, err error, wantMessage string) {
	t.Helper()

	if err == nil {
		t.Fatalf("err = nil, want %s", wantMessage)
	}
	var appErr *errs.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("err = %T %v, want AppError", err, err)
	}
	if appErr.Message != wantMessage {
		t.Fatalf("message = %s, want %s", appErr.Message, wantMessage)
	}
}
