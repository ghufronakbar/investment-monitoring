// internal/module/finance/controller/handler.go
package finance_controller

import (
	"net/http"

	"investment_monitoring/internal/http/internal_middleware"
	finance_service "investment_monitoring/internal/module/finance/service"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/response"
	"investment_monitoring/pkg/utils"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	financeSvc *finance_service.FinanceService
	store      entity.Store
}

func NewHandler(financeSvc *finance_service.FinanceService, store entity.Store) *Handler {
	return &Handler{
		financeSvc: financeSvc,
		store:      store,
	}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Project-scoped routes
	r.Route("/project/{projectId}", func(r chi.Router) {
		r.Get("/request", h.GetPaymentRequests)
		r.Post("/request", h.CreatePaymentRequest)
		r.Get("/summary", h.GetPaymentSummary)
	})

	// Request-scoped routes
	r.Route("/request/{requestId}", func(r chi.Router) {
		r.Get("/", h.GetPaymentRequestByID)
		r.Put("/approve-auditor", h.ApproveByAuditor)
		r.Put("/approve-agency", h.ApproveByAgency)
		r.Put("/disburse", h.Disburse)
	})

	return r
}

// Helper to get participant role as string
func getParticipantRoleString(role interface{}) string {
	if role == nil {
		return ""
	}
	if s, ok := role.(string); ok {
		return s
	}
	return ""
}

// ===== HANDLERS =====

// CreatePaymentRequest handles POST /finance/project/{projectId}/request
// Access: CONTRACTOR only
func (h *Handler) CreatePaymentRequest(w http.ResponseWriter, r *http.Request) {
	var req finance_service.CreatePaymentRequestParams

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

	// Check access - must be CONTRACTOR
	access, err := h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: projectID,
		UserID:    user.ID,
	})
	if err != nil {
		response.Error(w, r, errs.NewForbidden("NOT_PROJECT_PARTICIPANT"), nil)
		return
	}

	role := getParticipantRoleString(access.ParticipantRole)
	if role != string(entity.OrganizationRoleEnumCONTRACTOR) {
		response.Error(w, r, errs.NewForbidden("CONTRACTOR_REQUIRED"), nil)
		return
	}

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	req.ProjectID = projectID
	req.ContractorID = access.OrganizationID
	req.UserID = user.ID

	result, err := h.financeSvc.CreatePaymentRequest(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_CREATE_PAYMENT_REQUEST", result)
}

// GetPaymentRequests handles GET /finance/project/{projectId}/request
// Access: Any project participant
func (h *Handler) GetPaymentRequests(w http.ResponseWriter, r *http.Request) {
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

	params := finance_service.GetPaymentRequestsParams{
		ProjectID: projectID,
	}

	result, err := h.financeSvc.GetPaymentRequests(r.Context(), params)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_GET_PAYMENT_REQUESTS", result)
}

// GetPaymentRequestByID handles GET /finance/request/{requestId}
// Access: Any project participant
func (h *Handler) GetPaymentRequestByID(w http.ResponseWriter, r *http.Request) {
	requestIDStr := chi.URLParam(r, "requestId")
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"requestId": "Invalid UUID"}), nil)
		return
	}

	user, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	// Get request to check contract
	pr, err := h.store.GetPaymentRequestWithContract(r.Context(), requestID)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("PAYMENT_REQUEST_NOT_FOUND"), nil)
		return
	}

	// Check access via contract's project
	_, err = h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: pr.ProjectID,
		UserID:    user.ID,
	})
	if err != nil {
		response.Error(w, r, errs.NewForbidden("NOT_PROJECT_PARTICIPANT"), nil)
		return
	}

	params := finance_service.GetPaymentRequestByIDParams{
		RequestID: requestID,
	}

	result, err := h.financeSvc.GetPaymentRequestByID(r.Context(), params)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_GET_PAYMENT_REQUEST", result)
}

// ApproveByAuditor handles PUT /finance/request/{requestId}/approve-auditor
// Access: INDEPENDENT_AUDITOR only
func (h *Handler) ApproveByAuditor(w http.ResponseWriter, r *http.Request) {
	var req finance_service.ApprovalParams

	requestIDStr := chi.URLParam(r, "requestId")
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"requestId": "Invalid UUID"}), nil)
		return
	}

	user, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	// Get request to check contract
	pr, err := h.store.GetPaymentRequestWithContract(r.Context(), requestID)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("PAYMENT_REQUEST_NOT_FOUND"), nil)
		return
	}

	// Check access
	access, err := h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: pr.ProjectID,
		UserID:    user.ID,
	})
	if err != nil {
		response.Error(w, r, errs.NewForbidden("NOT_PROJECT_PARTICIPANT"), nil)
		return
	}

	role := getParticipantRoleString(access.ParticipantRole)
	if role != string(entity.OrganizationRoleEnumINDEPENDENTAUDITOR) {
		response.Error(w, r, errs.NewForbidden("AUDITOR_REQUIRED"), nil)
		return
	}

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	req.RequestID = requestID
	req.OrgID = access.OrganizationID
	req.UserID = user.ID

	result, err := h.financeSvc.ApproveByAuditor(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_APPROVE_BY_AUDITOR", result)
}

// ApproveByAgency handles PUT /finance/request/{requestId}/approve-agency
// Access: EXECUTING_AGENCY only
func (h *Handler) ApproveByAgency(w http.ResponseWriter, r *http.Request) {
	var req finance_service.ApprovalParams

	requestIDStr := chi.URLParam(r, "requestId")
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"requestId": "Invalid UUID"}), nil)
		return
	}

	user, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	// Get request to check contract
	pr, err := h.store.GetPaymentRequestWithContract(r.Context(), requestID)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("PAYMENT_REQUEST_NOT_FOUND"), nil)
		return
	}

	// Check access
	access, err := h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: pr.ProjectID,
		UserID:    user.ID,
	})
	if err != nil {
		response.Error(w, r, errs.NewForbidden("NOT_PROJECT_PARTICIPANT"), nil)
		return
	}

	role := getParticipantRoleString(access.ParticipantRole)
	if role != string(entity.OrganizationRoleEnumEXECUTINGAGENCY) {
		response.Error(w, r, errs.NewForbidden("EXECUTING_AGENCY_REQUIRED"), nil)
		return
	}

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	req.RequestID = requestID
	req.OrgID = access.OrganizationID
	req.UserID = user.ID

	result, err := h.financeSvc.ApproveByAgency(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_APPROVE_BY_AGENCY", result)
}

// Disburse handles PUT /finance/request/{requestId}/disburse
// Access: LOAN_PROVIDER only
func (h *Handler) Disburse(w http.ResponseWriter, r *http.Request) {
	var req finance_service.DisburseParams

	requestIDStr := chi.URLParam(r, "requestId")
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"requestId": "Invalid UUID"}), nil)
		return
	}

	user, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	// Get request to check contract
	pr, err := h.store.GetPaymentRequestWithContract(r.Context(), requestID)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("PAYMENT_REQUEST_NOT_FOUND"), nil)
		return
	}

	// Check access
	access, err := h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: pr.ProjectID,
		UserID:    user.ID,
	})
	if err != nil {
		response.Error(w, r, errs.NewForbidden("NOT_PROJECT_PARTICIPANT"), nil)
		return
	}

	role := getParticipantRoleString(access.ParticipantRole)
	if role != string(entity.OrganizationRoleEnumLOANPROVIDER) {
		response.Error(w, r, errs.NewForbidden("LOAN_PROVIDER_REQUIRED"), nil)
		return
	}

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	req.RequestID = requestID
	req.OrgID = access.OrganizationID
	req.UserID = user.ID

	result, err := h.financeSvc.Disburse(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_DISBURSE", result)
}

// GetPaymentSummary handles GET /finance/project/{projectId}/summary
// Access: Any project participant
func (h *Handler) GetPaymentSummary(w http.ResponseWriter, r *http.Request) {
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

	params := finance_service.GetPaymentSummaryParams{
		ProjectID: projectID,
	}

	result, err := h.financeSvc.GetPaymentSummary(r.Context(), params)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_GET_PAYMENT_SUMMARY", result)
}
