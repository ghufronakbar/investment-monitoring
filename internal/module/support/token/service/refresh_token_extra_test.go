package token_service

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestValidateRefreshTokenRejectsMissingRevocationClaims(t *testing.T) {
	tm := &TokenMaker{refreshSecret: []byte("test-refresh-secret")}

	tests := []struct {
		name   string
		claims jwt.RegisteredClaims
	}{
		{
			name: "missing jti",
			claims: jwt.RegisteredClaims{
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		},
		{
			name: "missing issued at",
			claims: jwt.RegisteredClaims{
				ID:        uuid.NewString(),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		},
		{
			name: "missing expires at",
			claims: jwt.RegisteredClaims{
				ID:       uuid.NewString(),
				IssuedAt: jwt.NewNumericDate(time.Now()),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenString := signRefreshToken(t, tm.refreshSecret, RefreshTokenClaims{
				ID:               uuid.New(),
				Email:            "user@example.com",
				RegisteredClaims: tt.claims,
			})

			if _, err := tm.ValidateRefreshToken(tokenString); err == nil {
				t.Fatal("expected invalid refresh token claims error")
			}
		})
	}
}

func TestValidateRefreshTokenRejectsExpiredToken(t *testing.T) {
	tm := &TokenMaker{refreshSecret: []byte("test-refresh-secret")}
	tokenString := signRefreshToken(t, tm.refreshSecret, RefreshTokenClaims{
		ID:    uuid.New(),
		Email: "user@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	})

	if _, err := tm.ValidateRefreshToken(tokenString); err == nil {
		t.Fatal("expected expired refresh token error")
	}
}

func TestRefreshDecodeTokenWithoutVerify(t *testing.T) {
	issuer := &TokenMaker{
		refreshSecret: []byte("issuer-secret"),
		refreshTTL:    time.Hour,
	}
	decoder := &TokenMaker{refreshSecret: []byte("different-secret")}
	userID := uuid.New()

	tokenString, err := issuer.GenerateRefreshToken(GenerateRefreshTokenInput{ID: userID, Email: "user@example.com"})
	if err != nil {
		t.Fatalf("GenerateRefreshToken returned error: %v", err)
	}

	claims, err := decoder.RefreshDecodeTokenWithoutVerify(tokenString)
	if err != nil {
		t.Fatalf("RefreshDecodeTokenWithoutVerify returned error: %v", err)
	}
	if claims.ID != userID || claims.Email != "user@example.com" {
		t.Fatalf("claims = %+v, want id/email", claims)
	}
}

func signRefreshToken(t *testing.T, secret []byte, claims RefreshTokenClaims) string {
	t.Helper()

	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims).SignedString(secret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tokenString
}
