// internal/http/internal_middleware/auth.go
package internal_middleware

import (
	"context"
	"net/http"
	"strings"

	token_service "investment_monitoring/internal/module/support/token/service"
	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/response"
)

type contextKey string

const UserContextKey contextKey = "userClaims"

func AuthMiddleware(tm token_service.TokenMaker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractToken(r)
			if tokenStr == "" {
				response.Error(w, r, errs.NewUnauthorized("MISSING_TOKEN"), nil)
				return
			}

			claims, err := tm.ValidateAccessToken(tokenStr)
			if err != nil {
				response.Error(w, r, errs.NewUnauthorized("INVALID_OR_EXPIRED_TOKEN"), nil)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractToken(r *http.Request) string {
	if q := r.URL.Query().Get("postmaticAccessToken"); q != "" {
		return q
	}
	if h := r.Header.Get("X-Postmatic-AccessToken"); h != "" {
		return h
	}
	auth := r.Header.Get("Authorization")
	parts := strings.Split(auth, " ")
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

func GetUserFromContext(ctx context.Context) (*token_service.AccessTokenClaims, error) {
	claims, ok := ctx.Value(UserContextKey).(*token_service.AccessTokenClaims)
	if !ok || claims == nil {
		// lebih cocok 401/forbidden daripada internal error
		return nil, errs.NewUnauthorized("MISSING_AUTH_CONTEXT")
	}
	return claims, nil
}
