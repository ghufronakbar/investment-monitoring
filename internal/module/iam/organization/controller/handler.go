// internal/module/iam/organization/controller/handler.go
package organization_controller

import (
	"net/http"

	"investment_monitoring/internal/http/internal_middleware"
	organization_service "investment_monitoring/internal/module/iam/organization/service"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/response"
	"investment_monitoring/pkg/utils"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc   *organization_service.OrganizationService
	store entity.Store
}

func NewHandler(svc *organization_service.OrganizationService, store entity.Store) *Handler {
	return &Handler{
		svc:   svc,
		store: store,
	}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Public route - no auth required
	r.Get("/registered", h.GetRegisteredOrganizations)

	// Route to get all organizations the current user has joined (no orgId needed)
	r.Get("/my", h.GetMyOrganizations)

	// Routes that require orgId with role-based access control
	r.Route("/{orgId}", func(r chi.Router) {
		// Create Organization - Only GOV_SUPERVISOR with ADMIN role can create new organizations
		r.With(internal_middleware.OrgRoleMiddleware(h.store, internal_middleware.OrgRoleMiddlewareOptions{
			AllowedOrgRoles:    internal_middleware.GovSupervisorAndExecutingAgency(),
			AllowedMemberRoles: internal_middleware.AdminOnly(),
		})).Post("/", h.Create)

		// Get Organization - Any member of the org can view
		r.With(internal_middleware.OrgRoleMiddleware(h.store, internal_middleware.OrgRoleMiddlewareOptions{
			AllowedOrgRoles:    internal_middleware.AllOrgRoles(),
			AllowedMemberRoles: internal_middleware.AllMemberRoles(),
		})).Get("/", h.GetByID)

		// Get Members - Any member of the org can view members
		r.With(internal_middleware.OrgRoleMiddleware(h.store, internal_middleware.OrgRoleMiddlewareOptions{
			AllowedOrgRoles:    internal_middleware.AllOrgRoles(),
			AllowedMemberRoles: internal_middleware.AllMemberRoles(),
		})).Get("/member", h.GetMembers)

		// Invite Member - Any org, but only ADMIN can invite
		r.With(internal_middleware.OrgRoleMiddleware(h.store, internal_middleware.OrgRoleMiddlewareOptions{
			AllowedOrgRoles:    internal_middleware.AllOrgRoles(),
			AllowedMemberRoles: internal_middleware.AdminOnly(),
		})).Post("/member/invite", h.InviteMember)

		// Update Organization - Any org, but only ADMIN can update
		r.With(internal_middleware.OrgRoleMiddleware(h.store, internal_middleware.OrgRoleMiddlewareOptions{
			AllowedOrgRoles:    internal_middleware.AllOrgRoles(),
			AllowedMemberRoles: internal_middleware.AdminOnly(),
		})).Put("/", h.Update)
	})

	return r
}

// Create handles POST /iam/organization/{orgId}/create
// Creates a new organization (only GOV_SUPERVISOR ADMIN can do this)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req organization_service.CreateOrganizationParams

	// Get org membership from middleware (validates the caller is GOV_SUPERVISOR ADMIN)
	membership, err := internal_middleware.GetOrgMembershipFromContext(r.Context())
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	req.UserID = membership.UserID
	req.ActorOrgID = membership.OrganizationID

	res, err := h.svc.Create(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_CREATE_ORGANIZATION", res)
}

// GetByID handles GET /iam/organization/{orgId}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	// Get org membership from middleware
	membership, err := internal_middleware.GetOrgMembershipFromContext(r.Context())
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	res, err := h.svc.GetByID(r.Context(), organization_service.GetOrganizationParams{
		ID: membership.OrganizationID,
	})
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_GET_ORGANIZATION", res)
}

// GetMembers handles GET /iam/organization/{orgId}/members
func (h *Handler) GetMembers(w http.ResponseWriter, r *http.Request) {
	// Get org membership from middleware
	membership, err := internal_middleware.GetOrgMembershipFromContext(r.Context())
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	res, err := h.svc.GetMembers(r.Context(), organization_service.GetMembersParams{
		OrganizationID: membership.OrganizationID,
	})
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_GET_MEMBERS", res)
}

// InviteMember handles POST /iam/organization/{orgId}/member/invite
func (h *Handler) InviteMember(w http.ResponseWriter, r *http.Request) {
	var req organization_service.InviteMemberParams

	// Get org membership from middleware (validates caller is ADMIN)
	membership, err := internal_middleware.GetOrgMembershipFromContext(r.Context())
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	orgIdStr := chi.URLParam(r, "orgId")
	orgId, err := uuid.Parse(orgIdStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"orgId": "Invalid UUID"}), nil)
		return
	}

	// Verify the orgId matches the membership context (sanity check)
	if orgId != membership.OrganizationID {
		response.Error(w, r, errs.NewForbidden("ORG_ID_MISMATCH"), nil)
		return
	}

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	req.OrganizationID = orgId
	req.UserID = membership.UserID
	req.ActorOrgID = membership.OrganizationID

	res, err := h.svc.InviteMember(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_INVITE_MEMBER", res)
}

// GetMyOrganizations handles GET /iam/organization/my
// Returns all organizations the authenticated user has joined
func (h *Handler) GetMyOrganizations(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user from context
	userClaims, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	res, err := h.svc.GetMyOrganizations(r.Context(), organization_service.GetMyOrganizationsParams{
		UserID: userClaims.ID,
	})
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_GET_MY_ORGANIZATIONS", res)
}

// GetRegisteredOrganizations handles GET /iam/organization/registered
// Returns all registered organizations (public access, no auth required)
func (h *Handler) GetRegisteredOrganizations(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.GetRegisteredOrganizations(r.Context())
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_GET_REGISTERED_ORGANIZATIONS", res)
}

// Update handles PUT /iam/organization/{orgId}
// Updates organization name and logo (only ADMIN can do this)
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req organization_service.UpdateOrganizationParams

	// Get org membership from middleware (validates caller is ADMIN)
	membership, err := internal_middleware.GetOrgMembershipFromContext(r.Context())
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	orgIdStr := chi.URLParam(r, "orgId")
	orgId, err := uuid.Parse(orgIdStr)
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"orgId": "Invalid UUID"}), nil)
		return
	}

	// Verify the orgId matches the membership context
	if orgId != membership.OrganizationID {
		response.Error(w, r, errs.NewForbidden("ORG_ID_MISMATCH"), nil)
		return
	}

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	req.ID = orgId

	res, err := h.svc.Update(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_UPDATE_ORGANIZATION", res)
}
