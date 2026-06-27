package policy

import (
	"context"

	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"

	"github.com/google/uuid"
)

type Service struct {
	store entity.Store
}

type ProjectAccess struct {
	ContractID       uuid.UUID
	ProjectID        uuid.UUID
	OrganizationID   uuid.UUID
	OrganizationName string
	ParticipantRole  string
	MemberRole       entity.MemberRoleEnum
	OrganizationType entity.OrganizationRoleEnum
}

func NewService(store entity.Store) *Service {
	return &Service{store: store}
}

func participantRoleString(role interface{}) string {
	if role == nil {
		return ""
	}
	if s, ok := role.(string); ok {
		return s
	}
	return ""
}

func mapProjectAccess(access entity.GetUserProjectAccessRow) *ProjectAccess {
	return &ProjectAccess{
		ContractID:       access.ContractID,
		ProjectID:        access.ProjectID,
		OrganizationID:   access.OrganizationID,
		OrganizationName: access.OrgName,
		ParticipantRole:  participantRoleString(access.ParticipantRole),
		MemberRole:       access.MemberRole,
		OrganizationType: access.OrgType,
	}
}

func (s *Service) RequireProjectParticipant(ctx context.Context, userID, projectID uuid.UUID, forbiddenMessage string) (*ProjectAccess, error) {
	access, err := s.store.GetUserProjectAccess(ctx, entity.GetUserProjectAccessParams{
		ProjectID: projectID,
		UserID:    userID,
	})
	if err != nil {
		return nil, errs.NewForbidden(forbiddenMessage)
	}
	return mapProjectAccess(access), nil
}

func (s *Service) RequireProjectRole(ctx context.Context, userID, projectID uuid.UUID, role entity.OrganizationRoleEnum, adminOnly bool, forbiddenMessage string) (*ProjectAccess, error) {
	access, err := s.RequireProjectParticipant(ctx, userID, projectID, forbiddenMessage)
	if err != nil {
		return nil, err
	}
	if access.ParticipantRole != string(role) {
		return nil, errs.NewForbidden(forbiddenMessage)
	}
	if adminOnly && access.MemberRole != entity.MemberRoleEnumADMIN {
		return nil, errs.NewForbidden(forbiddenMessage)
	}
	return access, nil
}

func (s *Service) UserHasOrgRole(ctx context.Context, userID uuid.UUID, role entity.OrganizationRoleEnum, adminOnly bool) (uuid.UUID, bool, error) {
	memberships, err := s.store.GetUserMemberships(ctx, userID)
	if err != nil {
		return uuid.Nil, false, errs.NewInternalServerError(err)
	}
	for _, membership := range memberships {
		if membership.OrgRole != role {
			continue
		}
		if adminOnly && membership.MemberRole != entity.MemberRoleEnumADMIN {
			continue
		}
		return membership.OrgID, true, nil
	}
	return uuid.Nil, false, nil
}

func (s *Service) CanCreateProject(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	orgID, ok, err := s.UserHasOrgRole(ctx, userID, entity.OrganizationRoleEnumGOVSUPERVISOR, true)
	if err != nil {
		return uuid.Nil, err
	}
	if !ok {
		return uuid.Nil, errs.NewForbidden("GOV_SUPERVISOR_ADMIN_REQUIRED")
	}
	return orgID, nil
}

func (s *Service) CanViewProject(ctx context.Context, userID, projectID uuid.UUID) error {
	if _, err := s.RequireProjectParticipant(ctx, userID, projectID, "NO_PROJECT_ACCESS"); err == nil {
		return nil
	}

	_, ok, err := s.UserHasOrgRole(ctx, userID, entity.OrganizationRoleEnumGOVSUPERVISOR, false)
	if err != nil {
		return err
	}
	if !ok {
		return errs.NewForbidden("NO_PROJECT_ACCESS")
	}
	return nil
}

func (s *Service) CanConfigureProject(ctx context.Context, userID, projectID uuid.UUID) error {
	_, err := s.RequireProjectRole(ctx, userID, projectID, entity.OrganizationRoleEnumEXECUTINGAGENCY, true, "EXECUTING_AGENCY_ADMIN_REQUIRED")
	return err
}

func (s *Service) CanAssignExecutingAgency(ctx context.Context, userID, projectID uuid.UUID) error {
	_, err := s.RequireProjectRole(ctx, userID, projectID, entity.OrganizationRoleEnumGOVSUPERVISOR, true, "GOV_SUPERVISOR_ADMIN_REQUIRED")
	return err
}

func (s *Service) CanManageProjectContract(ctx context.Context, userID, projectID uuid.UUID) error {
	_, err := s.RequireProjectRole(ctx, userID, projectID, entity.OrganizationRoleEnumEXECUTINGAGENCY, true, "EXECUTING_AGENCY_ADMIN_REQUIRED")
	return err
}

func (s *Service) CanManageTask(ctx context.Context, userID, taskID uuid.UUID) error {
	task, err := s.store.GetTaskByID(ctx, taskID)
	if err != nil {
		return errs.NewNotFound("TASK_NOT_FOUND")
	}
	contract, err := s.store.GetContractByID(ctx, task.ContractID)
	if err != nil {
		return errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	_, err = s.RequireProjectRole(ctx, userID, contract.ProjectID, entity.OrganizationRoleEnumEXECUTINGAGENCY, true, "EXECUTING_AGENCY_ADMIN_REQUIRED")
	return err
}

func (s *Service) CanSubmitProgress(ctx context.Context, userID, taskID uuid.UUID) (*ProjectAccess, error) {
	task, err := s.store.GetTaskByID(ctx, taskID)
	if err != nil {
		return nil, errs.NewNotFound("TASK_NOT_FOUND")
	}
	contract, err := s.store.GetContractByID(ctx, task.ContractID)
	if err != nil {
		return nil, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	return s.RequireProjectRole(ctx, userID, contract.ProjectID, entity.OrganizationRoleEnumCONTRACTOR, false, "CONTRACTOR_REQUIRED")
}

func (s *Service) CanViewMonitoringProject(ctx context.Context, userID, projectID uuid.UUID) error {
	_, err := s.RequireProjectParticipant(ctx, userID, projectID, "NO_PROJECT_ACCESS")
	return err
}

func (s *Service) CanViewPendingValidations(ctx context.Context, userID, projectID uuid.UUID) error {
	_, err := s.RequireProjectRole(ctx, userID, projectID, entity.OrganizationRoleEnumINDEPENDENTAUDITOR, false, "AUDITOR_REQUIRED")
	return err
}

func (s *Service) CanValidateProgress(ctx context.Context, userID, logID uuid.UUID) (*ProjectAccess, error) {
	log, err := s.store.GetProgressLogWithDetails(ctx, logID)
	if err != nil {
		return nil, errs.NewNotFound("PROGRESS_LOG_NOT_FOUND")
	}
	contract, err := s.store.GetContractByID(ctx, log.ContractID)
	if err != nil {
		return nil, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	return s.RequireProjectRole(ctx, userID, contract.ProjectID, entity.OrganizationRoleEnumINDEPENDENTAUDITOR, false, "AUDITOR_REQUIRED")
}

func (s *Service) CanCreateTicket(ctx context.Context, userID, projectID uuid.UUID) (*ProjectAccess, error) {
	access, err := s.RequireProjectParticipant(ctx, userID, projectID, "NOT_PROJECT_PARTICIPANT")
	if err != nil {
		return nil, err
	}
	if access.ParticipantRole != string(entity.OrganizationRoleEnumCONTRACTOR) {
		return nil, errs.NewForbidden("CONTRACTOR_REQUIRED")
	}
	return access, nil
}

func (s *Service) CanViewTrackerProject(ctx context.Context, userID, projectID uuid.UUID) error {
	_, err := s.RequireProjectParticipant(ctx, userID, projectID, "NOT_PROJECT_PARTICIPANT")
	return err
}

func (s *Service) ticketProjectAccess(ctx context.Context, userID, ticketID uuid.UUID) (entity.GetTicketWithContractRow, *ProjectAccess, error) {
	ticket, err := s.store.GetTicketWithContract(ctx, ticketID)
	if err != nil {
		return entity.GetTicketWithContractRow{}, nil, errs.NewNotFound("TICKET_NOT_FOUND")
	}
	contract, err := s.store.GetContractByID(ctx, ticket.ContractID)
	if err != nil {
		return entity.GetTicketWithContractRow{}, nil, errs.NewNotFound("CONTRACT_NOT_FOUND")
	}
	access, err := s.RequireProjectParticipant(ctx, userID, contract.ProjectID, "NOT_PROJECT_PARTICIPANT")
	if err != nil {
		return entity.GetTicketWithContractRow{}, nil, err
	}
	return ticket, access, nil
}

func (s *Service) CanViewTicket(ctx context.Context, userID, ticketID uuid.UUID) error {
	_, _, err := s.ticketProjectAccess(ctx, userID, ticketID)
	return err
}

func (s *Service) CanAssignTicket(ctx context.Context, userID, ticketID uuid.UUID) (*ProjectAccess, error) {
	_, access, err := s.ticketProjectAccess(ctx, userID, ticketID)
	if err != nil {
		return nil, err
	}
	if access.ParticipantRole != string(entity.OrganizationRoleEnumEXECUTINGAGENCY) {
		return nil, errs.NewForbidden("EXECUTING_AGENCY_REQUIRED")
	}
	if access.MemberRole != entity.MemberRoleEnumADMIN {
		return nil, errs.NewForbidden("ADMIN_REQUIRED")
	}
	return access, nil
}

func (s *Service) CanAddTicketLog(ctx context.Context, userID, ticketID uuid.UUID) (*ProjectAccess, error) {
	ticket, access, err := s.ticketProjectAccess(ctx, userID, ticketID)
	if err != nil {
		return nil, err
	}

	isGovPartner := access.ParticipantRole == string(entity.OrganizationRoleEnumGOVPARTNER)
	isExecutingAgency := access.ParticipantRole == string(entity.OrganizationRoleEnumEXECUTINGAGENCY)
	if !isGovPartner && !isExecutingAgency {
		return nil, errs.NewForbidden("GOV_PARTNER_OR_EXECUTING_AGENCY_REQUIRED")
	}
	if isGovPartner && ticket.TicketAssignToID.Valid && ticket.TicketAssignToID.UUID != access.OrganizationID {
		return nil, errs.NewForbidden("NOT_ASSIGNED_TO_YOUR_ORG")
	}
	return access, nil
}

func (s *Service) CanCloseTicket(ctx context.Context, userID, ticketID uuid.UUID) (*ProjectAccess, error) {
	return s.CanAssignTicket(ctx, userID, ticketID)
}

func (s *Service) CanCreatePaymentRequest(ctx context.Context, userID, projectID uuid.UUID) (*ProjectAccess, error) {
	access, err := s.RequireProjectParticipant(ctx, userID, projectID, "NOT_PROJECT_PARTICIPANT")
	if err != nil {
		return nil, err
	}
	if access.ParticipantRole != string(entity.OrganizationRoleEnumCONTRACTOR) {
		return nil, errs.NewForbidden("CONTRACTOR_REQUIRED")
	}
	return access, nil
}

func (s *Service) CanViewFinanceProject(ctx context.Context, userID, projectID uuid.UUID) error {
	_, err := s.RequireProjectParticipant(ctx, userID, projectID, "NOT_PROJECT_PARTICIPANT")
	return err
}

func (s *Service) CanViewDashboardProject(ctx context.Context, userID, projectID uuid.UUID) error {
	_, err := s.RequireProjectParticipant(ctx, userID, projectID, "NOT_PROJECT_PARTICIPANT")
	return err
}

func (s *Service) paymentProjectAccess(ctx context.Context, userID, requestID uuid.UUID) (*ProjectAccess, error) {
	pr, err := s.store.GetPaymentRequestWithContract(ctx, requestID)
	if err != nil {
		return nil, errs.NewNotFound("PAYMENT_REQUEST_NOT_FOUND")
	}
	return s.RequireProjectParticipant(ctx, userID, pr.ProjectID, "NOT_PROJECT_PARTICIPANT")
}

func (s *Service) CanViewPaymentRequest(ctx context.Context, userID, requestID uuid.UUID) error {
	_, err := s.paymentProjectAccess(ctx, userID, requestID)
	return err
}

func (s *Service) CanApprovePaymentAsAuditor(ctx context.Context, userID, requestID uuid.UUID) (*ProjectAccess, error) {
	access, err := s.paymentProjectAccess(ctx, userID, requestID)
	if err != nil {
		return nil, err
	}
	if access.ParticipantRole != string(entity.OrganizationRoleEnumINDEPENDENTAUDITOR) {
		return nil, errs.NewForbidden("AUDITOR_REQUIRED")
	}
	return access, nil
}

func (s *Service) CanApprovePaymentAsAgency(ctx context.Context, userID, requestID uuid.UUID) (*ProjectAccess, error) {
	access, err := s.paymentProjectAccess(ctx, userID, requestID)
	if err != nil {
		return nil, err
	}
	if access.ParticipantRole != string(entity.OrganizationRoleEnumEXECUTINGAGENCY) {
		return nil, errs.NewForbidden("EXECUTING_AGENCY_REQUIRED")
	}
	return access, nil
}

func (s *Service) CanDisbursePayment(ctx context.Context, userID, requestID uuid.UUID) (*ProjectAccess, error) {
	access, err := s.paymentProjectAccess(ctx, userID, requestID)
	if err != nil {
		return nil, err
	}
	if access.ParticipantRole != string(entity.OrganizationRoleEnumLOANPROVIDER) {
		return nil, errs.NewForbidden("LOAN_PROVIDER_REQUIRED")
	}
	return access, nil
}
