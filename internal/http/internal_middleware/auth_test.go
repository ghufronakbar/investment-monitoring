package internal_middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"investment_monitoring/config"
	token_service "investment_monitoring/internal/module/support/token/service"

	"github.com/google/uuid"
)

func TestAuthMiddlewareAcceptsBearerTokenAndInjectsClaims(t *testing.T) {
	tokenMaker := newTokenMaker()
	userID := uuid.New()
	token, err := tokenMaker.GenerateAccessToken(token_service.GenerateAccessTokenInput{ID: userID, Email: "user@example.com"})
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}

	handler := AuthMiddleware(*tokenMaker)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := GetUserFromContext(r.Context())
		if err != nil {
			t.Fatalf("GetUserFromContext returned error: %v", err)
		}
		if claims.ID != userID || claims.Email != "user@example.com" {
			t.Fatalf("claims = %+v, want user claims", claims)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", rr.Code, rr.Body.String())
	}
}

func TestAuthMiddlewareRejectsMissingAndInvalidToken(t *testing.T) {
	handler := AuthMiddleware(*newTokenMaker())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	tests := []struct {
		name       string
		headerName string
		token      string
		wantStatus int
	}{
		{name: "missing", wantStatus: http.StatusUnauthorized},
		{name: "invalid", headerName: "Authorization", token: "Bearer bad-token", wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.headerName != "" {
				req.Header.Set(tt.headerName, tt.token)
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

func TestExtractTokenSources(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?postmaticAccessToken=query-token", nil)
	if got := extractToken(req); got != "query-token" {
		t.Fatalf("query token = %s, want query-token", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Postmatic-AccessToken", "header-token")
	if got := extractToken(req); got != "header-token" {
		t.Fatalf("header token = %s, want header-token", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "bearer bearer-token")
	if got := extractToken(req); got != "bearer-token" {
		t.Fatalf("bearer token = %s, want bearer-token", got)
	}
}

func TestGetUserFromContextRejectsMissingClaims(t *testing.T) {
	if _, err := GetUserFromContext(context.Background()); err == nil {
		t.Fatal("expected missing auth context error")
	}
}

func newTokenMaker() *token_service.TokenMaker {
	return token_service.NewService(&config.Config{
		JWT_ACCESS_TOKEN_SECRET:  "access-secret",
		JWT_REFRESH_TOKEN_SECRET: "refresh-secret",
		JWT_ACCESS_TOKEN_EXPIRED: time.Hour,
	})
}
