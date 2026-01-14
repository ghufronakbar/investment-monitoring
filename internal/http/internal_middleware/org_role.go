// internal/http/internal_middleware/org_role.go
package internal_middleware

import (
	"context"
	"database/sql"
	"net/http"

	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// OrgRoleContextKey is the key for storing org membership in context
type orgRoleContextKey string

const OrgMembershipContextKey orgRoleContextKey = "orgMembership"

// OrgMembershipContext holds the current user's membership info for the org
type OrgMembershipContext struct {
	OrganizationID   uuid.UUID
	OrganizationName string
	OrganizationRole entity.OrganizationRoleEnum
	MemberID         uuid.UUID
	MemberRole       entity.MemberRoleEnum
	UserID           uuid.UUID
	UserName         string
	UserEmail        string
	IsGovSupervisor  bool // True if user has GOV_SUPERVISOR bypass
}

// OrgRoleMiddlewareOptions configures which roles are allowed
type OrgRoleMiddlewareOptions struct {
	AllowedOrgRoles    []entity.OrganizationRoleEnum
	AllowedMemberRoles []entity.MemberRoleEnum
}

// OrgRoleMiddleware validates that the authenticated user:
// 1. Is a member of the organization specified by {orgId} URL param
// 2. Has one of the allowed organization roles
// 3. Has one of the allowed member roles within that organization
//
// Special rule: For GET requests, if user has at least 1 membership in any
// GOV_SUPERVISOR organization, they are allowed to view all organizations.
func OrgRoleMiddleware(store entity.Store, opts OrgRoleMiddlewareOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Get authenticated user from context
			userClaims, err := GetUserFromContext(r.Context())
			if err != nil {
				response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
				return
			}

			// 2. Get orgId from URL
			orgIdStr := chi.URLParam(r, "orgId")
			if orgIdStr == "" {
				response.Error(w, r, errs.NewBadRequest("MISSING_ORG_ID"), nil)
				return
			}

			orgId, err := uuid.Parse(orgIdStr)
			if err != nil {
				response.Error(w, r, errs.NewValidationFailed(map[string]string{"orgId": "Invalid UUID"}), nil)
				return
			}

			// 3. Get organization
			org, err := store.GetOrganizationByID(r.Context(), orgId)
			if err == sql.ErrNoRows {
				response.Error(w, r, errs.NewNotFound("ORGANIZATION_NOT_FOUND"), nil)
				return
			}
			if err != nil {
				response.Error(w, r, errs.NewInternalServerError(err), nil)
				return
			}

			// 4. For GET requests, check if user has GOV_SUPERVISOR membership (bypass)
			isGovSupervisorBypass := false
			if r.Method == http.MethodGet {
				memberships, err := store.GetUserMemberships(r.Context(), userClaims.ID)
				if err == nil {
					for _, m := range memberships {
						if m.OrgRole == entity.OrganizationRoleEnumGOVSUPERVISOR {
							isGovSupervisorBypass = true
							break
						}
					}
				}
			}

			// 5. Check if user is member of this organization
			membership, err := store.GetOrganizationMember(r.Context(), entity.GetOrganizationMemberParams{
				UserID:         userClaims.ID,
				OrganizationID: orgId,
			})

			// If not a member of this org
			if err == sql.ErrNoRows {
				// Allow if GOV_SUPERVISOR bypass for GET requests
				if isGovSupervisorBypass {
					// Create a virtual membership context for the viewer
					membershipCtx := OrgMembershipContext{
						OrganizationID:   org.ID,
						OrganizationName: org.Name,
						OrganizationRole: org.Role,
						MemberID:         uuid.Nil, // Not a real member
						MemberRole:       "",       // Not a real member
						UserID:           userClaims.ID,
						UserName:         "",
						UserEmail:        userClaims.Email,
						IsGovSupervisor:  true, // Flag indicating bypass
					}
					ctx := context.WithValue(r.Context(), OrgMembershipContextKey, &membershipCtx)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				response.Error(w, r, errs.NewForbidden("NOT_MEMBER_OF_ORGANIZATION"), nil)
				return
			}
			if err != nil {
				response.Error(w, r, errs.NewInternalServerError(err), nil)
				return
			}

			// 6. Validate organization role (skip if GOV_SUPERVISOR bypass)
			if !isGovSupervisorBypass && len(opts.AllowedOrgRoles) > 0 {
				allowed := false
				for _, allowedRole := range opts.AllowedOrgRoles {
					if org.Role == allowedRole {
						allowed = true
						break
					}
				}
				if !allowed {
					response.Error(w, r, errs.NewForbidden("ORGANIZATION_ROLE_NOT_ALLOWED"), nil)
					return
				}
			}

			// 7. Validate member role (skip if GOV_SUPERVISOR bypass)
			if !isGovSupervisorBypass && len(opts.AllowedMemberRoles) > 0 {
				allowed := false
				for _, allowedRole := range opts.AllowedMemberRoles {
					if membership.Role == allowedRole {
						allowed = true
						break
					}
				}
				if !allowed {
					response.Error(w, r, errs.NewForbidden("MEMBER_ROLE_NOT_ALLOWED"), nil)
					return
				}
			}

			// 8. Store membership context for handlers
			membershipCtx := OrgMembershipContext{
				OrganizationID:   org.ID,
				OrganizationName: org.Name,
				OrganizationRole: org.Role,
				MemberID:         membership.ID,
				MemberRole:       membership.Role,
				UserID:           userClaims.ID,
				UserName:         membership.UserName,
				UserEmail:        membership.UserEmail,
				IsGovSupervisor:  isGovSupervisorBypass,
			}

			ctx := context.WithValue(r.Context(), OrgMembershipContextKey, &membershipCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetOrgMembershipFromContext retrieves org membership from request context
func GetOrgMembershipFromContext(ctx context.Context) (*OrgMembershipContext, error) {
	membership, ok := ctx.Value(OrgMembershipContextKey).(*OrgMembershipContext)
	if !ok || membership == nil {
		return nil, errs.NewForbidden("MISSING_ORG_MEMBERSHIP_CONTEXT")
	}
	return membership, nil
}

// Helper functions to create common middleware configurations

// AllOrgRoles returns all organization roles
func AllOrgRoles() []entity.OrganizationRoleEnum {
	return []entity.OrganizationRoleEnum{
		entity.OrganizationRoleEnumGOVSUPERVISOR,
		entity.OrganizationRoleEnumLOANPROVIDER,
		entity.OrganizationRoleEnumEXECUTINGAGENCY,
		entity.OrganizationRoleEnumCONTRACTOR,
		entity.OrganizationRoleEnumINDEPENDENTAUDITOR,
		entity.OrganizationRoleEnumGOVPARTNER,
	}
}

// AllMemberRoles returns all member roles
func AllMemberRoles() []entity.MemberRoleEnum {
	return []entity.MemberRoleEnum{
		entity.MemberRoleEnumADMIN,
		entity.MemberRoleEnumMEMBER,
	}
}

// AdminOnly returns only ADMIN role
func AdminOnly() []entity.MemberRoleEnum {
	return []entity.MemberRoleEnum{
		entity.MemberRoleEnumADMIN,
	}
}

// GovSupervisorOnly returns only GOV_SUPERVISOR role
func GovSupervisorOnly() []entity.OrganizationRoleEnum {
	return []entity.OrganizationRoleEnum{
		entity.OrganizationRoleEnumGOVSUPERVISOR,
	}
}

func GovSupervisorAndExecutingAgency() []entity.OrganizationRoleEnum {
	return []entity.OrganizationRoleEnum{
		entity.OrganizationRoleEnumGOVSUPERVISOR,
		entity.OrganizationRoleEnumEXECUTINGAGENCY,
	}
}
