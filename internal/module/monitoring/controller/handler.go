// internal/module/monitoring/controller/handler.go
package monitoring_controller

import (
	"net/http"

	"investment_monitoring/internal/http/internal_middleware"
	monitoring_service "investment_monitoring/internal/module/monitoring/service"
	"investment_monitoring/internal/policy"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/response"
	"investment_monitoring/pkg/utils"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc    *monitoring_service.MonitoringService
	policy *policy.Service
}

func NewHandler(svc *monitoring_service.MonitoringService, store entity.Store) *Handler {
	return &Handler{
		svc:    svc,
		policy: policy.NewService(store),
	}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Project-scoped routes
	r.Route("/project/{projectId}", func(r chi.Router) {
		// Task routes
		r.Get("/task", h.GetTasks)
		r.Post("/task", h.CreateTask)
		r.Get("/task/leaf", h.GetLeafTasks)
		r.Get("/wbs/validation", h.ValidateWBS)

		// Progress log routes
		r.Get("/log", h.GetProgressLogs)
		r.Get("/log/pending", h.GetPendingValidations)

		// S-Curve data
		r.Get("/scurve", h.GetSCurve)
	})

	// Task-specific routes
	r.Route("/task/{taskId}", func(r chi.Router) {
		r.Get("/", h.GetTaskByID)
		r.Put("/", h.UpdateTask)
		r.Delete("/", h.DeleteTask)
		r.Post("/log", h.SubmitProgress)
	})

	// Log-specific routes
	r.Route("/log/{logId}", func(r chi.Router) {
		r.Get("/", h.GetProgressLogByID)
		r.Post("/validate", h.ValidateProgress)
	})

	return r
}

// ===== TASK HANDLERS =====

// CreateTask handles POST /monitoring/project/{projectId}/task
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req monitoring_service.CreateTaskParams

	userClaims, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	projectIdStr := chi.URLParam(r, "projectId")
	projectId, err := uuid.Parse(projectIdStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"projectId": "Invalid UUID"}), nil)
		return
	}

	if err := h.policy.CanConfigureProject(r.Context(), userClaims.ID, projectId); err != nil {
		response.Error(w, r, err, nil)
		return
	}

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	req.ProjectID = projectId

	res, err := h.svc.CreateTask(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_CREATE_TASK", res)
}

// GetTasks handles GET /monitoring/project/{projectId}/task
func (h *Handler) GetTasks(w http.ResponseWriter, r *http.Request) {
	userClaims, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	projectIdStr := chi.URLParam(r, "projectId")
	projectId, err := uuid.Parse(projectIdStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"projectId": "Invalid UUID"}), nil)
		return
	}

	if err := h.policy.CanViewMonitoringProject(r.Context(), userClaims.ID, projectId); err != nil {
		response.Error(w, r, err, nil)
		return
	}

	res, err := h.svc.GetTasks(r.Context(), monitoring_service.GetTasksParams{
		ProjectID: projectId,
		UserID:    userClaims.ID,
	})
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_GET_TASKS", res)
}

// GetLeafTasks handles GET /monitoring/project/{projectId}/task/leaf
func (h *Handler) GetLeafTasks(w http.ResponseWriter, r *http.Request) {
	userClaims, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	projectIdStr := chi.URLParam(r, "projectId")
	projectId, err := uuid.Parse(projectIdStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"projectId": "Invalid UUID"}), nil)
		return
	}

	if err := h.policy.CanViewMonitoringProject(r.Context(), userClaims.ID, projectId); err != nil {
		response.Error(w, r, err, nil)
		return
	}

	res, err := h.svc.GetLeafTasks(r.Context(), monitoring_service.GetTasksParams{
		ProjectID: projectId,
		UserID:    userClaims.ID,
	})
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_GET_LEAF_TASKS", res)
}

// ValidateWBS handles GET /monitoring/project/{projectId}/wbs/validation
func (h *Handler) ValidateWBS(w http.ResponseWriter, r *http.Request) {
	userClaims, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	projectIdStr := chi.URLParam(r, "projectId")
	projectId, err := uuid.Parse(projectIdStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"projectId": "Invalid UUID"}), nil)
		return
	}

	if err := h.policy.CanViewMonitoringProject(r.Context(), userClaims.ID, projectId); err != nil {
		response.Error(w, r, err, nil)
		return
	}

	res, err := h.svc.ValidateWBS(r.Context(), monitoring_service.ValidateWBSParams{
		ProjectID: projectId,
		UserID:    userClaims.ID,
	})
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_VALIDATE_WBS", res)
}

// GetTaskByID handles GET /monitoring/task/{taskId}
func (h *Handler) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	// For simplicity, just return task not implemented
	// In production, would need to verify project access via task's contract
	response.Error(w, r, errs.NewBadRequest("NOT_IMPLEMENTED"), nil)
}

// UpdateTask handles PUT /monitoring/task/{taskId}
func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	var req monitoring_service.UpdateTaskParams

	userClaims, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	taskIdStr := chi.URLParam(r, "taskId")
	taskId, err := uuid.Parse(taskIdStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"taskId": "Invalid UUID"}), nil)
		return
	}

	if err := h.policy.CanManageTask(r.Context(), userClaims.ID, taskId); err != nil {
		response.Error(w, r, err, nil)
		return
	}

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	req.TaskID = taskId

	res, err := h.svc.UpdateTask(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_UPDATE_TASK", res)
}

// DeleteTask handles DELETE /monitoring/task/{taskId}
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	userClaims, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	taskIdStr := chi.URLParam(r, "taskId")
	taskId, err := uuid.Parse(taskIdStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"taskId": "Invalid UUID"}), nil)
		return
	}

	if err := h.policy.CanManageTask(r.Context(), userClaims.ID, taskId); err != nil {
		response.Error(w, r, err, nil)
		return
	}

	err = h.svc.DeleteTask(r.Context(), monitoring_service.DeleteTaskParams{TaskID: taskId})
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_DELETE_TASK", nil)
}

// ===== PROGRESS LOG HANDLERS =====

// SubmitProgress handles POST /monitoring/task/{taskId}/log
func (h *Handler) SubmitProgress(w http.ResponseWriter, r *http.Request) {
	var req monitoring_service.SubmitProgressParams

	userClaims, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	taskIdStr := chi.URLParam(r, "taskId")
	taskId, err := uuid.Parse(taskIdStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"taskId": "Invalid UUID"}), nil)
		return
	}

	access, err := h.policy.CanSubmitProgress(r.Context(), userClaims.ID, taskId)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	req.TaskID = taskId
	req.VendorID = access.OrganizationID // From JWT context via access check
	req.UserID = userClaims.ID

	res, err := h.svc.SubmitProgress(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_SUBMIT_PROGRESS", res)
}

// GetProgressLogs handles GET /monitoring/project/{projectId}/log
func (h *Handler) GetProgressLogs(w http.ResponseWriter, r *http.Request) {
	userClaims, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	projectIdStr := chi.URLParam(r, "projectId")
	projectId, err := uuid.Parse(projectIdStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"projectId": "Invalid UUID"}), nil)
		return
	}

	if err := h.policy.CanViewMonitoringProject(r.Context(), userClaims.ID, projectId); err != nil {
		response.Error(w, r, err, nil)
		return
	}

	res, err := h.svc.GetProgressLogs(r.Context(), monitoring_service.GetProgressLogsParams{
		ProjectID: projectId,
		UserID:    userClaims.ID,
	})
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_GET_PROGRESS_LOGS", res)
}

// GetPendingValidations handles GET /monitoring/project/{projectId}/log/pending
func (h *Handler) GetPendingValidations(w http.ResponseWriter, r *http.Request) {
	userClaims, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	projectIdStr := chi.URLParam(r, "projectId")
	projectId, err := uuid.Parse(projectIdStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"projectId": "Invalid UUID"}), nil)
		return
	}

	if err := h.policy.CanViewPendingValidations(r.Context(), userClaims.ID, projectId); err != nil {
		response.Error(w, r, err, nil)
		return
	}

	res, err := h.svc.GetPendingValidations(r.Context(), monitoring_service.GetProgressLogsParams{
		ProjectID: projectId,
		UserID:    userClaims.ID,
	})
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_GET_PENDING_VALIDATIONS", res)
}

// GetProgressLogByID handles GET /monitoring/log/{logId}
func (h *Handler) GetProgressLogByID(w http.ResponseWriter, r *http.Request) {
	// For simplicity, return not implemented
	response.Error(w, r, errs.NewBadRequest("NOT_IMPLEMENTED"), nil)
}

// ===== VALIDATION HANDLERS =====

// ValidateProgress handles POST /monitoring/log/{logId}/validate
func (h *Handler) ValidateProgress(w http.ResponseWriter, r *http.Request) {
	var req monitoring_service.ValidateProgressParams

	userClaims, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	logIdStr := chi.URLParam(r, "logId")
	logId, err := uuid.Parse(logIdStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"logId": "Invalid UUID"}), nil)
		return
	}

	access, err := h.policy.CanValidateProgress(r.Context(), userClaims.ID, logId)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	req.ProgressLogID = logId
	req.AuditorID = access.OrganizationID // From JWT context via access check
	req.UserID = userClaims.ID

	res, err := h.svc.ValidateProgress(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_VALIDATE_PROGRESS", res)
}

// ===== S-CURVE HANDLERS =====

// GetSCurve handles GET /monitoring/project/{projectId}/scurve
func (h *Handler) GetSCurve(w http.ResponseWriter, r *http.Request) {
	userClaims, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	projectIdStr := chi.URLParam(r, "projectId")
	projectId, err := uuid.Parse(projectIdStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"projectId": "Invalid UUID"}), nil)
		return
	}

	if err := h.policy.CanViewMonitoringProject(r.Context(), userClaims.ID, projectId); err != nil {
		response.Error(w, r, err, nil)
		return
	}

	res, err := h.svc.GetSCurve(r.Context(), monitoring_service.GetSCurveParams{
		ProjectID: projectId,
		UserID:    userClaims.ID,
	})
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_GET_SCURVE", res)
}
