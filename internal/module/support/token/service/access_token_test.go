package token_service

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestAccessTokenGenerateValidateAndDecode(t *testing.T) {
	tm := &TokenMaker{
		accessSecret: []byte("test-access-secret"),
		accessTTL:    time.Hour,
	}

	userID := uuid.New()
	token, err := tm.GenerateAccessToken(GenerateAccessTokenInput{
		ID:    userID,
		Email: "user@example.com",
	})
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}

	claims, err := tm.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken returned error: %v", err)
	}
	if claims.ID != userID {
		t.Fatalf("id = %s, want %s", claims.ID, userID)
	}
	if claims.Email != "user@example.com" {
		t.Fatalf("email = %s, want user@example.com", claims.Email)
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.After(time.Now()) {
		t.Fatalf("expires at = %v, want future expiry", claims.ExpiresAt)
	}

	decoded, err := tm.AccessDecodeTokenWithoutVerify(token)
	if err != nil {
		t.Fatalf("AccessDecodeTokenWithoutVerify returned error: %v", err)
	}
	if decoded.ID != userID || decoded.Email != "user@example.com" {
		t.Fatalf("decoded claims = %+v, want id/email", decoded)
	}
}

func TestValidateAccessTokenRejectsInvalidSignature(t *testing.T) {
	issuer := &TokenMaker{
		accessSecret: []byte("issuer-secret"),
		accessTTL:    time.Hour,
	}
	validator := &TokenMaker{
		accessSecret: []byte("validator-secret"),
		accessTTL:    time.Hour,
	}

	token, err := issuer.GenerateAccessToken(GenerateAccessTokenInput{ID: uuid.New(), Email: "user@example.com"})
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}

	if _, err := validator.ValidateAccessToken(token); err == nil {
		t.Fatal("expected invalid signature error")
	}
}

func TestValidateAccessTokenRejectsExpiredToken(t *testing.T) {
	tm := &TokenMaker{accessSecret: []byte("test-access-secret")}
	claims := &AccessTokenClaims{
		ID:    uuid.New(),
		Email: "user@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(tm.accessSecret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := tm.ValidateAccessToken(token); err == nil {
		t.Fatal("expected expired token error")
	}
}
