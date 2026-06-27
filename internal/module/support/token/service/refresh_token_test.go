package token_service

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRefreshTokenIncludesRevocationClaims(t *testing.T) {
	tm := &TokenMaker{
		refreshSecret: []byte("test-refresh-secret"),
		refreshTTL:    time.Hour,
	}

	userID := uuid.New()
	token, err := tm.GenerateRefreshToken(GenerateRefreshTokenInput{
		ID:    userID,
		Email: "user@example.com",
	})
	if err != nil {
		t.Fatalf("GenerateRefreshToken returned error: %v", err)
	}

	claims, err := tm.ValidateRefreshToken(token)
	if err != nil {
		t.Fatalf("ValidateRefreshToken returned error: %v", err)
	}

	if claims.ID != userID {
		t.Fatalf("expected user id %s, got %s", userID, claims.ID)
	}
	if claims.RegisteredClaims.ID == "" {
		t.Fatal("expected refresh token jti to be set")
	}
	if claims.IssuedAt == nil {
		t.Fatal("expected refresh token issued_at to be set")
	}
	if claims.ExpiresAt == nil {
		t.Fatal("expected refresh token expires_at to be set")
	}
}
