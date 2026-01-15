// internal/module/support/dashboard/controller/handler.go
package dashboard_controller

import (
	"net/http"

	"investment_monitoring/internal/http/internal_middleware"
	dashboard_service "investment_monitoring/internal/module/support/dashboard/service"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	dashboardSvc *dashboard_service.DashboardService
	store        entity.Store
}

func NewHandler(dashboardSvc *dashboard_service.DashboardService, store entity.Store) *Handler {
	return &Handler{
		dashboardSvc: dashboardSvc,
		store:        store,
	}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/s-curve/{projectId}", h.GetSCurve)
	r.Get("/stats/{projectId}", h.GetStats)

	return r
}

// GetSCurve handles GET /support/dashboard/s-curve/{projectId}
// Access: Any project participant
func (h *Handler) GetSCurve(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"projectId": "Invalid UUID"}), nil)
		return
	}

	user, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	// Check access
	_, err = h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: projectID,
		UserID:    user.ID,
	})
	if err != nil {
		response.Error(w, r, errs.NewForbidden("NOT_PROJECT_PARTICIPANT"), nil)
		return
	}

	params := dashboard_service.GetSCurveParams{
		ProjectID: projectID,
	}

	result, err := h.dashboardSvc.GetSCurve(r.Context(), params)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_GET_SCURVE", result)
}

// GetStats handles GET /support/dashboard/stats/{projectId}
// Access: Any project participant
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"projectId": "Invalid UUID"}), nil)
		return
	}

	user, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	// Check access
	_, err = h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: projectID,
		UserID:    user.ID,
	})
	if err != nil {
		response.Error(w, r, errs.NewForbidden("NOT_PROJECT_PARTICIPANT"), nil)
		return
	}

	params := dashboard_service.GetStatsParams{
		ProjectID: projectID,
	}

	result, err := h.dashboardSvc.GetStats(r.Context(), params)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_GET_STATS", result)
}
