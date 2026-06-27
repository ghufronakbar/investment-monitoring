package tracker_controller

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"investment_monitoring/internal/http/internal_middleware"
	token_service "investment_monitoring/internal/module/support/token/service"
	tracker_service "investment_monitoring/internal/module/tracker/service"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/internal/testutil"

	"github.com/google/uuid"
)

type fakeTrackerService struct {
	createFn func(context.Context, tracker_service.CreateTicketParams) (tracker_service.TicketReturn, error)
	listFn   func(context.Context, tracker_service.GetTicketsParams) ([]tracker_service.TicketReturn, error)
	getFn    func(context.Context, tracker_service.GetTicketByIDParams) (tracker_service.TicketReturn, error)
	assignFn func(context.Context, tracker_service.AssignTicketParams) (tracker_service.TicketReturn, error)
	logFn    func(context.Context, tracker_service.AddTicketLogParams) (tracker_service.TicketLogReturn, error)
	closeFn  func(context.Context, tracker_service.CloseTicketParams) (tracker_service.TicketReturn, error)
}

func (f fakeTrackerService) CreateTicket(ctx context.Context, p tracker_service.CreateTicketParams) (tracker_service.TicketReturn, error) {
	return f.createFn(ctx, p)
}
func (f fakeTrackerService) GetTickets(ctx context.Context, p tracker_service.GetTicketsParams) ([]tracker_service.TicketReturn, error) {
	return f.listFn(ctx, p)
}
func (f fakeTrackerService) GetTicketByID(ctx context.Context, p tracker_service.GetTicketByIDParams) (tracker_service.TicketReturn, error) {
	return f.getFn(ctx, p)
}
func (f fakeTrackerService) AssignTicket(ctx context.Context, p tracker_service.AssignTicketParams) (tracker_service.TicketReturn, error) {
	return f.assignFn(ctx, p)
}
func (f fakeTrackerService) AddLog(ctx context.Context, p tracker_service.AddTicketLogParams) (tracker_service.TicketLogReturn, error) {
	return f.logFn(ctx, p)
}
func (f fakeTrackerService) CloseTicket(ctx context.Context, p tracker_service.CloseTicketParams) (tracker_service.TicketReturn, error) {
	return f.closeFn(ctx, p)
}

type fakePolicyStore struct {
	entity.Store
	projectID  uuid.UUID
	contractID uuid.UUID
	orgID      uuid.UUID
	role       entity.OrganizationRoleEnum
	member     entity.MemberRoleEnum
	assignTo   uuid.UUID
}

func (f fakePolicyStore) GetUserProjectAccess(ctx context.Context, arg entity.GetUserProjectAccessParams) (entity.GetUserProjectAccessRow, error) {
	return entity.GetUserProjectAccessRow{
		ProjectID:       f.projectID,
		ContractID:      f.contractID,
		ParticipantRole: string(f.role),
		MemberRole:      f.member,
		OrganizationID:  f.orgID,
		OrgName:         "Org",
		OrgType:         f.role,
	}, nil
}

func (f fakePolicyStore) GetTicketWithContract(ctx context.Context, id uuid.UUID) (entity.GetTicketWithContractRow, error) {
	assignTo := uuid.NullUUID{}
	if f.assignTo != uuid.Nil {
		assignTo = uuid.NullUUID{UUID: f.assignTo, Valid: true}
	}
	return entity.GetTicketWithContractRow{ID: id, ContractID: f.contractID, TicketAssignToID: assignTo}, nil
}

func (f fakePolicyStore) GetContractByID(ctx context.Context, id uuid.UUID) (entity.Contract, error) {
	return entity.Contract{ID: id, ProjectID: f.projectID}, nil
}

func TestTrackerGetTicketsContract(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()
	contractID := uuid.New()
	orgID := uuid.New()
	ticketID := uuid.New()
	handler := NewHandler(fakeTrackerService{
		listFn: func(ctx context.Context, p tracker_service.GetTicketsParams) ([]tracker_service.TicketReturn, error) {
			if p.ProjectID != projectID || p.StatusFilter != "open" {
				t.Fatalf("params = %+v, want project/status", p)
			}
			return []tracker_service.TicketReturn{{ID: ticketID, Title: "Ticket", CurrentStatus: "OPEN"}}, nil
		},
	}, fakePolicyStore{projectID: projectID, contractID: contractID, orgID: orgID, role: entity.OrganizationRoleEnumCONTRACTOR, member: entity.MemberRoleEnumMEMBER})

	rr := httptest.NewRecorder()
	req := requestWithUser(http.MethodGet, "/project/"+projectID.String()+"/ticket?status=open", nil, userID)
	handler.Routes().ServeHTTP(rr, req)

	testutil.AssertResponse(t, rr, http.StatusOK, "SUCCESS_GET_TICKETS")
}

func TestTrackerCreateTicketContract(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()
	contractID := uuid.New()
	orgID := uuid.New()
	handler := NewHandler(fakeTrackerService{
		createFn: func(ctx context.Context, p tracker_service.CreateTicketParams) (tracker_service.TicketReturn, error) {
			if p.ProjectID != projectID || p.ContractorID != orgID || p.UserID != userID || p.Title != "Ticket" {
				t.Fatalf("params = %+v, want route/auth/body values", p)
			}
			return tracker_service.TicketReturn{ID: uuid.New(), Title: p.Title}, nil
		},
	}, fakePolicyStore{projectID: projectID, contractID: contractID, orgID: orgID, role: entity.OrganizationRoleEnumCONTRACTOR, member: entity.MemberRoleEnumMEMBER})

	t.Run("success", func(t *testing.T) {
		body := `{"title":"Ticket","description":"Desc","location":"Jakarta","evidenceImageUrls":["https://example.com/image.png"]}`
		rr := httptest.NewRecorder()
		req := requestWithUser(http.MethodPost, "/project/"+projectID.String()+"/ticket", strings.NewReader(body), userID)
		handler.Routes().ServeHTTP(rr, req)
		testutil.AssertResponse(t, rr, http.StatusOK, "SUCCESS_CREATE_TICKET")
	})

	t.Run("validation", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := requestWithUser(http.MethodPost, "/project/"+projectID.String()+"/ticket", strings.NewReader(`{"title":""}`), userID)
		handler.Routes().ServeHTTP(rr, req)
		testutil.AssertResponse(t, rr, http.StatusBadRequest, "VALIDATION_FAILED")
	})
}

func TestTrackerAssignAndAddLogContracts(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()
	contractID := uuid.New()
	orgID := uuid.New()
	ticketID := uuid.New()
	assignToOrgID := uuid.New()
	handler := NewHandler(fakeTrackerService{
		assignFn: func(ctx context.Context, p tracker_service.AssignTicketParams) (tracker_service.TicketReturn, error) {
			if p.TicketID != ticketID || p.AssignToOrgID != assignToOrgID || p.OrgID != orgID || p.UserID != userID {
				t.Fatalf("assign params = %+v, want route/auth/body values", p)
			}
			return tracker_service.TicketReturn{ID: ticketID, CurrentStatus: "ASSIGNED"}, nil
		},
		logFn: func(ctx context.Context, p tracker_service.AddTicketLogParams) (tracker_service.TicketLogReturn, error) {
			if p.TicketID != ticketID || p.OrgID != orgID || p.UserID != userID || p.Description != "Started" {
				t.Fatalf("log params = %+v, want route/auth/body values", p)
			}
			return tracker_service.TicketLogReturn{ID: uuid.New(), Description: p.Description}, nil
		},
	}, fakePolicyStore{
		projectID:  projectID,
		contractID: contractID,
		orgID:      orgID,
		role:       entity.OrganizationRoleEnumEXECUTINGAGENCY,
		member:     entity.MemberRoleEnumADMIN,
	})

	rr := httptest.NewRecorder()
	req := requestWithUser(http.MethodPut, "/ticket/"+ticketID.String()+"/assign", strings.NewReader(`{"assignToOrgId":"`+assignToOrgID.String()+`"}`), userID)
	handler.Routes().ServeHTTP(rr, req)
	testutil.AssertResponse(t, rr, http.StatusOK, "SUCCESS_ASSIGN_TICKET")

	rr = httptest.NewRecorder()
	req = requestWithUser(http.MethodPost, "/ticket/"+ticketID.String()+"/log", strings.NewReader(`{"description":"Started","newStatus":"IN_PROGRESS"}`), userID)
	handler.Routes().ServeHTTP(rr, req)
	testutil.AssertResponse(t, rr, http.StatusOK, "SUCCESS_ADD_LOG")
}

func requestWithUser(method, target string, body io.Reader, userID uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, target, body)
	claims := &token_service.AccessTokenClaims{ID: userID, Email: "user@example.com"}
	return req.WithContext(context.WithValue(req.Context(), internal_middleware.UserContextKey, claims))
}
