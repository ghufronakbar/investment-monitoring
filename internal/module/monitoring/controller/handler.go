// internal/module/monitoring/controller/handler.go
package monitoring_controller

import (
	"net/http"

	"investment_monitoring/internal/http/internal_middleware"
	monitoring_service "investment_monitoring/internal/module/monitoring/service"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/response"
	"investment_monitoring/pkg/utils"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc   *monitoring_service.MonitoringService
	store entity.Store
}

func NewHandler(svc *monitoring_service.MonitoringService, store entity.Store) *Handler {
	return &Handler{
		svc:   svc,
		store: store,
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

	// Check access: must be EXECUTING_AGENCY ADMIN
	access, err := h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: projectId,
		UserID:    userClaims.ID,
	})
	if err != nil || getParticipantRoleString(access.ParticipantRole) != "EXECUTING_AGENCY" || access.MemberRole != entity.MemberRoleEnumADMIN {
		response.Error(w, r, errs.NewForbidden("EXECUTING_AGENCY_ADMIN_REQUIRED"), nil)
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

	// Check access: any project participant
	_, err = h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: projectId,
		UserID:    userClaims.ID,
	})
	if err != nil {
		response.Error(w, r, errs.NewForbidden("NO_PROJECT_ACCESS"), nil)
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

	// Check access: any project participant
	_, err = h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: projectId,
		UserID:    userClaims.ID,
	})
	if err != nil {
		response.Error(w, r, errs.NewForbidden("NO_PROJECT_ACCESS"), nil)
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

	// Get task to find contract/project
	task, err := h.store.GetTaskByID(r.Context(), taskId)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("TASK_NOT_FOUND"), nil)
		return
	}

	// Get contract to find project
	contract, err := h.store.GetContractByID(r.Context(), task.ContractID)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("CONTRACT_NOT_FOUND"), nil)
		return
	}

	// Check access: must be EXECUTING_AGENCY ADMIN
	access, err := h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: contract.ProjectID,
		UserID:    userClaims.ID,
	})
	if err != nil || getParticipantRoleString(access.ParticipantRole) != "EXECUTING_AGENCY" || access.MemberRole != entity.MemberRoleEnumADMIN {
		response.Error(w, r, errs.NewForbidden("EXECUTING_AGENCY_ADMIN_REQUIRED"), nil)
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

	// Get task to find contract/project
	task, err := h.store.GetTaskByID(r.Context(), taskId)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("TASK_NOT_FOUND"), nil)
		return
	}

	// Get contract to find project
	contract, err := h.store.GetContractByID(r.Context(), task.ContractID)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("CONTRACT_NOT_FOUND"), nil)
		return
	}

	// Check access: must be EXECUTING_AGENCY ADMIN
	access, err := h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: contract.ProjectID,
		UserID:    userClaims.ID,
	})
	if err != nil || getParticipantRoleString(access.ParticipantRole) != "EXECUTING_AGENCY" || access.MemberRole != entity.MemberRoleEnumADMIN {
		response.Error(w, r, errs.NewForbidden("EXECUTING_AGENCY_ADMIN_REQUIRED"), nil)
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

	// Get task to find contract/project
	task, err := h.store.GetTaskByID(r.Context(), taskId)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("TASK_NOT_FOUND"), nil)
		return
	}

	// Get contract to find project
	contract, err := h.store.GetContractByID(r.Context(), task.ContractID)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("CONTRACT_NOT_FOUND"), nil)
		return
	}

	// Check access: must be CONTRACTOR (vendor)
	access, err := h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: contract.ProjectID,
		UserID:    userClaims.ID,
	})
	if err != nil || getParticipantRoleString(access.ParticipantRole) != "CONTRACTOR" {
		response.Error(w, r, errs.NewForbidden("CONTRACTOR_REQUIRED"), nil)
		return
	}

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	req.TaskID = taskId
	req.VendorID = access.OrganizationID // From JWT context via access check

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

	// Check access: any project participant
	_, err = h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: projectId,
		UserID:    userClaims.ID,
	})
	if err != nil {
		response.Error(w, r, errs.NewForbidden("NO_PROJECT_ACCESS"), nil)
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

	// Check access: must be INDEPENDENT_AUDITOR
	access, err := h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: projectId,
		UserID:    userClaims.ID,
	})
	if err != nil || getParticipantRoleString(access.ParticipantRole) != "INDEPENDENT_AUDITOR" {
		response.Error(w, r, errs.NewForbidden("AUDITOR_REQUIRED"), nil)
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

	// Get log to find task/contract/project
	log, err := h.store.GetProgressLogWithDetails(r.Context(), logId)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("PROGRESS_LOG_NOT_FOUND"), nil)
		return
	}

	// Get contract to find project
	contract, err := h.store.GetContractByID(r.Context(), log.ContractID)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("CONTRACT_NOT_FOUND"), nil)
		return
	}

	// Check access: must be INDEPENDENT_AUDITOR
	access, err := h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: contract.ProjectID,
		UserID:    userClaims.ID,
	})
	if err != nil || getParticipantRoleString(access.ParticipantRole) != "INDEPENDENT_AUDITOR" {
		response.Error(w, r, errs.NewForbidden("AUDITOR_REQUIRED"), nil)
		return
	}

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	req.ProgressLogID = logId
	req.AuditorID = access.OrganizationID // From JWT context via access check

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

	// Check access: any project participant
	_, err = h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: projectId,
		UserID:    userClaims.ID,
	})
	if err != nil {
		response.Error(w, r, errs.NewForbidden("NO_PROJECT_ACCESS"), nil)
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

// ===== HELPER =====

// getParticipantRoleString safely converts interface{} to string
func getParticipantRoleString(role interface{}) string {
	if role == nil {
		return ""
	}
	if s, ok := role.(string); ok {
		return s
	}
	return ""
}
