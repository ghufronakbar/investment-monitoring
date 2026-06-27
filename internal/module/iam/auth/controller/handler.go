// internal/module/iam/auth/controller/handler.go
package auth_controller

import (
	"context"
	"net/http"

	auth_service "investment_monitoring/internal/module/iam/auth/service"
	"investment_monitoring/pkg/response"
	"investment_monitoring/pkg/utils"

	"github.com/go-chi/chi/v5"
)

type service interface {
	CheckEmail(context.Context, auth_service.CheckEmailParams) (auth_service.CheckEmailReturn, error)
	Login(context.Context, auth_service.LoginParams) (auth_service.LoginReturn, error)
	SetPassword(context.Context, auth_service.SetPasswordParams) (auth_service.SetPasswordReturn, error)
	RefreshToken(context.Context, auth_service.RefreshTokenParams) (auth_service.RefreshTokenReturn, error)
	Logout(context.Context, auth_service.LogoutParams) (auth_service.LogoutReturn, error)
	RevokeAll(context.Context, auth_service.RevokeAllParams) (auth_service.RevokeAllReturn, error)
}

type Handler struct {
	svc service
}

func NewHandler(svc service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/check-email", h.CheckEmail)
	r.Post("/login", h.Login)
	r.Post("/set-password", h.SetPassword)
	r.Post("/refresh-token", h.RefreshToken)
	r.Post("/logout", h.Logout)
	r.Post("/revoke-all", h.RevokeAll)

	return r
}

// CheckEmail handles POST /iam/auth/check-email
func (h *Handler) CheckEmail(w http.ResponseWriter, r *http.Request) {
	var req auth_service.CheckEmailParams

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	res, err := h.svc.CheckEmail(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_CHECK_EMAIL", res)
}

// Login handles POST /iam/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req auth_service.LoginParams

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	res, err := h.svc.Login(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_LOGIN", res)
}

// SetPassword handles POST /iam/auth/set-password
func (h *Handler) SetPassword(w http.ResponseWriter, r *http.Request) {
	var req auth_service.SetPasswordParams

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	res, err := h.svc.SetPassword(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_SET_PASSWORD", res)
}

// RefreshToken handles POST /iam/auth/refresh-token
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req auth_service.RefreshTokenParams

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	res, err := h.svc.RefreshToken(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_REFRESH_TOKEN", res)
}

// Logout handles POST /iam/auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req auth_service.LogoutParams

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	res, err := h.svc.Logout(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_LOGOUT", res)
}

// RevokeAll handles POST /iam/auth/revoke-all
func (h *Handler) RevokeAll(w http.ResponseWriter, r *http.Request) {
	var req auth_service.RevokeAllParams

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	res, err := h.svc.RevokeAll(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_REVOKE_ALL_REFRESH_TOKENS", res)
}
