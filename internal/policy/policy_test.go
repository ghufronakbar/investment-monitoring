package policy

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"

	"github.com/google/uuid"
)

type fakeStore struct {
	entity.Store

	memberships []entity.GetUserMembershipsRow
	access      entity.GetUserProjectAccessRow
	accessErr   error

	task          entity.TaskItem
	taskErr       error
	contract      entity.Contract
	contractErr   error
	ticket        entity.GetTicketWithContractRow
	ticketErr     error
	payment       entity.GetPaymentRequestWithContractRow
	paymentErr    error
	progress      entity.GetProgressLogWithDetailsRow
	progressErr   error
	membershipErr error
}

func (f *fakeStore) GetUserMemberships(ctx context.Context, userID uuid.UUID) ([]entity.GetUserMembershipsRow, error) {
	if f.membershipErr != nil {
		return nil, f.membershipErr
	}
	return f.memberships, nil
}

func (f *fakeStore) GetUserProjectAccess(ctx context.Context, arg entity.GetUserProjectAccessParams) (entity.GetUserProjectAccessRow, error) {
	if f.accessErr != nil {
		return entity.GetUserProjectAccessRow{}, f.accessErr
	}
	return f.access, nil
}

func (f *fakeStore) GetTaskByID(ctx context.Context, id uuid.UUID) (entity.TaskItem, error) {
	if f.taskErr != nil {
		return entity.TaskItem{}, f.taskErr
	}
	return f.task, nil
}

func (f *fakeStore) GetContractByID(ctx context.Context, id uuid.UUID) (entity.Contract, error) {
	if f.contractErr != nil {
		return entity.Contract{}, f.contractErr
	}
	return f.contract, nil
}

func (f *fakeStore) GetTicketWithContract(ctx context.Context, id uuid.UUID) (entity.GetTicketWithContractRow, error) {
	if f.ticketErr != nil {
		return entity.GetTicketWithContractRow{}, f.ticketErr
	}
	return f.ticket, nil
}

func (f *fakeStore) GetPaymentRequestWithContract(ctx context.Context, id uuid.UUID) (entity.GetPaymentRequestWithContractRow, error) {
	if f.paymentErr != nil {
		return entity.GetPaymentRequestWithContractRow{}, f.paymentErr
	}
	return f.payment, nil
}

func (f *fakeStore) GetProgressLogWithDetails(ctx context.Context, id uuid.UUID) (entity.GetProgressLogWithDetailsRow, error) {
	if f.progressErr != nil {
		return entity.GetProgressLogWithDetailsRow{}, f.progressErr
	}
	return f.progress, nil
}

func TestCanCreateProjectRequiresGovSupervisorAdmin(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name        string
		memberships []entity.GetUserMembershipsRow
		wantOrgID   uuid.UUID
		wantErr     string
	}{
		{
			name: "allow gov supervisor admin",
			memberships: []entity.GetUserMembershipsRow{{
				OrgID:      orgID,
				OrgRole:    entity.OrganizationRoleEnumGOVSUPERVISOR,
				MemberRole: entity.MemberRoleEnumADMIN,
			}},
			wantOrgID: orgID,
		},
		{
			name: "reject gov supervisor member",
			memberships: []entity.GetUserMembershipsRow{{
				OrgID:      orgID,
				OrgRole:    entity.OrganizationRoleEnumGOVSUPERVISOR,
				MemberRole: entity.MemberRoleEnumMEMBER,
			}},
			wantErr: "GOV_SUPERVISOR_ADMIN_REQUIRED",
		},
		{
			name: "reject other org role",
			memberships: []entity.GetUserMembershipsRow{{
				OrgID:      orgID,
				OrgRole:    entity.OrganizationRoleEnumEXECUTINGAGENCY,
				MemberRole: entity.MemberRoleEnumADMIN,
			}},
			wantErr: "GOV_SUPERVISOR_ADMIN_REQUIRED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&fakeStore{memberships: tt.memberships})
			got, err := svc.CanCreateProject(context.Background(), userID)
			assertPolicyError(t, err, tt.wantErr)
			if tt.wantErr == "" && got != tt.wantOrgID {
				t.Fatalf("org id = %s, want %s", got, tt.wantOrgID)
			}
		})
	}
}

func TestCanViewProjectAllowsParticipantOrGovSupervisorFallback(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()

	tests := []struct {
		name    string
		store   *fakeStore
		wantErr string
	}{
		{
			name:  "allow participant",
			store: &fakeStore{access: accessRow(projectID, entity.OrganizationRoleEnumCONTRACTOR, entity.MemberRoleEnumMEMBER, uuid.New())},
		},
		{
			name: "allow gov supervisor fallback",
			store: &fakeStore{
				accessErr: sql.ErrNoRows,
				memberships: []entity.GetUserMembershipsRow{{
					OrgID:      uuid.New(),
					OrgRole:    entity.OrganizationRoleEnumGOVSUPERVISOR,
					MemberRole: entity.MemberRoleEnumMEMBER,
				}},
			},
		},
		{
			name: "reject non participant without fallback",
			store: &fakeStore{
				accessErr: sql.ErrNoRows,
				memberships: []entity.GetUserMembershipsRow{{
					OrgID:      uuid.New(),
					OrgRole:    entity.OrganizationRoleEnumCONTRACTOR,
					MemberRole: entity.MemberRoleEnumADMIN,
				}},
			},
			wantErr: "NO_PROJECT_ACCESS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewService(tt.store).CanViewProject(context.Background(), userID, projectID)
			assertPolicyError(t, err, tt.wantErr)
		})
	}
}

func TestProjectRolePolicies(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()

	tests := []struct {
		name    string
		access  entity.GetUserProjectAccessRow
		call    func(*Service) error
		wantErr string
	}{
		{
			name:   "configure project allows executing agency admin",
			access: accessRow(projectID, entity.OrganizationRoleEnumEXECUTINGAGENCY, entity.MemberRoleEnumADMIN, uuid.New()),
			call: func(s *Service) error {
				return s.CanConfigureProject(context.Background(), userID, projectID)
			},
		},
		{
			name:   "configure project rejects non admin",
			access: accessRow(projectID, entity.OrganizationRoleEnumEXECUTINGAGENCY, entity.MemberRoleEnumMEMBER, uuid.New()),
			call: func(s *Service) error {
				return s.CanConfigureProject(context.Background(), userID, projectID)
			},
			wantErr: "EXECUTING_AGENCY_ADMIN_REQUIRED",
		},
		{
			name:   "assign executing agency requires gov supervisor admin",
			access: accessRow(projectID, entity.OrganizationRoleEnumGOVSUPERVISOR, entity.MemberRoleEnumADMIN, uuid.New()),
			call: func(s *Service) error {
				return s.CanAssignExecutingAgency(context.Background(), userID, projectID)
			},
		},
		{
			name:   "manage project contract rejects wrong role",
			access: accessRow(projectID, entity.OrganizationRoleEnumCONTRACTOR, entity.MemberRoleEnumADMIN, uuid.New()),
			call: func(s *Service) error {
				return s.CanManageProjectContract(context.Background(), userID, projectID)
			},
			wantErr: "EXECUTING_AGENCY_ADMIN_REQUIRED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call(NewService(&fakeStore{access: tt.access}))
			assertPolicyError(t, err, tt.wantErr)
		})
	}
}

func TestTaskAndProgressPolicies(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()
	contractID := uuid.New()
	taskID := uuid.New()
	logID := uuid.New()

	t.Run("CanManageTask allows executing agency admin", func(t *testing.T) {
		store := &fakeStore{
			task:     entity.TaskItem{ID: taskID, ContractID: contractID},
			contract: entity.Contract{ID: contractID, ProjectID: projectID},
			access:   accessRow(projectID, entity.OrganizationRoleEnumEXECUTINGAGENCY, entity.MemberRoleEnumADMIN, uuid.New()),
		}

		err := NewService(store).CanManageTask(context.Background(), userID, taskID)
		assertPolicyError(t, err, "")
	})

	t.Run("CanManageTask returns not found when task missing", func(t *testing.T) {
		store := &fakeStore{taskErr: sql.ErrNoRows}
		err := NewService(store).CanManageTask(context.Background(), userID, taskID)
		assertPolicyError(t, err, "TASK_NOT_FOUND")
	})

	t.Run("CanSubmitProgress allows contractor", func(t *testing.T) {
		store := &fakeStore{
			task:     entity.TaskItem{ID: taskID, ContractID: contractID},
			contract: entity.Contract{ID: contractID, ProjectID: projectID},
			access:   accessRow(projectID, entity.OrganizationRoleEnumCONTRACTOR, entity.MemberRoleEnumMEMBER, uuid.New()),
		}

		access, err := NewService(store).CanSubmitProgress(context.Background(), userID, taskID)
		assertPolicyError(t, err, "")
		if access.ParticipantRole != string(entity.OrganizationRoleEnumCONTRACTOR) {
			t.Fatalf("participant role = %s, want contractor", access.ParticipantRole)
		}
	})

	t.Run("CanValidateProgress requires auditor", func(t *testing.T) {
		store := &fakeStore{
			progress: entity.GetProgressLogWithDetailsRow{ID: logID, ContractID: contractID},
			contract: entity.Contract{ID: contractID, ProjectID: projectID},
			access:   accessRow(projectID, entity.OrganizationRoleEnumCONTRACTOR, entity.MemberRoleEnumMEMBER, uuid.New()),
		}

		_, err := NewService(store).CanValidateProgress(context.Background(), userID, logID)
		assertPolicyError(t, err, "AUDITOR_REQUIRED")
	})
}

func TestTicketPolicies(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()
	contractID := uuid.New()
	ticketID := uuid.New()
	assignedOrgID := uuid.New()

	t.Run("CanCreateTicket allows contractor participant", func(t *testing.T) {
		store := &fakeStore{access: accessRow(projectID, entity.OrganizationRoleEnumCONTRACTOR, entity.MemberRoleEnumMEMBER, uuid.New())}
		_, err := NewService(store).CanCreateTicket(context.Background(), userID, projectID)
		assertPolicyError(t, err, "")
	})

	t.Run("CanCreateTicket rejects non contractor", func(t *testing.T) {
		store := &fakeStore{access: accessRow(projectID, entity.OrganizationRoleEnumEXECUTINGAGENCY, entity.MemberRoleEnumADMIN, uuid.New())}
		_, err := NewService(store).CanCreateTicket(context.Background(), userID, projectID)
		assertPolicyError(t, err, "CONTRACTOR_REQUIRED")
	})

	t.Run("CanAssignTicket allows executing agency admin", func(t *testing.T) {
		store := ticketStore(projectID, contractID, ticketID, uuid.Nil, entity.OrganizationRoleEnumEXECUTINGAGENCY, entity.MemberRoleEnumADMIN, uuid.New())
		_, err := NewService(store).CanAssignTicket(context.Background(), userID, ticketID)
		assertPolicyError(t, err, "")
	})

	t.Run("CanAssignTicket rejects executing agency member", func(t *testing.T) {
		store := ticketStore(projectID, contractID, ticketID, uuid.Nil, entity.OrganizationRoleEnumEXECUTINGAGENCY, entity.MemberRoleEnumMEMBER, uuid.New())
		_, err := NewService(store).CanAssignTicket(context.Background(), userID, ticketID)
		assertPolicyError(t, err, "ADMIN_REQUIRED")
	})

	t.Run("CanAddTicketLog allows assigned gov partner", func(t *testing.T) {
		store := ticketStore(projectID, contractID, ticketID, assignedOrgID, entity.OrganizationRoleEnumGOVPARTNER, entity.MemberRoleEnumMEMBER, assignedOrgID)
		_, err := NewService(store).CanAddTicketLog(context.Background(), userID, ticketID)
		assertPolicyError(t, err, "")
	})

	t.Run("CanAddTicketLog rejects unassigned gov partner", func(t *testing.T) {
		store := ticketStore(projectID, contractID, ticketID, assignedOrgID, entity.OrganizationRoleEnumGOVPARTNER, entity.MemberRoleEnumMEMBER, uuid.New())
		_, err := NewService(store).CanAddTicketLog(context.Background(), userID, ticketID)
		assertPolicyError(t, err, "NOT_ASSIGNED_TO_YOUR_ORG")
	})

	t.Run("CanAddTicketLog rejects contractor", func(t *testing.T) {
		store := ticketStore(projectID, contractID, ticketID, assignedOrgID, entity.OrganizationRoleEnumCONTRACTOR, entity.MemberRoleEnumMEMBER, uuid.New())
		_, err := NewService(store).CanAddTicketLog(context.Background(), userID, ticketID)
		assertPolicyError(t, err, "GOV_PARTNER_OR_EXECUTING_AGENCY_REQUIRED")
	})
}

func TestPaymentPolicies(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()
	requestID := uuid.New()

	tests := []struct {
		name    string
		role    entity.OrganizationRoleEnum
		call    func(*Service) error
		wantErr string
	}{
		{
			name: "view payment request allows participant",
			role: entity.OrganizationRoleEnumCONTRACTOR,
			call: func(s *Service) error {
				return s.CanViewPaymentRequest(context.Background(), userID, requestID)
			},
		},
		{
			name: "auditor approval allows auditor",
			role: entity.OrganizationRoleEnumINDEPENDENTAUDITOR,
			call: func(s *Service) error {
				_, err := s.CanApprovePaymentAsAuditor(context.Background(), userID, requestID)
				return err
			},
		},
		{
			name: "auditor approval rejects executing agency",
			role: entity.OrganizationRoleEnumEXECUTINGAGENCY,
			call: func(s *Service) error {
				_, err := s.CanApprovePaymentAsAuditor(context.Background(), userID, requestID)
				return err
			},
			wantErr: "AUDITOR_REQUIRED",
		},
		{
			name: "agency approval allows executing agency",
			role: entity.OrganizationRoleEnumEXECUTINGAGENCY,
			call: func(s *Service) error {
				_, err := s.CanApprovePaymentAsAgency(context.Background(), userID, requestID)
				return err
			},
		},
		{
			name: "disburse allows loan provider",
			role: entity.OrganizationRoleEnumLOANPROVIDER,
			call: func(s *Service) error {
				_, err := s.CanDisbursePayment(context.Background(), userID, requestID)
				return err
			},
		},
		{
			name: "disburse rejects contractor",
			role: entity.OrganizationRoleEnumCONTRACTOR,
			call: func(s *Service) error {
				_, err := s.CanDisbursePayment(context.Background(), userID, requestID)
				return err
			},
			wantErr: "LOAN_PROVIDER_REQUIRED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{
				payment: entity.GetPaymentRequestWithContractRow{ID: requestID, ProjectID: projectID},
				access:  accessRow(projectID, tt.role, entity.MemberRoleEnumMEMBER, uuid.New()),
			}

			err := tt.call(NewService(store))
			assertPolicyError(t, err, tt.wantErr)
		})
	}
}

func TestUserHasOrgRolePropagatesStoreError(t *testing.T) {
	svc := NewService(&fakeStore{membershipErr: errors.New("db down")})

	_, _, err := svc.UserHasOrgRole(context.Background(), uuid.New(), entity.OrganizationRoleEnumGOVSUPERVISOR, true)
	if err == nil {
		t.Fatal("expected store error")
	}
	var appErr *errs.AppError
	if !errors.As(err, &appErr) || appErr.Code != 500 {
		t.Fatalf("err = %v, want internal app error", err)
	}
}

func accessRow(projectID uuid.UUID, role entity.OrganizationRoleEnum, memberRole entity.MemberRoleEnum, orgID uuid.UUID) entity.GetUserProjectAccessRow {
	return entity.GetUserProjectAccessRow{
		ProjectID:       projectID,
		ContractID:      uuid.New(),
		ParticipantRole: string(role),
		MemberRole:      memberRole,
		OrganizationID:  orgID,
		OrgName:         "org",
		OrgType:         role,
	}
}

func ticketStore(projectID, contractID, ticketID, assignedOrgID uuid.UUID, role entity.OrganizationRoleEnum, memberRole entity.MemberRoleEnum, orgID uuid.UUID) *fakeStore {
	assignTo := uuid.NullUUID{}
	if assignedOrgID != uuid.Nil {
		assignTo = uuid.NullUUID{UUID: assignedOrgID, Valid: true}
	}
	return &fakeStore{
		ticket: entity.GetTicketWithContractRow{
			ID:               ticketID,
			ContractID:       contractID,
			TicketAssignToID: assignTo,
		},
		contract: entity.Contract{ID: contractID, ProjectID: projectID},
		access:   accessRow(projectID, role, memberRole, orgID),
	}
}

func assertPolicyError(t *testing.T, err error, wantMessage string) {
	t.Helper()

	if wantMessage == "" {
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		return
	}
	if err == nil {
		t.Fatalf("err = nil, want %s", wantMessage)
	}
	var appErr *errs.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("err = %T %v, want AppError", err, err)
	}
	if appErr.Message != wantMessage {
		t.Fatalf("message = %s, want %s", appErr.Message, wantMessage)
	}
}
