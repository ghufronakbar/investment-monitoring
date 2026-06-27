package auth_controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	auth_service "investment_monitoring/internal/module/iam/auth/service"
	"investment_monitoring/pkg/errs"

	"github.com/google/uuid"
)

type fakeAuthService struct {
	checkEmailFn func(context.Context, auth_service.CheckEmailParams) (auth_service.CheckEmailReturn, error)
	loginFn      func(context.Context, auth_service.LoginParams) (auth_service.LoginReturn, error)
	setPassFn    func(context.Context, auth_service.SetPasswordParams) (auth_service.SetPasswordReturn, error)
	refreshFn    func(context.Context, auth_service.RefreshTokenParams) (auth_service.RefreshTokenReturn, error)
	logoutFn     func(context.Context, auth_service.LogoutParams) (auth_service.LogoutReturn, error)
	revokeAllFn  func(context.Context, auth_service.RevokeAllParams) (auth_service.RevokeAllReturn, error)
}

func (f fakeAuthService) CheckEmail(ctx context.Context, p auth_service.CheckEmailParams) (auth_service.CheckEmailReturn, error) {
	return f.checkEmailFn(ctx, p)
}

func (f fakeAuthService) Login(ctx context.Context, p auth_service.LoginParams) (auth_service.LoginReturn, error) {
	return f.loginFn(ctx, p)
}

func (f fakeAuthService) SetPassword(ctx context.Context, p auth_service.SetPasswordParams) (auth_service.SetPasswordReturn, error) {
	return f.setPassFn(ctx, p)
}

func (f fakeAuthService) RefreshToken(ctx context.Context, p auth_service.RefreshTokenParams) (auth_service.RefreshTokenReturn, error) {
	return f.refreshFn(ctx, p)
}

func (f fakeAuthService) Logout(ctx context.Context, p auth_service.LogoutParams) (auth_service.LogoutReturn, error) {
	return f.logoutFn(ctx, p)
}

func (f fakeAuthService) RevokeAll(ctx context.Context, p auth_service.RevokeAllParams) (auth_service.RevokeAllReturn, error) {
	return f.revokeAllFn(ctx, p)
}

func TestCheckEmailContract(t *testing.T) {
	handler := NewHandler(fakeAuthService{
		checkEmailFn: func(ctx context.Context, p auth_service.CheckEmailParams) (auth_service.CheckEmailReturn, error) {
			if p.Email != "user@example.com" {
				t.Fatalf("email = %s, want user@example.com", p.Email)
			}
			return auth_service.CheckEmailReturn{Status: auth_service.EmailStatusLoginReady}, nil
		},
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/check-email", strings.NewReader(`{"email":"user@example.com"}`))
	handler.Routes().ServeHTTP(rr, req)

	assertResponse(t, rr, http.StatusOK, "SUCCESS_CHECK_EMAIL")
	var body responseBody
	decodeBody(t, rr, &body)
	data := body.Data.(map[string]any)
	if data["status"] != string(auth_service.EmailStatusLoginReady) {
		t.Fatalf("status = %v, want LOGIN_READY", data["status"])
	}
}

func TestLoginContract(t *testing.T) {
	userID := uuid.New()
	handler := NewHandler(fakeAuthService{
		loginFn: func(ctx context.Context, p auth_service.LoginParams) (auth_service.LoginReturn, error) {
			if p.Email != "user@example.com" || p.Password != "password123" {
				t.Fatalf("params = %+v, want email/password", p)
			}
			return auth_service.LoginReturn{
				AccessToken:  "access",
				RefreshToken: "refresh",
				User: auth_service.UserReturn{
					ID:    userID,
					Name:  "User",
					Email: "user@example.com",
				},
				Memberships: []auth_service.Membership{},
			}, nil
		},
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"user@example.com","password":"password123"}`))
	handler.Routes().ServeHTTP(rr, req)

	assertResponse(t, rr, http.StatusOK, "SUCCESS_LOGIN")
	var body responseBody
	decodeBody(t, rr, &body)
	data := body.Data.(map[string]any)
	if data["accessToken"] != "access" || data["refreshToken"] != "refresh" {
		t.Fatalf("data = %+v, want tokens", data)
	}
	if memberships, ok := data["memberships"].([]any); !ok || len(memberships) != 0 {
		t.Fatalf("memberships = %#v, want empty array", data["memberships"])
	}
}

func TestAuthControllerValidationAndServiceErrorContracts(t *testing.T) {
	handler := NewHandler(fakeAuthService{
		loginFn: func(ctx context.Context, p auth_service.LoginParams) (auth_service.LoginReturn, error) {
			return auth_service.LoginReturn{}, errs.NewUnauthorized("INVALID_CREDENTIALS")
		},
	})

	t.Run("validation failed", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"bad","password":"short"}`))
		handler.Routes().ServeHTTP(rr, req)

		assertResponse(t, rr, http.StatusBadRequest, "VALIDATION_FAILED")
		var body responseBody
		decodeBody(t, rr, &body)
		if body.ValidationErrors["email"] == "" || body.ValidationErrors["password"] == "" {
			t.Fatalf("validation errors = %+v, want email and password", body.ValidationErrors)
		}
	})

	t.Run("service error", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"user@example.com","password":"password123"}`))
		handler.Routes().ServeHTTP(rr, req)

		assertResponse(t, rr, http.StatusUnauthorized, "INVALID_CREDENTIALS")
	})
}

type responseBody struct {
	ResponseMessage  string            `json:"responseMessage"`
	Data             any               `json:"data"`
	ValidationErrors map[string]string `json:"validationErrors"`
}

func assertResponse(t *testing.T, rr *httptest.ResponseRecorder, wantStatus int, wantMessage string) {
	t.Helper()
	if rr.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, wantStatus, rr.Body.String())
	}
	var body responseBody
	decodeBody(t, rr, &body)
	if body.ResponseMessage != wantMessage {
		t.Fatalf("response message = %s, want %s", body.ResponseMessage, wantMessage)
	}
}

func decodeBody(t *testing.T, rr *httptest.ResponseRecorder, dest any) {
	t.Helper()
	if err := json.Unmarshal(rr.Body.Bytes(), dest); err != nil {
		t.Fatalf("decode body: %v; body=%s", err, rr.Body.String())
	}
}
