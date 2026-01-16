// internal/module/iam/profile/controller/handler.go
package profile_controller

import (
	"net/http"

	"investment_monitoring/internal/http/internal_middleware"
	profile_service "investment_monitoring/internal/module/iam/profile/service"
	"investment_monitoring/pkg/response"
	"investment_monitoring/pkg/utils"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *profile_service.ProfileService
}

func NewHandler(svc *profile_service.ProfileService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.GetProfile)
	r.Put("/", h.UpdateProfile)
	r.Put("/password", h.ChangePassword)

	return r
}

// GetProfile handles GET /iam/profile
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userClaims, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	res, err := h.svc.GetProfile(r.Context(), userClaims.ID)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_GET_PROFILE", res)
}

// UpdateProfile handles PUT /iam/profile
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userClaims, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	var req profile_service.UpdateProfileParams
	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	res, err := h.svc.UpdateProfile(r.Context(), userClaims.ID, req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_UPDATE_PROFILE", res)
}

// ChangePassword handles PUT /iam/profile/password
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userClaims, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	var req profile_service.ChangePasswordParams
	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	res, err := h.svc.ChangePassword(r.Context(), userClaims.ID, req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_CHANGE_PASSWORD", res)
}
