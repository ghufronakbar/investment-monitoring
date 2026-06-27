// internal/module/iam/organization/service/service.go
package organization_service

import (
	"context"
	"database/sql"

	"investment_monitoring/config"
	"investment_monitoring/internal/audit"
	mail_service "investment_monitoring/internal/module/support/mail/service"
	"investment_monitoring/internal/outbox"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"

	"github.com/google/uuid"
)

type OrganizationService struct {
	store   entity.Store
	audit   *audit.Service
	outbox  *outbox.Service
	mailSvc *mail_service.MailService
	cfg     *config.Config
}

func NewService(store entity.Store, mailSvc *mail_service.MailService, cfg *config.Config) *OrganizationService {
	return &OrganizationService{
		store:   store,
		audit:   audit.NewService(store),
		outbox:  outbox.NewService(store),
		mailSvc: mailSvc,
		cfg:     cfg,
	}
}

// Create creates a new organization with its first admin user
// This is an atomic transaction that creates: Organization, User (password=null), OrganizationMember
func (s *OrganizationService) Create(ctx context.Context, params CreateOrganizationParams) (CreateOrganizationReturn, error) {
	var result CreateOrganizationReturn

	// Check if email already exists
	exists, err := s.store.CheckUserEmailExists(ctx, params.AdminEmail)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}
	if exists {
		return result, errs.NewBadRequest("EMAIL_ALREADY_EXISTS")
	}

	// Execute in transaction
	err = s.store.ExecTx(ctx, func(q *entity.Queries) error {
		// 1. Create Organization
		org, err := q.CreateOrganization(ctx, entity.CreateOrganizationParams{
			Name:    params.Name,
			LogoUrl: sql.NullString{String: params.LogoUrl, Valid: params.LogoUrl != ""},
			Role:    entity.OrganizationRoleEnum(params.Role),
		})
		if err != nil {
			return err
		}

		// 2. Create User (password = null, will be set when user activates)
		user, err := q.CreateUser(ctx, entity.CreateUserParams{
			Name:     params.AdminName,
			Email:    params.AdminEmail,
			Password: sql.NullString{Valid: false}, // null password
			ImageUrl: sql.NullString{Valid: false},
		})
		if err != nil {
			return err
		}

		// 3. Create OrganizationMember with ADMIN role
		_, err = q.CreateOrganizationMember(ctx, entity.CreateOrganizationMemberParams{
			Role:           entity.MemberRoleEnumADMIN,
			UserID:         user.ID,
			OrganizationID: org.ID,
		})
		if err != nil {
			return err
		}

		if err := s.audit.RecordTx(ctx, q, audit.RecordParams{
			ActorUserID: params.UserID,
			ActorOrgID:  params.ActorOrgID,
			Action:      audit.ActionCreateOrganization,
			EntityType:  audit.EntityOrganization,
			EntityID:    org.ID,
			AfterData: map[string]any{
				"organizationId": org.ID,
				"organization":   org.Name,
				"role":           org.Role,
				"adminUserId":    user.ID,
				"adminEmail":     user.Email,
			},
		}); err != nil {
			return err
		}

		// Map to return values
		var logoUrl *string
		if org.LogoUrl.Valid {
			logoUrl = &org.LogoUrl.String
		}

		result = CreateOrganizationReturn{
			Organization: OrganizationReturn{
				ID:        org.ID,
				Name:      org.Name,
				Role:      string(org.Role),
				LogoUrl:   logoUrl,
				CreatedAt: org.CreatedAt,
			},
			Admin: AdminReturn{
				ID:    user.ID,
				Name:  user.Name,
				Email: user.Email,
			},
		}

		return nil
	})

	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Send invitation email (non-blocking, error logged but not returned)
	go func() {
		_ = s.mailSvc.SendInvitationEmail(context.Background(), mail_service.SendInvitationInput{
			ToEmail:          params.AdminEmail,
			ToName:           params.AdminName,
			OrganizationName: params.Name,
			OrganizationRole: params.Role,
			MemberRole:       "ADMIN",
		})
	}()

	return result, nil
}

// InviteMember invites a new member to an organization
func (s *OrganizationService) InviteMember(ctx context.Context, params InviteMemberParams) (InviteMemberReturn, error) {
	var result InviteMemberReturn

	// Verify organization exists
	org, err := s.store.GetOrganizationByID(ctx, params.OrganizationID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("ORGANIZATION_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Check if user already exists
	existingUser, err := s.store.GetUserByEmailFull(ctx, params.Email)
	if err != nil && err != sql.ErrNoRows {
		return result, errs.NewInternalServerError(err)
	}

	var userID = existingUser.ID

	// If user doesn't exist, create new user
	if err == sql.ErrNoRows {
		// Execute in transaction
		err = s.store.ExecTx(ctx, func(q *entity.Queries) error {
			// Create new user
			user, err := q.CreateUser(ctx, entity.CreateUserParams{
				Name:     params.Name,
				Email:    params.Email,
				Password: sql.NullString{Valid: false},
				ImageUrl: sql.NullString{Valid: false},
			})
			if err != nil {
				return err
			}

			userID = user.ID

			// Create membership with MEMBER role
			membership, err := q.CreateOrganizationMember(ctx, entity.CreateOrganizationMemberParams{
				Role:           entity.MemberRoleEnumMEMBER,
				UserID:         user.ID,
				OrganizationID: params.OrganizationID,
			})
			if err != nil {
				return err
			}

			result = InviteMemberReturn{
				Member: MemberReturn{
					ID:         membership.ID,
					Name:       user.Name,
					Email:      user.Email,
					MemberRole: string(membership.Role),
					CreatedAt:  membership.CreatedAt,
				},
			}

			if err := s.audit.RecordTx(ctx, q, audit.RecordParams{
				ActorUserID: params.UserID,
				ActorOrgID:  params.ActorOrgID,
				Action:      audit.ActionInviteOrganizationUser,
				EntityType:  audit.EntityUser,
				EntityID:    user.ID,
				AfterData: map[string]any{
					"organizationId": params.OrganizationID,
					"organization":   org.Name,
					"userId":         user.ID,
					"email":          user.Email,
					"memberRole":     membership.Role,
					"isNewUser":      true,
				},
			}); err != nil {
				return err
			}

			return s.outbox.PublishTx(ctx, q, outbox.Event{
				EventType:     outbox.EventMemberInvited,
				AggregateType: outbox.AggregateUser,
				AggregateID:   user.ID,
				Payload: map[string]any{
					"organizationId":   params.OrganizationID,
					"organizationName": org.Name,
					"userId":           user.ID,
					"email":            user.Email,
					"name":             user.Name,
					"memberRole":       membership.Role,
					"isNewUser":        true,
				},
			})
		})

		if err != nil {
			return result, errs.NewInternalServerError(err)
		}
	} else {
		// User exists, check if already member of this org
		exists, err := s.store.CheckMembershipExists(ctx, entity.CheckMembershipExistsParams{
			UserID:         userID,
			OrganizationID: params.OrganizationID,
		})
		if err != nil {
			return result, errs.NewInternalServerError(err)
		}
		if exists {
			return result, errs.NewBadRequest("USER_ALREADY_MEMBER")
		}

		var membership entity.OrganizationMember
		err = s.store.ExecTx(ctx, func(q *entity.Queries) error {
			var txErr error
			// Add existing user to organization
			membership, txErr = q.CreateOrganizationMember(ctx, entity.CreateOrganizationMemberParams{
				Role:           entity.MemberRoleEnumMEMBER,
				UserID:         userID,
				OrganizationID: params.OrganizationID,
			})
			if txErr != nil {
				return txErr
			}

			if err := s.audit.RecordTx(ctx, q, audit.RecordParams{
				ActorUserID: params.UserID,
				ActorOrgID:  params.ActorOrgID,
				Action:      audit.ActionInviteOrganizationUser,
				EntityType:  audit.EntityUser,
				EntityID:    userID,
				AfterData: map[string]any{
					"organizationId": params.OrganizationID,
					"organization":   org.Name,
					"userId":         userID,
					"email":          existingUser.Email,
					"memberRole":     membership.Role,
					"isNewUser":      false,
				},
			}); err != nil {
				return err
			}

			return s.outbox.PublishTx(ctx, q, outbox.Event{
				EventType:     outbox.EventMemberInvited,
				AggregateType: outbox.AggregateUser,
				AggregateID:   userID,
				Payload: map[string]any{
					"organizationId":   params.OrganizationID,
					"organizationName": org.Name,
					"userId":           userID,
					"email":            existingUser.Email,
					"name":             existingUser.Name,
					"memberRole":       membership.Role,
					"isNewUser":        false,
				},
			})
		})
		if err != nil {
			return result, errs.NewInternalServerError(err)
		}

		result = InviteMemberReturn{
			Member: MemberReturn{
				ID:         membership.ID,
				Name:       existingUser.Name,
				Email:      existingUser.Email,
				MemberRole: string(membership.Role),
				CreatedAt:  membership.CreatedAt,
			},
		}
	}

	// Send invitation email
	go func() {
		_ = s.mailSvc.SendInvitationEmail(context.Background(), mail_service.SendInvitationInput{
			ToEmail:          params.Email,
			ToName:           params.Name,
			OrganizationName: org.Name,
			OrganizationRole: string(org.Role),
			MemberRole:       "MEMBER",
		})
	}()

	return result, nil
}

// GetByID retrieves organization details by ID
func (s *OrganizationService) GetByID(ctx context.Context, params GetOrganizationParams) (GetOrganizationReturn, error) {
	org, err := s.store.GetOrganizationByID(ctx, params.ID)
	if err == sql.ErrNoRows {
		return GetOrganizationReturn{}, errs.NewNotFound("ORGANIZATION_NOT_FOUND")
	}
	if err != nil {
		return GetOrganizationReturn{}, errs.NewInternalServerError(err)
	}

	var logoUrl *string
	if org.LogoUrl.Valid {
		logoUrl = &org.LogoUrl.String
	}

	return GetOrganizationReturn{
		ID:        org.ID,
		Name:      org.Name,
		Role:      string(org.Role),
		LogoUrl:   logoUrl,
		CreatedAt: org.CreatedAt,
	}, nil
}

// GetMembers retrieves all members of an organization
func (s *OrganizationService) GetMembers(ctx context.Context, params GetMembersParams) ([]MemberDetailReturn, error) {
	// Verify organization exists
	_, err := s.store.GetOrganizationByID(ctx, params.OrganizationID)
	if err == sql.ErrNoRows {
		return nil, errs.NewNotFound("ORGANIZATION_NOT_FOUND")
	}
	if err != nil {
		return []MemberDetailReturn{}, errs.NewInternalServerError(err)
	}

	members, err := s.store.GetOrganizationMembers(ctx, params.OrganizationID)
	if err != nil {
		return []MemberDetailReturn{}, errs.NewInternalServerError(err)
	}

	memberReturns := make([]MemberDetailReturn, 0, len(members))
	for _, m := range members {
		var imageUrl *string
		if m.UserImageUrl.Valid {
			imageUrl = &m.UserImageUrl.String
		}
		memberReturns = append(memberReturns, MemberDetailReturn{
			ID:         m.ID,
			UserID:     m.UserID,
			Name:       m.UserName,
			Email:      m.UserEmail,
			ImageUrl:   imageUrl,
			MemberRole: string(m.MemberRole),
			CreatedAt:  m.CreatedAt,
		})
	}

	return memberReturns, nil
}

// GetMyOrganizations retrieves all organizations that the user has joined
func (s *OrganizationService) GetMyOrganizations(ctx context.Context, params GetMyOrganizationsParams) ([]MyOrganizationReturn, error) {
	memberships, err := s.store.GetUserMemberships(ctx, params.UserID)
	if err != nil {
		return []MyOrganizationReturn{}, errs.NewInternalServerError(err)
	}

	orgs := make([]MyOrganizationReturn, 0, len(memberships))
	for _, m := range memberships {
		var logoUrl *string
		if m.OrgLogoUrl.Valid {
			logoUrl = &m.OrgLogoUrl.String
		}
		orgs = append(orgs, MyOrganizationReturn{
			MembershipID: m.ID,
			MemberRole:   string(m.MemberRole),
			OrgID:        m.OrgID,
			OrgName:      m.OrgName,
			OrgRole:      string(m.OrgRole),
			OrgLogoUrl:   logoUrl,
			JoinedAt:     m.CreatedAt,
		})
	}

	return orgs, nil
}

// RegisteredOrganizationReturn for public listing
type RegisteredOrganizationReturn struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Role    string    `json:"role"`
	LogoUrl *string   `json:"logoUrl"`
}

// GetRegisteredOrganizations retrieves all registered organizations (public access)
func (s *OrganizationService) GetRegisteredOrganizations(ctx context.Context) ([]RegisteredOrganizationReturn, error) {
	orgs, err := s.store.ListOrganizations(ctx, entity.ListOrganizationsParams{
		Limit:  1000, // TODO: Add pagination if needed
		Offset: 0,
	})
	if err != nil {
		return []RegisteredOrganizationReturn{}, errs.NewInternalServerError(err)
	}

	result := make([]RegisteredOrganizationReturn, 0, len(orgs))
	for _, o := range orgs {
		var logoUrl *string
		if o.LogoUrl.Valid {
			logoUrl = &o.LogoUrl.String
		}
		result = append(result, RegisteredOrganizationReturn{
			ID:      o.ID,
			Name:    o.Name,
			Role:    string(o.Role),
			LogoUrl: logoUrl,
		})
	}

	return result, nil
}

// Update updates an organization's name and logo
func (s *OrganizationService) Update(ctx context.Context, params UpdateOrganizationParams) (UpdateOrganizationReturn, error) {
	// Verify organization exists
	_, err := s.store.GetOrganizationByID(ctx, params.ID)
	if err == sql.ErrNoRows {
		return UpdateOrganizationReturn{}, errs.NewNotFound("ORGANIZATION_NOT_FOUND")
	}
	if err != nil {
		return UpdateOrganizationReturn{}, errs.NewInternalServerError(err)
	}

	// Prepare logo URL
	var logoUrl sql.NullString
	if params.LogoUrl != nil && *params.LogoUrl != "" {
		logoUrl = sql.NullString{String: *params.LogoUrl, Valid: true}
	}

	// Update organization
	org, err := s.store.UpdateOrganization(ctx, entity.UpdateOrganizationParams{
		ID:      params.ID,
		Name:    params.Name,
		LogoUrl: logoUrl,
	})
	if err != nil {
		return UpdateOrganizationReturn{}, errs.NewInternalServerError(err)
	}

	// Map response
	var returnLogoUrl *string
	if org.LogoUrl.Valid {
		returnLogoUrl = &org.LogoUrl.String
	}

	return UpdateOrganizationReturn{
		ID:        org.ID,
		Name:      org.Name,
		Role:      string(org.Role),
		LogoUrl:   returnLogoUrl,
		UpdatedAt: org.UpdatedAt,
	}, nil
}
