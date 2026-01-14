// internal/module/tracker/controller/handler.go
package tracker_controller

import (
	"net/http"

	"investment_monitoring/internal/http/internal_middleware"
	tracker_service "investment_monitoring/internal/module/tracker/service"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/response"
	"investment_monitoring/pkg/utils"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	trackerSvc *tracker_service.TrackerService
	store      entity.Store
}

func NewHandler(trackerSvc *tracker_service.TrackerService, store entity.Store) *Handler {
	return &Handler{
		trackerSvc: trackerSvc,
		store:      store,
	}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Project-scoped routes
	r.Route("/project/{projectId}", func(r chi.Router) {
		r.Get("/ticket", h.GetTickets)
		r.Post("/ticket", h.CreateTicket)
	})

	// Ticket-scoped routes
	r.Route("/ticket/{ticketId}", func(r chi.Router) {
		r.Get("/", h.GetTicketByID)
		r.Put("/assign", h.AssignTicket)
		r.Post("/log", h.AddLog)
		r.Put("/close", h.CloseTicket)
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

// CreateTicket handles POST /tracker/project/{projectId}/ticket
// Access: CONTRACTOR only
func (h *Handler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	var req tracker_service.CreateTicketParams

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

	result, err := h.trackerSvc.CreateTicket(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_CREATE_TICKET", result)
}

// GetTickets handles GET /tracker/project/{projectId}/ticket
// Access: Any project participant
func (h *Handler) GetTickets(w http.ResponseWriter, r *http.Request) {
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

	// Check query params: status=open|closed (default: all)
	statusFilter := r.URL.Query().Get("status")

	params := tracker_service.GetTicketsParams{
		ProjectID:    projectID,
		StatusFilter: statusFilter,
	}

	result, err := h.trackerSvc.GetTickets(r.Context(), params)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_GET_TICKETS", result)
}

// GetTicketByID handles GET /tracker/ticket/{ticketId}
// Access: Any project participant
func (h *Handler) GetTicketByID(w http.ResponseWriter, r *http.Request) {
	ticketIDStr := chi.URLParam(r, "ticketId")
	ticketID, err := uuid.Parse(ticketIDStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"ticketId": "Invalid UUID"}), nil)
		return
	}

	user, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	// Get ticket to check contract
	ticket, err := h.store.GetTicketWithContract(r.Context(), ticketID)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("TICKET_NOT_FOUND"), nil)
		return
	}

	// Check access via contract's project
	contract, err := h.store.GetContractByID(r.Context(), ticket.ContractID)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("CONTRACT_NOT_FOUND"), nil)
		return
	}

	_, err = h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: contract.ProjectID,
		UserID:    user.ID,
	})
	if err != nil {
		response.Error(w, r, errs.NewForbidden("NOT_PROJECT_PARTICIPANT"), nil)
		return
	}

	params := tracker_service.GetTicketByIDParams{
		TicketID: ticketID,
	}

	result, err := h.trackerSvc.GetTicketByID(r.Context(), params)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_GET_TICKET", result)
}

// AssignTicket handles PUT /tracker/ticket/{ticketId}/assign
// Access: EXECUTING_AGENCY ADMIN only
func (h *Handler) AssignTicket(w http.ResponseWriter, r *http.Request) {
	var req tracker_service.AssignTicketParams

	ticketIDStr := chi.URLParam(r, "ticketId")
	ticketID, err := uuid.Parse(ticketIDStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"ticketId": "Invalid UUID"}), nil)
		return
	}

	user, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	// Get ticket to check contract
	ticket, err := h.store.GetTicketWithContract(r.Context(), ticketID)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("TICKET_NOT_FOUND"), nil)
		return
	}

	// Check access
	contract, err := h.store.GetContractByID(r.Context(), ticket.ContractID)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("CONTRACT_NOT_FOUND"), nil)
		return
	}

	access, err := h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: contract.ProjectID,
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
	if access.MemberRole != entity.MemberRoleEnumADMIN {
		response.Error(w, r, errs.NewForbidden("ADMIN_REQUIRED"), nil)
		return
	}

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	req.TicketID = ticketID
	req.OrgID = access.OrganizationID
	req.UserID = user.ID

	result, err := h.trackerSvc.AssignTicket(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_ASSIGN_TICKET", result)
}

// AddLog handles POST /tracker/ticket/{ticketId}/log
// Access: GOV_PARTNER (assigned) or EXECUTING_AGENCY
func (h *Handler) AddLog(w http.ResponseWriter, r *http.Request) {
	var req tracker_service.AddTicketLogParams

	ticketIDStr := chi.URLParam(r, "ticketId")
	ticketID, err := uuid.Parse(ticketIDStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"ticketId": "Invalid UUID"}), nil)
		return
	}

	user, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	// Get ticket to check contract
	ticket, err := h.store.GetTicketWithContract(r.Context(), ticketID)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("TICKET_NOT_FOUND"), nil)
		return
	}

	// Check access
	contract, err := h.store.GetContractByID(r.Context(), ticket.ContractID)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("CONTRACT_NOT_FOUND"), nil)
		return
	}

	access, err := h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: contract.ProjectID,
		UserID:    user.ID,
	})
	if err != nil {
		response.Error(w, r, errs.NewForbidden("NOT_PROJECT_PARTICIPANT"), nil)
		return
	}

	role := getParticipantRoleString(access.ParticipantRole)
	isGovPartner := role == string(entity.OrganizationRoleEnumGOVPARTNER)
	isExecutingAgency := role == string(entity.OrganizationRoleEnumEXECUTINGAGENCY)

	if !isGovPartner && !isExecutingAgency {
		response.Error(w, r, errs.NewForbidden("GOV_PARTNER_OR_EXECUTING_AGENCY_REQUIRED"), nil)
		return
	}

	// If GOV_PARTNER, must be the assigned org
	if isGovPartner && ticket.TicketAssignToID.Valid && ticket.TicketAssignToID.UUID != access.OrganizationID {
		response.Error(w, r, errs.NewForbidden("NOT_ASSIGNED_TO_YOUR_ORG"), nil)
		return
	}

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	req.TicketID = ticketID
	req.OrgID = access.OrganizationID
	req.UserID = user.ID

	result, err := h.trackerSvc.AddLog(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_ADD_LOG", result)
}

// CloseTicket handles PUT /tracker/ticket/{ticketId}/close
// Access: EXECUTING_AGENCY ADMIN only
func (h *Handler) CloseTicket(w http.ResponseWriter, r *http.Request) {
	ticketIDStr := chi.URLParam(r, "ticketId")
	ticketID, err := uuid.Parse(ticketIDStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"ticketId": "Invalid UUID"}), nil)
		return
	}

	user, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	// Get ticket to check contract
	ticket, err := h.store.GetTicketWithContract(r.Context(), ticketID)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("TICKET_NOT_FOUND"), nil)
		return
	}

	// Check access
	contract, err := h.store.GetContractByID(r.Context(), ticket.ContractID)
	if err != nil {
		response.Error(w, r, errs.NewNotFound("CONTRACT_NOT_FOUND"), nil)
		return
	}

	access, err := h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: contract.ProjectID,
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
	if access.MemberRole != entity.MemberRoleEnumADMIN {
		response.Error(w, r, errs.NewForbidden("ADMIN_REQUIRED"), nil)
		return
	}

	params := tracker_service.CloseTicketParams{
		TicketID: ticketID,
		OrgID:    access.OrganizationID,
		UserID:   user.ID,
	}

	result, err := h.trackerSvc.CloseTicket(r.Context(), params)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_CLOSE_TICKET", result)
}
