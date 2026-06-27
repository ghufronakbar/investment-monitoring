// internal/module/headless/token/token_service/refresh_token.go
package token_service

import (
	"investment_monitoring/pkg/errs"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type RefreshTokenClaims struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	jwt.RegisteredClaims
}

type GenerateRefreshTokenInput struct {
	ID    uuid.UUID
	Email string
}

func (tm *TokenMaker) GenerateRefreshToken(input GenerateRefreshTokenInput) (string, error) {
	expirationTime := time.Now().Add(tm.refreshTTL)
	tokenID := uuid.NewString()
	claims := &RefreshTokenClaims{
		ID:    input.ID,
		Email: input.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        tokenID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(tm.refreshSecret)
}

func (tm *TokenMaker) ValidateRefreshToken(tokenString string) (*RefreshTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return tm.refreshSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errs.NewBadRequest("INVALID_REFRESH_TOKEN")
	}
	claims := token.Claims.(*RefreshTokenClaims)
	if claims.RegisteredClaims.ID == "" || claims.IssuedAt == nil || claims.ExpiresAt == nil {
		return nil, errs.NewBadRequest("INVALID_REFRESH_TOKEN_CLAIMS")
	}
	return claims, nil
}

func (tm *TokenMaker) RefreshDecodeTokenWithoutVerify(tokenString string) (*RefreshTokenClaims, error) {
	parser := jwt.NewParser(
		jwt.WithoutClaimsValidation(),
	)

	claims := &RefreshTokenClaims{}
	_, _, err := parser.ParseUnverified(tokenString, claims)
	if err != nil {
		return nil, err
	}

	return claims, nil
}
