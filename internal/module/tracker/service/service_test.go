package tracker_service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/internal/testutil"

	"github.com/google/uuid"
)

type fakeStore struct {
	entity.Store

	contract          entity.Contract
	contractErr       error
	tickets           []entity.GetTicketsByContractIDRow
	ticketsErr        error
	openTickets       []entity.GetOpenTicketsByContractIDRow
	openTicketsErr    error
	ticket            entity.GetTicketByIDRow
	ticketErr         error
	logs              []entity.GetTicketLogsByTicketIDRow
	logsErr           error
	latestStatus      entity.TicketStatusEnum
	statusErr         error
	org               entity.Organization
	orgErr            error
	ticketContract    entity.GetTicketWithContractRow
	ticketContractErr error
	govPartners       []entity.GetGovParticipantsByContractIDRow
	govPartnersErr    error
}

func (f *fakeStore) GetContractByProjectID(ctx context.Context, projectID uuid.UUID) (entity.Contract, error) {
	if f.contractErr != nil {
		return entity.Contract{}, f.contractErr
	}
	return f.contract, nil
}

func (f *fakeStore) GetTicketsByContractID(ctx context.Context, contractID uuid.UUID) ([]entity.GetTicketsByContractIDRow, error) {
	if f.ticketsErr != nil {
		return nil, f.ticketsErr
	}
	return f.tickets, nil
}

func (f *fakeStore) GetOpenTicketsByContractID(ctx context.Context, contractID uuid.UUID) ([]entity.GetOpenTicketsByContractIDRow, error) {
	if f.openTicketsErr != nil {
		return nil, f.openTicketsErr
	}
	return f.openTickets, nil
}

func (f *fakeStore) GetTicketByID(ctx context.Context, id uuid.UUID) (entity.GetTicketByIDRow, error) {
	if f.ticketErr != nil {
		return entity.GetTicketByIDRow{}, f.ticketErr
	}
	return f.ticket, nil
}

func (f *fakeStore) GetTicketLogsByTicketID(ctx context.Context, ticketID uuid.UUID) ([]entity.GetTicketLogsByTicketIDRow, error) {
	if f.logsErr != nil {
		return nil, f.logsErr
	}
	return f.logs, nil
}

func (f *fakeStore) GetLatestTicketStatus(ctx context.Context, ticketID uuid.UUID) (entity.TicketStatusEnum, error) {
	if f.statusErr != nil {
		return "", f.statusErr
	}
	return f.latestStatus, nil
}

func (f *fakeStore) GetOrganizationByID(ctx context.Context, id uuid.UUID) (entity.Organization, error) {
	if f.orgErr != nil {
		return entity.Organization{}, f.orgErr
	}
	return f.org, nil
}

func (f *fakeStore) GetTicketWithContract(ctx context.Context, id uuid.UUID) (entity.GetTicketWithContractRow, error) {
	if f.ticketContractErr != nil {
		return entity.GetTicketWithContractRow{}, f.ticketContractErr
	}
	return f.ticketContract, nil
}

func (f *fakeStore) GetGovParticipantsByContractID(ctx context.Context, contractID uuid.UUID) ([]entity.GetGovParticipantsByContractIDRow, error) {
	if f.govPartnersErr != nil {
		return nil, f.govPartnersErr
	}
	return f.govPartners, nil
}

func (f *fakeStore) GetSLARecordByEntity(ctx context.Context, arg entity.GetSLARecordByEntityParams) (entity.SlaRecord, error) {
	return entity.SlaRecord{}, sql.ErrNoRows
}

func TestGetTicketsMapsDefaultAndOpenFilters(t *testing.T) {
	projectID := uuid.New()
	contractID := uuid.New()
	ticketID := uuid.New()
	taskID := uuid.New()
	assigneeID := uuid.New()
	creatorID := uuid.New()
	lastLogAt := time.Now()
	store := &fakeStore{
		contract: entity.Contract{ID: contractID, ProjectID: projectID},
		tickets: []entity.GetTicketsByContractIDRow{{
			ID:                         ticketID,
			Title:                      "Ticket",
			Description:                "Desc",
			Location:                   "Jakarta",
			EvidenceImageUrls:          []string{"https://example.com/image.png"},
			TaskItemID:                 uuid.NullUUID{UUID: taskID, Valid: true},
			ContractID:                 contractID,
			TicketAssignerContractorID: creatorID,
			TicketAssignToID:           uuid.NullUUID{UUID: assigneeID, Valid: true},
			CreatedAt:                  lastLogAt.Add(-time.Hour),
			CreatorOrgName:             "Contractor",
			AssigneeOrgName:            sql.NullString{String: "Gov Partner", Valid: true},
			TaskTitle:                  sql.NullString{String: "WBS Task", Valid: true},
			CurrentStatus:              entity.TicketStatusEnumASSIGNED,
			LastLogAt:                  lastLogAt,
		}},
		openTickets: []entity.GetOpenTicketsByContractIDRow{{
			ID:                         ticketID,
			Title:                      "Ticket",
			Description:                "Desc",
			Location:                   "Jakarta",
			ContractID:                 contractID,
			TicketAssignerContractorID: creatorID,
			CurrentStatus:              entity.TicketStatusEnumOPEN,
			CreatorOrgName:             "Contractor",
			LastLogAt:                  lastLogAt,
		}},
	}

	got, err := NewService(store, nil, nil).GetTickets(context.Background(), GetTicketsParams{ProjectID: projectID})
	if err != nil {
		t.Fatalf("GetTickets returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].ID != ticketID || got[0].TaskItemID == nil || *got[0].TaskItemID != taskID {
		t.Fatalf("ticket = %+v, want mapped task ticket", got[0])
	}
	if got[0].AssignedTo == nil || got[0].AssignedTo.OrgID != assigneeID || got[0].AssignedTo.OrgName != "Gov Partner" {
		t.Fatalf("assigned to = %+v, want mapped assignee", got[0].AssignedTo)
	}
	if got[0].TaskTitle == nil || *got[0].TaskTitle != "WBS Task" {
		t.Fatalf("task title = %v, want WBS Task", got[0].TaskTitle)
	}
	if got[0].LastLogAt == nil || !got[0].LastLogAt.Equal(lastLogAt) {
		t.Fatalf("last log at = %v, want %v", got[0].LastLogAt, lastLogAt)
	}

	open, err := NewService(store, nil, nil).GetTickets(context.Background(), GetTicketsParams{ProjectID: projectID, StatusFilter: "open"})
	if err != nil {
		t.Fatalf("GetTickets open returned error: %v", err)
	}
	if len(open) != 1 || open[0].CurrentStatus != string(entity.TicketStatusEnumOPEN) {
		t.Fatalf("open tickets = %+v, want open status", open)
	}
}

func TestGetTicketByIDMapsLogsAndOptionalFields(t *testing.T) {
	ticketID := uuid.New()
	contractID := uuid.New()
	taskID := uuid.New()
	orgID := uuid.New()
	userID := uuid.New()
	attachmentURL := "https://example.com/proof.png"
	now := time.Now()
	store := &fakeStore{
		ticket: entity.GetTicketByIDRow{
			ID:                         ticketID,
			Title:                      "Ticket",
			Description:                "Desc",
			Location:                   "Jakarta",
			TaskItemID:                 uuid.NullUUID{UUID: taskID, Valid: true},
			ContractID:                 contractID,
			TicketAssignerContractorID: orgID,
			CreatorOrgName:             "Contractor",
			TaskTitle:                  sql.NullString{String: "WBS Task", Valid: true},
			CurrentStatus:              entity.TicketStatusEnumINPROGRESS,
			CreatedAt:                  now,
		},
		logs: []entity.GetTicketLogsByTicketIDRow{{
			ID:             uuid.New(),
			Action:         entity.TicketLogActionEnumSTATUSCHANGED,
			Description:    "Started",
			AttachmentUrl:  sql.NullString{String: attachmentURL, Valid: true},
			PreviousStatus: entity.NullTicketStatusEnum{TicketStatusEnum: entity.TicketStatusEnumASSIGNED, Valid: true},
			NewStatus:      entity.TicketStatusEnumINPROGRESS,
			TicketID:       ticketID,
			OrganizationID: orgID,
			UserID:         userID,
			CreatedAt:      now,
			OrgName:        "Agency",
			UserName:       "User",
		}},
	}

	got, err := NewService(store, nil, nil).GetTicketByID(context.Background(), GetTicketByIDParams{TicketID: ticketID})
	if err != nil {
		t.Fatalf("GetTicketByID returned error: %v", err)
	}
	if got.ID != ticketID || got.CurrentStatus != string(entity.TicketStatusEnumINPROGRESS) {
		t.Fatalf("ticket = %+v, want mapped ticket", got)
	}
	if got.TaskTitle == nil || *got.TaskTitle != "WBS Task" {
		t.Fatalf("task title = %v, want WBS Task", got.TaskTitle)
	}
	if len(got.Logs) != 1 {
		t.Fatalf("logs len = %d, want 1", len(got.Logs))
	}
	if got.Logs[0].AttachmentUrl == nil || *got.Logs[0].AttachmentUrl != attachmentURL {
		t.Fatalf("attachment = %v, want %s", got.Logs[0].AttachmentUrl, attachmentURL)
	}
	if got.Logs[0].PreviousStatus == nil || *got.Logs[0].PreviousStatus != string(entity.TicketStatusEnumASSIGNED) {
		t.Fatalf("previous status = %v, want ASSIGNED", got.Logs[0].PreviousStatus)
	}
}

func TestAssignTicketPreTransactionValidation(t *testing.T) {
	ticketID := uuid.New()
	assignToOrgID := uuid.New()
	contractID := uuid.New()

	t.Run("closed ticket cannot be assigned", func(t *testing.T) {
		store := &fakeStore{latestStatus: entity.TicketStatusEnumCLOSED}
		_, err := NewService(store, nil, nil).AssignTicket(context.Background(), AssignTicketParams{
			TicketID:      ticketID,
			AssignToOrgID: assignToOrgID,
		})
		testutil.AssertAppError(t, err, "TICKET_ALREADY_CLOSED")
	})

	t.Run("assignee must be gov partner", func(t *testing.T) {
		store := &fakeStore{
			latestStatus: entity.TicketStatusEnumOPEN,
			org:          entity.Organization{ID: assignToOrgID, Role: entity.OrganizationRoleEnumCONTRACTOR},
		}
		_, err := NewService(store, nil, nil).AssignTicket(context.Background(), AssignTicketParams{
			TicketID:      ticketID,
			AssignToOrgID: assignToOrgID,
		})
		testutil.AssertAppError(t, err, "ASSIGNEE_MUST_BE_GOV_PARTNER")
	})

	t.Run("assignee must participate in project", func(t *testing.T) {
		store := &fakeStore{
			latestStatus:   entity.TicketStatusEnumOPEN,
			org:            entity.Organization{ID: assignToOrgID, Role: entity.OrganizationRoleEnumGOVPARTNER},
			ticketContract: entity.GetTicketWithContractRow{ID: ticketID, ContractID: contractID},
			govPartners:    []entity.GetGovParticipantsByContractIDRow{{GovParticipantID: uuid.New()}},
		}
		_, err := NewService(store, nil, nil).AssignTicket(context.Background(), AssignTicketParams{
			TicketID:      ticketID,
			AssignToOrgID: assignToOrgID,
		})
		testutil.AssertAppError(t, err, "GOV_PARTNER_NOT_IN_PROJECT")
	})
}

func TestGetTicketsMissingContract(t *testing.T) {
	_, err := NewService(&fakeStore{contractErr: sql.ErrNoRows}, nil, nil).GetTickets(context.Background(), GetTicketsParams{ProjectID: uuid.New()})
	testutil.AssertAppError(t, err, "CONTRACT_NOT_FOUND")
}

func TestTicketLifecycleRejectsOutOfOrderTransitions(t *testing.T) {
	ticketID := uuid.New()
	status := func(s entity.TicketStatusEnum) *string {
		v := string(s)
		return &v
	}

	addLogCases := []struct {
		name    string
		current entity.TicketStatusEnum
		target  entity.TicketStatusEnum
		want    string
	}{
		{"open cannot skip to in progress", entity.TicketStatusEnumOPEN, entity.TicketStatusEnumINPROGRESS, "INVALID_STATUS_TRANSITION"},
		{"open cannot skip to resolved", entity.TicketStatusEnumOPEN, entity.TicketStatusEnumRESOLVED, "INVALID_STATUS_TRANSITION"},
		{"assigned cannot skip to resolved", entity.TicketStatusEnumASSIGNED, entity.TicketStatusEnumRESOLVED, "INVALID_STATUS_TRANSITION"},
		{"status cannot be set back to open", entity.TicketStatusEnumINPROGRESS, entity.TicketStatusEnumOPEN, "INVALID_STATUS_TRANSITION"},
		{"assigned is reachable only by assignment", entity.TicketStatusEnumOPEN, entity.TicketStatusEnumASSIGNED, "INVALID_STATUS_TRANSITION"},
		{"log entry cannot close a ticket", entity.TicketStatusEnumRESOLVED, entity.TicketStatusEnumCLOSED, "CLOSE_VIA_CLOSE_ENDPOINT"},
		{"closed ticket is terminal", entity.TicketStatusEnumCLOSED, entity.TicketStatusEnumINPROGRESS, "TICKET_ALREADY_CLOSED"},
	}
	for _, tt := range addLogCases {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{latestStatus: tt.current}
			_, err := NewService(store, nil, nil).AddLog(context.Background(), AddTicketLogParams{
				TicketID:    ticketID,
				Description: "status update",
				NewStatus:   status(tt.target),
			})
			testutil.AssertAppError(t, err, tt.want)
		})
	}

	for _, current := range []entity.TicketStatusEnum{
		entity.TicketStatusEnumOPEN,
		entity.TicketStatusEnumASSIGNED,
		entity.TicketStatusEnumINPROGRESS,
	} {
		t.Run("close requires resolved from "+string(current), func(t *testing.T) {
			store := &fakeStore{latestStatus: current}
			_, err := NewService(store, nil, nil).CloseTicket(context.Background(), CloseTicketParams{TicketID: ticketID})
			testutil.AssertAppError(t, err, "INVALID_STATUS_TRANSITION")
		})
	}

	t.Run("resolved ticket cannot be reassigned", func(t *testing.T) {
		store := &fakeStore{latestStatus: entity.TicketStatusEnumRESOLVED}
		_, err := NewService(store, nil, nil).AssignTicket(context.Background(), AssignTicketParams{
			TicketID:      ticketID,
			AssignToOrgID: uuid.New(),
		})
		testutil.AssertAppError(t, err, "INVALID_STATUS_TRANSITION")
	})
}
