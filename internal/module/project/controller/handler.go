// internal/module/project/controller/handler.go
package project_controller

import (
	"net/http"

	"investment_monitoring/internal/http/internal_middleware"
	project_service "investment_monitoring/internal/module/project/service"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/response"
	"investment_monitoring/pkg/utils"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc   *project_service.ProjectService
	store entity.Store
}

func NewHandler(svc *project_service.ProjectService, store entity.Store) *Handler {
	return &Handler{
		svc:   svc,
		store: store,
	}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// List projects accessible to the user
	r.Get("/", h.List)

	// Create project - Only GOV_SUPERVISOR ADMIN can create
	r.Post("/", h.Create)

	// Project-specific routes
	r.Route("/{projectId}", func(r chi.Router) {
		// Get project details
		r.Get("/", h.GetByID)

		// Configure settings - EXECUTING_AGENCY ADMIN
		r.Put("/settings", h.ConfigureSettings)

		// Contract assignments
		r.Route("/contract", func(r chi.Router) {
			// Assign executing agency - GOV_SUPERVISOR ADMIN
			r.Put("/assign-executing-agency", h.AssignExecutingAgency)
			// Assign vendor - EXECUTING_AGENCY ADMIN
			r.Put("/assign-vendor", h.AssignVendor)
			// Assign auditor - EXECUTING_AGENCY ADMIN
			r.Put("/assign-auditor", h.AssignAuditor)
			// Assign gov partners - EXECUTING_AGENCY ADMIN
			r.Put("/assign-gov-partners", h.AssignGovPartners)
		})
	})

	return r
}

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

// Create handles POST /project
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req project_service.CreateProjectParams

	// Get authenticated user
	userClaims, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	// Check if user has GOV_SUPERVISOR membership
	memberships, err := h.store.GetUserMemberships(r.Context(), userClaims.ID)
	if err != nil {
		response.Error(w, r, errs.NewInternalServerError(err), nil)
		return
	}

	var callerOrgID uuid.UUID
	hasGovSupervisor := false
	for _, m := range memberships {
		if m.OrgRole == entity.OrganizationRoleEnumGOVSUPERVISOR && m.MemberRole == entity.MemberRoleEnumADMIN {
			callerOrgID = m.OrgID
			hasGovSupervisor = true
			break
		}
	}

	if !hasGovSupervisor {
		response.Error(w, r, errs.NewForbidden("GOV_SUPERVISOR_ADMIN_REQUIRED"), nil)
		return
	}

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	res, err := h.svc.Create(r.Context(), req, callerOrgID)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_CREATE_PROJECT", res)
}

// List handles GET /project
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user
	userClaims, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	res, err := h.svc.ListAccessible(r.Context(), userClaims.ID)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_LIST_PROJECTS", res)
}

// GetByID handles GET /project/{projectId}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user
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

	// Check if user has access to this project
	_, err = h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: projectId,
		UserID:    userClaims.ID,
	})
	if err != nil {
		// Allow GOV_SUPERVISOR to view any project
		memberships, _ := h.store.GetUserMemberships(r.Context(), userClaims.ID)
		hasGovSupervisor := false
		for _, m := range memberships {
			if m.OrgRole == entity.OrganizationRoleEnumGOVSUPERVISOR {
				hasGovSupervisor = true
				break
			}
		}
		if !hasGovSupervisor {
			response.Error(w, r, errs.NewForbidden("NO_PROJECT_ACCESS"), nil)
			return
		}
	}

	res, err := h.svc.GetByID(r.Context(), project_service.GetProjectParams{
		ProjectID: projectId,
		UserID:    userClaims.ID,
	})
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_GET_PROJECT", res)
}

// ConfigureSettings handles PUT /project/{projectId}/settings
func (h *Handler) ConfigureSettings(w http.ResponseWriter, r *http.Request) {
	var req project_service.ConfigureSettingsParams

	// Get authenticated user and check EXECUTING_AGENCY ADMIN access
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

	// Check access: must be EXECUTING_AGENCY ADMIN for this project
	access, err := h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: projectId,
		UserID:    userClaims.ID,
	})
	if err != nil {
		response.Error(w, r, errs.NewForbidden("NO_PROJECT_ACCESS"), nil)
		return
	}
	if getParticipantRoleString(access.ParticipantRole) != "EXECUTING_AGENCY" || access.MemberRole != entity.MemberRoleEnumADMIN {
		response.Error(w, r, errs.NewForbidden("EXECUTING_AGENCY_ADMIN_REQUIRED"), nil)
		return
	}

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	req.ProjectID = projectId

	res, err := h.svc.ConfigureSettings(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_CONFIGURE_SETTINGS", res)
}

// AssignExecutingAgency handles PUT /project/{projectId}/contract/assign-executing-agency
func (h *Handler) AssignExecutingAgency(w http.ResponseWriter, r *http.Request) {
	var req project_service.AssignExecutingAgencyParams

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

	// Check access: must be GOV_SUPERVISOR ADMIN
	access, err := h.store.GetUserProjectAccess(r.Context(), entity.GetUserProjectAccessParams{
		ProjectID: projectId,
		UserID:    userClaims.ID,
	})
	if err != nil || getParticipantRoleString(access.ParticipantRole) != "GOV_SUPERVISOR" || access.MemberRole != entity.MemberRoleEnumADMIN {
		response.Error(w, r, errs.NewForbidden("GOV_SUPERVISOR_ADMIN_REQUIRED"), nil)
		return
	}

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	req.ProjectID = projectId

	res, err := h.svc.AssignExecutingAgency(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_ASSIGN_EXECUTING_AGENCY", res)
}

// AssignVendor handles PUT /project/{projectId}/contract/assign-vendor
func (h *Handler) AssignVendor(w http.ResponseWriter, r *http.Request) {
	var req project_service.AssignVendorParams

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

	res, err := h.svc.AssignVendor(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_ASSIGN_VENDOR", res)
}

// AssignAuditor handles PUT /project/{projectId}/contract/assign-auditor
func (h *Handler) AssignAuditor(w http.ResponseWriter, r *http.Request) {
	var req project_service.AssignAuditorParams

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

	res, err := h.svc.AssignAuditor(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_ASSIGN_AUDITOR", res)
}

// AssignGovPartners handles PUT /project/{projectId}/contract/assign-gov-partners
func (h *Handler) AssignGovPartners(w http.ResponseWriter, r *http.Request) {
	var req project_service.AssignGovPartnersParams

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

	res, err := h.svc.AssignGovPartners(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_ASSIGN_GOV_PARTNERS", res)
}
