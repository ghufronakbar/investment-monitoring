// internal/module/finance/controller/handler.go
package finance_controller

import (
	"context"
	"net/http"

	"investment_monitoring/internal/http/internal_middleware"
	finance_service "investment_monitoring/internal/module/finance/service"
	"investment_monitoring/internal/policy"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/response"
	"investment_monitoring/pkg/utils"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type service interface {
	CreatePaymentRequest(context.Context, finance_service.CreatePaymentRequestParams) (finance_service.PaymentRequestReturn, error)
	GetPaymentRequests(context.Context, finance_service.GetPaymentRequestsParams) ([]finance_service.PaymentRequestReturn, error)
	GetPaymentRequestByID(context.Context, finance_service.GetPaymentRequestByIDParams) (finance_service.PaymentRequestReturn, error)
	ApproveByAuditor(context.Context, finance_service.ApprovalParams) (finance_service.PaymentRequestReturn, error)
	ApproveByAgency(context.Context, finance_service.ApprovalParams) (finance_service.PaymentRequestReturn, error)
	Disburse(context.Context, finance_service.DisburseParams) (finance_service.PaymentRequestReturn, error)
	GetPaymentSummary(context.Context, finance_service.GetPaymentSummaryParams) (finance_service.PaymentSummaryReturn, error)
}

type Handler struct {
	financeSvc service
	policy     *policy.Service
}

func NewHandler(financeSvc service, store entity.Store) *Handler {
	return &Handler{
		financeSvc: financeSvc,
		policy:     policy.NewService(store),
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

	access, err := h.policy.CanCreatePaymentRequest(r.Context(), user.ID, projectID)
	if err != nil {
		response.Error(w, r, err, nil)
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

	if err := h.policy.CanViewFinanceProject(r.Context(), user.ID, projectID); err != nil {
		response.Error(w, r, err, nil)
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

	if err := h.policy.CanViewPaymentRequest(r.Context(), user.ID, requestID); err != nil {
		response.Error(w, r, err, nil)
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

	access, err := h.policy.CanApprovePaymentAsAuditor(r.Context(), user.ID, requestID)
	if err != nil {
		response.Error(w, r, err, nil)
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

	access, err := h.policy.CanApprovePaymentAsAgency(r.Context(), user.ID, requestID)
	if err != nil {
		response.Error(w, r, err, nil)
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

	access, err := h.policy.CanDisbursePayment(r.Context(), user.ID, requestID)
	if err != nil {
		response.Error(w, r, err, nil)
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

	if err := h.policy.CanViewFinanceProject(r.Context(), user.ID, projectID); err != nil {
		response.Error(w, r, err, nil)
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
