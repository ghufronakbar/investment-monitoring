// internal/module/project/service/service.go
package project_service

import (
	"context"
	"database/sql"

	"investment_monitoring/config"
	"investment_monitoring/internal/audit"
	dashboard_cache "investment_monitoring/internal/module/support/dashboard/cache"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

type ProjectService struct {
	store *entity.SQLStore
	audit *audit.Service
	cache *dashboard_cache.Cache
	cfg   *config.Config
}

func NewService(store *entity.SQLStore, rdb *redis.Client, cfg *config.Config) *ProjectService {
	return &ProjectService{
		store: store,
		audit: audit.NewService(store),
		cache: dashboard_cache.New(rdb, cfg),
		cfg:   cfg,
	}
}

// Create creates a new project with contract (GOV_SUPERVISOR only)
func (s *ProjectService) Create(ctx context.Context, params CreateProjectParams, callerOrgID uuid.UUID) (CreateProjectReturn, error) {
	var result CreateProjectReturn

	// Validate caller organization is GOV_SUPERVISOR
	callerOrg, err := s.store.GetOrganizationByID(ctx, callerOrgID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("CALLER_ORGANIZATION_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}
	if callerOrg.Role != entity.OrganizationRoleEnumGOVSUPERVISOR {
		return result, errs.NewBadRequest("CALLER_NOT_GOV_SUPERVISOR")
	}

	// Validate loan provider organization exists and has LOAN_PROVIDER role
	loanProviderOrg, err := s.store.GetOrganizationByID(ctx, params.LoanProviderID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("LOAN_PROVIDER_ORGANIZATION_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}
	if loanProviderOrg.Role != entity.OrganizationRoleEnumLOANPROVIDER {
		return result, errs.NewBadRequest("ORGANIZATION_NOT_LOAN_PROVIDER")
	}

	// Prepare image urls
	imageUrls := params.ImageUrls
	if imageUrls == nil {
		imageUrls = []string{}
	}

	// Execute in transaction
	err = s.store.ExecTx(ctx, func(q *entity.Queries) error {
		// 1. Create Project
		project, err := q.CreateProject(ctx, entity.CreateProjectParams{
			Name:            params.Name,
			Description:     params.Description,
			ContractFileUrl: params.ContractFileUrl,
			ImageUrls:       imageUrls,
		})
		if err != nil {
			return err
		}

		// 2. Create Contract with GOV_SUPERVISOR and LOAN_PROVIDER
		contract, err := q.CreateContract(ctx, entity.CreateContractParams{
			Description:                params.Description,
			ProjectID:                  project.ID,
			GovSupervisorParticipantID: callerOrgID,
			LoanProviderParticipantID:  params.LoanProviderID,
		})
		if err != nil {
			return err
		}

		// Build caller org logo url
		var callerLogoUrl *string
		if callerOrg.LogoUrl.Valid {
			callerLogoUrl = &callerOrg.LogoUrl.String
		}

		// Build loan provider logo url
		var loanProviderLogoUrl *string
		if loanProviderOrg.LogoUrl.Valid {
			loanProviderLogoUrl = &loanProviderOrg.LogoUrl.String
		}

		// Map to return with all fields filled
		result = CreateProjectReturn{
			Project: ProjectReturn{
				ID:              project.ID,
				Name:            project.Name,
				Description:     project.Description,
				ContractFileUrl: project.ContractFileUrl,
				ImageUrls:       project.ImageUrls,
				CreatedAt:       project.CreatedAt,
			},
			Contract: ContractReturn{
				ID:          contract.ID,
				Description: contract.Description,
				GovSupervisor: ParticipantReturn{
					OrgID:   callerOrgID,
					OrgName: callerOrg.Name,
					OrgRole: string(callerOrg.Role),
					LogoUrl: callerLogoUrl,
				},
				LoanProvider: ParticipantReturn{
					OrgID:   params.LoanProviderID,
					OrgName: loanProviderOrg.Name,
					OrgRole: string(loanProviderOrg.Role),
					LogoUrl: loanProviderLogoUrl,
				},
				GovPartners: []ParticipantReturn{},
			},
		}

		return s.audit.RecordTx(ctx, q, audit.RecordParams{
			ActorUserID: params.UserID,
			ActorOrgID:  callerOrgID,
			Action:      audit.ActionCreateProject,
			EntityType:  audit.EntityProject,
			EntityID:    project.ID,
			AfterData: map[string]any{
				"projectId":      project.ID,
				"contractId":     contract.ID,
				"name":           project.Name,
				"loanProviderId": params.LoanProviderID,
			},
		})
	})

	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	return result, nil
}

// GetByID retrieves project details with contract and settings
func (s *ProjectService) GetByID(ctx context.Context, params GetProjectParams) (ProjectDetailReturn, error) {
	var result ProjectDetailReturn

	// Get project
	project, err := s.store.GetProjectByID(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("PROJECT_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Get contract with details
	contract, err := s.store.GetContractWithDetails(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return result, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Get gov partners
	govPartners, err := s.store.GetGovParticipantsByContractID(ctx, contract.ID)
	if err != nil {
		return result, errs.NewInternalServerError(err)
	}

	// Get settings (optional)
	settings, err := s.store.GetProjectSettingsByProjectID(ctx, params.ProjectID)
	var settingsReturn *ProjectSettingsReturn
	if err == nil {
		settingsReturn = &ProjectSettingsReturn{
			ID:               settings.ID,
			StartDate:        settings.StartDate,
			BaselineDeadline: settings.BaselineDeadline,
			TotalBudget:      settings.TotalBudget,
			Currency:         settings.Currency,
		}
	}

	// Build contract return
	contractReturn := ContractReturn{
		ID:          contract.ID,
		Description: contract.Description,
		GovSupervisor: ParticipantReturn{
			OrgID:   contract.GovSupervisorParticipantID,
			OrgName: contract.GovSupervisorName,
			OrgRole: string(entity.OrganizationRoleEnumGOVSUPERVISOR),
		},
		LoanProvider: ParticipantReturn{
			OrgID:   contract.LoanProviderParticipantID,
			OrgName: contract.LoanProviderName,
			OrgRole: string(entity.OrganizationRoleEnumLOANPROVIDER),
		},
		GovPartners: []ParticipantReturn{},
	}

	// Add optional fields
	if contract.ContractValue.Valid {
		contractReturn.ContractValue = &contract.ContractValue.Int64
	}
	if contract.ContractDocUrl.Valid {
		contractReturn.ContractDocUrl = &contract.ContractDocUrl.String
	}
	if contract.ExecutingAgencyParticipantID.Valid {
		contractReturn.ExecutingAgency = &ParticipantReturn{
			OrgID:   contract.ExecutingAgencyParticipantID.UUID,
			OrgName: contract.ExecutingAgencyName.String,
			OrgRole: string(entity.OrganizationRoleEnumEXECUTINGAGENCY),
		}
	}
	if contract.VendorParticipantID.Valid {
		contractReturn.Vendor = &ParticipantReturn{
			OrgID:   contract.VendorParticipantID.UUID,
			OrgName: contract.VendorName.String,
			OrgRole: string(entity.OrganizationRoleEnumCONTRACTOR),
		}
		if contract.VendorParticipantSelectedAt.Valid {
			contractReturn.VendorSelectedAt = &contract.VendorParticipantSelectedAt.Time
		}
	}
	if contract.AuditorParticipantID.Valid {
		contractReturn.Auditor = &ParticipantReturn{
			OrgID:   contract.AuditorParticipantID.UUID,
			OrgName: contract.AuditorName.String,
			OrgRole: string(entity.OrganizationRoleEnumINDEPENDENTAUDITOR),
		}
		if contract.AuditorParticipantSelectedAt.Valid {
			contractReturn.AuditorSelectedAt = &contract.AuditorParticipantSelectedAt.Time
		}
	}

	// Add gov partners
	for _, gp := range govPartners {
		var logoUrl *string
		if gp.OrgLogoUrl.Valid {
			logoUrl = &gp.OrgLogoUrl.String
		}
		contractReturn.GovPartners = append(contractReturn.GovPartners, ParticipantReturn{
			OrgID:   gp.GovParticipantID,
			OrgName: gp.OrgName,
			LogoUrl: logoUrl,
			OrgRole: string(entity.OrganizationRoleEnumGOVPARTNER),
		})
	}

	result = ProjectDetailReturn{
		Project: ProjectReturn{
			ID:              project.ID,
			Name:            project.Name,
			Description:     project.Description,
			ContractFileUrl: project.ContractFileUrl,
			ImageUrls:       project.ImageUrls,
			CreatedAt:       project.CreatedAt,
		},
		Contract: contractReturn,
		Settings: settingsReturn,
	}

	return result, nil
}

// ListAccessible retrieves all projects the user has access to
func (s *ProjectService) ListAccessible(ctx context.Context, userID uuid.UUID) ([]ProjectReturn, error) {
	projects, err := s.store.ListUserAccessibleProjects(ctx, userID)
	if err != nil {
		return nil, errs.NewInternalServerError(err)
	}

	result := make([]ProjectReturn, 0, len(projects))
	for _, p := range projects {
		result = append(result, ProjectReturn{
			ID:              p.ID,
			Name:            p.Name,
			Description:     p.Description,
			ContractFileUrl: p.ContractFileUrl,
			ImageUrls:       p.ImageUrls,
			CreatedAt:       p.CreatedAt,
		})
	}

	return result, nil
}

// ConfigureSettings creates or updates project settings
func (s *ProjectService) ConfigureSettings(ctx context.Context, params ConfigureSettingsParams) (ProjectSettingsReturn, error) {
	// Validate dates
	if params.BaselineDeadline.Before(params.StartDate) {
		return ProjectSettingsReturn{}, errs.NewBadRequest("BASELINE_DEADLINE_BEFORE_START_DATE")
	}

	// Verify project exists
	_, err := s.store.GetProjectByID(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return ProjectSettingsReturn{}, errs.NewNotFound("PROJECT_NOT_FOUND")
	}
	if err != nil {
		return ProjectSettingsReturn{}, errs.NewInternalServerError(err)
	}

	beforeSettings, beforeErr := s.store.GetProjectSettingsByProjectID(ctx, params.ProjectID)
	if beforeErr != nil && beforeErr != sql.ErrNoRows {
		return ProjectSettingsReturn{}, errs.NewInternalServerError(beforeErr)
	}

	var settings entity.ProjectSetting
	err = s.store.ExecTx(ctx, func(q *entity.Queries) error {
		var txErr error
		settings, txErr = q.UpsertProjectSettings(ctx, entity.UpsertProjectSettingsParams{
			ProjectID:        params.ProjectID,
			StartDate:        params.StartDate,
			BaselineDeadline: params.BaselineDeadline,
			TotalBudget:      params.TotalBudget,
			Currency:         params.Currency,
		})
		if txErr != nil {
			return txErr
		}

		var beforeData any
		if beforeErr == nil {
			beforeData = beforeSettings
		}

		return s.audit.RecordTx(ctx, q, audit.RecordParams{
			ActorUserID: params.UserID,
			ActorOrgID:  params.ActorOrgID,
			Action:      audit.ActionConfigureProject,
			EntityType:  audit.EntityProject,
			EntityID:    params.ProjectID,
			BeforeData:  beforeData,
			AfterData: map[string]any{
				"settingsId":       settings.ID,
				"projectId":        settings.ProjectID,
				"startDate":        settings.StartDate,
				"baselineDeadline": settings.BaselineDeadline,
				"totalBudget":      settings.TotalBudget,
				"currency":         settings.Currency,
			},
		})
	})
	if err != nil {
		return ProjectSettingsReturn{}, errs.NewInternalServerError(err)
	}
	_ = s.cache.InvalidateProject(ctx, params.ProjectID)

	return ProjectSettingsReturn{
		ID:               settings.ID,
		StartDate:        settings.StartDate,
		BaselineDeadline: settings.BaselineDeadline,
		TotalBudget:      settings.TotalBudget,
		Currency:         settings.Currency,
	}, nil
}

// AssignExecutingAgency assigns an executing agency to the contract
func (s *ProjectService) AssignExecutingAgency(ctx context.Context, params AssignExecutingAgencyParams) (AssignmentReturn, error) {
	// Get contract
	contract, err := s.store.GetContractByProjectID(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return AssignmentReturn{}, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	if err != nil {
		return AssignmentReturn{}, errs.NewInternalServerError(err)
	}

	// Verify org exists and is EXECUTING_AGENCY type
	org, err := s.store.GetOrganizationByID(ctx, params.ExecutingAgencyID)
	if err == sql.ErrNoRows {
		return AssignmentReturn{}, errs.NewNotFound("ORGANIZATION_NOT_FOUND")
	}
	if err != nil {
		return AssignmentReturn{}, errs.NewInternalServerError(err)
	}
	if org.Role != entity.OrganizationRoleEnumEXECUTINGAGENCY {
		return AssignmentReturn{}, errs.NewBadRequest("ORGANIZATION_NOT_EXECUTING_AGENCY")
	}

	err = s.store.ExecTx(ctx, func(q *entity.Queries) error {
		_, txErr := q.AssignExecutingAgency(ctx, entity.AssignExecutingAgencyParams{
			ID:                           contract.ID,
			ExecutingAgencyParticipantID: uuid.NullUUID{UUID: params.ExecutingAgencyID, Valid: true},
		})
		if txErr != nil {
			return txErr
		}
		return s.audit.RecordTx(ctx, q, audit.RecordParams{
			ActorUserID: params.UserID,
			ActorOrgID:  params.ActorOrgID,
			Action:      audit.ActionAssignExecutingAgency,
			EntityType:  audit.EntityContract,
			EntityID:    contract.ID,
			BeforeData: map[string]any{
				"executingAgencyId": contract.ExecutingAgencyParticipantID,
			},
			AfterData: map[string]any{
				"projectId":          params.ProjectID,
				"executingAgencyId":  params.ExecutingAgencyID,
				"executingAgencyOrg": org.Name,
			},
		})
	})
	if err != nil {
		return AssignmentReturn{}, errs.NewInternalServerError(err)
	}
	_ = s.cache.InvalidateProject(ctx, params.ProjectID)

	return AssignmentReturn{
		Success: true,
		Message: "EXECUTING_AGENCY_ASSIGNED",
	}, nil
}

// AssignVendor assigns a vendor/contractor to the contract
func (s *ProjectService) AssignVendor(ctx context.Context, params AssignVendorParams) (AssignmentReturn, error) {
	// Get contract
	contract, err := s.store.GetContractByProjectID(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return AssignmentReturn{}, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	if err != nil {
		return AssignmentReturn{}, errs.NewInternalServerError(err)
	}

	// Validate project settings exist - required before assigning vendor
	settings, err := s.store.GetProjectSettingsByProjectID(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return AssignmentReturn{}, errs.NewBadRequest("PROJECT_SETTINGS_REQUIRED")
	}
	if err != nil {
		return AssignmentReturn{}, errs.NewInternalServerError(err)
	}

	// Validate contract value is less than total budget
	if params.ContractValue >= settings.TotalBudget {
		return AssignmentReturn{}, errs.NewBadRequest("CONTRACT_VALUE_EXCEEDS_BUDGET")
	}

	// Verify org exists and is CONTRACTOR type
	org, err := s.store.GetOrganizationByID(ctx, params.VendorOrgID)
	if err == sql.ErrNoRows {
		return AssignmentReturn{}, errs.NewNotFound("ORGANIZATION_NOT_FOUND")
	}
	if err != nil {
		return AssignmentReturn{}, errs.NewInternalServerError(err)
	}
	if org.Role != entity.OrganizationRoleEnumCONTRACTOR {
		return AssignmentReturn{}, errs.NewBadRequest("ORGANIZATION_NOT_CONTRACTOR")
	}

	err = s.store.ExecTx(ctx, func(q *entity.Queries) error {
		_, txErr := q.AssignVendorToContract(ctx, entity.AssignVendorToContractParams{
			ID:                  contract.ID,
			VendorParticipantID: uuid.NullUUID{UUID: params.VendorOrgID, Valid: true},
			ContractValue:       sql.NullInt64{Int64: params.ContractValue, Valid: true},
			ContractDocUrl:      sql.NullString{String: params.ContractDocUrl, Valid: true},
		})
		if txErr != nil {
			return txErr
		}
		return s.audit.RecordTx(ctx, q, audit.RecordParams{
			ActorUserID: params.UserID,
			ActorOrgID:  params.ActorOrgID,
			Action:      audit.ActionAssignVendor,
			EntityType:  audit.EntityContract,
			EntityID:    contract.ID,
			BeforeData: map[string]any{
				"vendorOrgId":    contract.VendorParticipantID,
				"contractValue":  contract.ContractValue,
				"contractDocUrl": contract.ContractDocUrl,
			},
			AfterData: map[string]any{
				"projectId":      params.ProjectID,
				"vendorOrgId":    params.VendorOrgID,
				"vendorName":     org.Name,
				"contractValue":  params.ContractValue,
				"contractDocUrl": params.ContractDocUrl,
			},
		})
	})
	if err != nil {
		return AssignmentReturn{}, errs.NewInternalServerError(err)
	}
	_ = s.cache.InvalidateProject(ctx, params.ProjectID)

	return AssignmentReturn{
		Success: true,
		Message: "VENDOR_ASSIGNED",
	}, nil
}

// AssignAuditor assigns an independent auditor to the contract
func (s *ProjectService) AssignAuditor(ctx context.Context, params AssignAuditorParams) (AssignmentReturn, error) {
	// Get contract
	contract, err := s.store.GetContractByProjectID(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return AssignmentReturn{}, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	if err != nil {
		return AssignmentReturn{}, errs.NewInternalServerError(err)
	}

	// Verify org exists and is INDEPENDENT_AUDITOR type
	org, err := s.store.GetOrganizationByID(ctx, params.AuditorOrgID)
	if err == sql.ErrNoRows {
		return AssignmentReturn{}, errs.NewNotFound("ORGANIZATION_NOT_FOUND")
	}
	if err != nil {
		return AssignmentReturn{}, errs.NewInternalServerError(err)
	}
	if org.Role != entity.OrganizationRoleEnumINDEPENDENTAUDITOR {
		return AssignmentReturn{}, errs.NewBadRequest("ORGANIZATION_NOT_AUDITOR")
	}

	err = s.store.ExecTx(ctx, func(q *entity.Queries) error {
		_, txErr := q.AssignAuditor(ctx, entity.AssignAuditorParams{
			ID:                   contract.ID,
			AuditorParticipantID: uuid.NullUUID{UUID: params.AuditorOrgID, Valid: true},
		})
		if txErr != nil {
			return txErr
		}
		return s.audit.RecordTx(ctx, q, audit.RecordParams{
			ActorUserID: params.UserID,
			ActorOrgID:  params.ActorOrgID,
			Action:      audit.ActionAssignAuditor,
			EntityType:  audit.EntityContract,
			EntityID:    contract.ID,
			BeforeData: map[string]any{
				"auditorOrgId": contract.AuditorParticipantID,
			},
			AfterData: map[string]any{
				"projectId":    params.ProjectID,
				"auditorOrgId": params.AuditorOrgID,
				"auditorName":  org.Name,
			},
		})
	})
	if err != nil {
		return AssignmentReturn{}, errs.NewInternalServerError(err)
	}

	return AssignmentReturn{
		Success: true,
		Message: "AUDITOR_ASSIGNED",
	}, nil
}

// AssignGovPartners assigns government partners (multi-select)
func (s *ProjectService) AssignGovPartners(ctx context.Context, params AssignGovPartnersParams) (AssignmentReturn, error) {
	// Get contract
	contract, err := s.store.GetContractByProjectID(ctx, params.ProjectID)
	if err == sql.ErrNoRows {
		return AssignmentReturn{}, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	if err != nil {
		return AssignmentReturn{}, errs.NewInternalServerError(err)
	}

	// Validate all orgs are GOV_PARTNER type
	for _, orgID := range params.GovPartnerIDs {
		org, err := s.store.GetOrganizationByID(ctx, orgID)
		if err == sql.ErrNoRows {
			return AssignmentReturn{}, errs.NewNotFound("ORGANIZATION_NOT_FOUND")
		}
		if err != nil {
			return AssignmentReturn{}, errs.NewInternalServerError(err)
		}
		if org.Role != entity.OrganizationRoleEnumGOVPARTNER {
			return AssignmentReturn{}, errs.NewBadRequest("ORGANIZATION_NOT_GOV_PARTNER")
		}
	}

	existingPartners, err := s.store.GetGovParticipantsByContractID(ctx, contract.ID)
	if err != nil {
		return AssignmentReturn{}, errs.NewInternalServerError(err)
	}

	// Execute in transaction: delete existing, add new
	err = s.store.ExecTx(ctx, func(q *entity.Queries) error {
		// Delete existing gov partners
		err := q.DeleteGovParticipantsByContractID(ctx, contract.ID)
		if err != nil {
			return err
		}

		// Add new gov partners
		for _, orgID := range params.GovPartnerIDs {
			_, err := q.AddGovParticipant(ctx, entity.AddGovParticipantParams{
				ContractID:       contract.ID,
				GovParticipantID: orgID,
			})
			if err != nil {
				return err
			}
		}

		return s.audit.RecordTx(ctx, q, audit.RecordParams{
			ActorUserID: params.UserID,
			ActorOrgID:  params.ActorOrgID,
			Action:      audit.ActionAssignGovPartners,
			EntityType:  audit.EntityContract,
			EntityID:    contract.ID,
			BeforeData: map[string]any{
				"govPartners": existingPartners,
			},
			AfterData: map[string]any{
				"projectId":     params.ProjectID,
				"govPartnerIds": params.GovPartnerIDs,
			},
		})
	})

	if err != nil {
		return AssignmentReturn{}, errs.NewInternalServerError(err)
	}

	return AssignmentReturn{
		Success: true,
		Message: "GOV_PARTNERS_ASSIGNED",
	}, nil
}

// Ensure pq is imported for array handling
var _ = pq.Array
