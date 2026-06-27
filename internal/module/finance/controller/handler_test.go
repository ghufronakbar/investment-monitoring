package finance_controller

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"investment_monitoring/internal/http/internal_middleware"
	finance_service "investment_monitoring/internal/module/finance/service"
	token_service "investment_monitoring/internal/module/support/token/service"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/internal/testutil"

	"github.com/google/uuid"
)

type fakeFinanceService struct {
	createFn   func(context.Context, finance_service.CreatePaymentRequestParams) (finance_service.PaymentRequestReturn, error)
	listFn     func(context.Context, finance_service.GetPaymentRequestsParams) ([]finance_service.PaymentRequestReturn, error)
	getFn      func(context.Context, finance_service.GetPaymentRequestByIDParams) (finance_service.PaymentRequestReturn, error)
	auditorFn  func(context.Context, finance_service.ApprovalParams) (finance_service.PaymentRequestReturn, error)
	agencyFn   func(context.Context, finance_service.ApprovalParams) (finance_service.PaymentRequestReturn, error)
	disburseFn func(context.Context, finance_service.DisburseParams) (finance_service.PaymentRequestReturn, error)
	summaryFn  func(context.Context, finance_service.GetPaymentSummaryParams) (finance_service.PaymentSummaryReturn, error)
}

func (f fakeFinanceService) CreatePaymentRequest(ctx context.Context, p finance_service.CreatePaymentRequestParams) (finance_service.PaymentRequestReturn, error) {
	return f.createFn(ctx, p)
}
func (f fakeFinanceService) GetPaymentRequests(ctx context.Context, p finance_service.GetPaymentRequestsParams) ([]finance_service.PaymentRequestReturn, error) {
	return f.listFn(ctx, p)
}
func (f fakeFinanceService) GetPaymentRequestByID(ctx context.Context, p finance_service.GetPaymentRequestByIDParams) (finance_service.PaymentRequestReturn, error) {
	return f.getFn(ctx, p)
}
func (f fakeFinanceService) ApproveByAuditor(ctx context.Context, p finance_service.ApprovalParams) (finance_service.PaymentRequestReturn, error) {
	return f.auditorFn(ctx, p)
}
func (f fakeFinanceService) ApproveByAgency(ctx context.Context, p finance_service.ApprovalParams) (finance_service.PaymentRequestReturn, error) {
	return f.agencyFn(ctx, p)
}
func (f fakeFinanceService) Disburse(ctx context.Context, p finance_service.DisburseParams) (finance_service.PaymentRequestReturn, error) {
	return f.disburseFn(ctx, p)
}
func (f fakeFinanceService) GetPaymentSummary(ctx context.Context, p finance_service.GetPaymentSummaryParams) (finance_service.PaymentSummaryReturn, error) {
	return f.summaryFn(ctx, p)
}

type fakePolicyStore struct {
	entity.Store
	projectID uuid.UUID
	orgID     uuid.UUID
	role      entity.OrganizationRoleEnum
	member    entity.MemberRoleEnum
	request   entity.GetPaymentRequestWithContractRow
}

func (f fakePolicyStore) GetUserProjectAccess(ctx context.Context, arg entity.GetUserProjectAccessParams) (entity.GetUserProjectAccessRow, error) {
	return entity.GetUserProjectAccessRow{
		ProjectID:       f.projectID,
		ContractID:      uuid.New(),
		ParticipantRole: string(f.role),
		MemberRole:      f.member,
		OrganizationID:  f.orgID,
		OrgName:         "Org",
		OrgType:         f.role,
	}, nil
}

func (f fakePolicyStore) GetPaymentRequestWithContract(ctx context.Context, id uuid.UUID) (entity.GetPaymentRequestWithContractRow, error) {
	if f.request.ID != uuid.Nil {
		return f.request, nil
	}
	return entity.GetPaymentRequestWithContractRow{ID: id, ProjectID: f.projectID}, nil
}

func TestFinanceSummaryContract(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()
	orgID := uuid.New()
	handler := NewHandler(fakeFinanceService{
		summaryFn: func(ctx context.Context, p finance_service.GetPaymentSummaryParams) (finance_service.PaymentSummaryReturn, error) {
			if p.ProjectID != projectID {
				t.Fatalf("project id = %s, want %s", p.ProjectID, projectID)
			}
			return finance_service.PaymentSummaryReturn{ContractValue: "1000", AvailableAmount: "500", CanSubmit: true, Currency: "IDR"}, nil
		},
	}, fakePolicyStore{projectID: projectID, orgID: orgID, role: entity.OrganizationRoleEnumEXECUTINGAGENCY, member: entity.MemberRoleEnumADMIN})

	rr := httptest.NewRecorder()
	req := requestWithUser(http.MethodGet, "/project/"+projectID.String()+"/summary", nil, userID)
	handler.Routes().ServeHTTP(rr, req)

	testutil.AssertResponse(t, rr, http.StatusOK, "SUCCESS_GET_PAYMENT_SUMMARY")
}

func TestFinanceCreatePaymentRequestContract(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()
	orgID := uuid.New()
	handler := NewHandler(fakeFinanceService{
		createFn: func(ctx context.Context, p finance_service.CreatePaymentRequestParams) (finance_service.PaymentRequestReturn, error) {
			if p.ProjectID != projectID || p.ContractorID != orgID || p.UserID != userID || p.Amount != 1000 {
				t.Fatalf("params = %+v, want route/auth/body values", p)
			}
			return finance_service.PaymentRequestReturn{ID: uuid.New(), Title: p.Title, Amount: "1000"}, nil
		},
	}, fakePolicyStore{projectID: projectID, orgID: orgID, role: entity.OrganizationRoleEnumCONTRACTOR, member: entity.MemberRoleEnumMEMBER})

	t.Run("success", func(t *testing.T) {
		body := `{"title":"Termin 1","amount":1000,"invoiceFileUrl":"https://example.com/invoice.pdf"}`
		rr := httptest.NewRecorder()
		req := requestWithUser(http.MethodPost, "/project/"+projectID.String()+"/request", strings.NewReader(body), userID)
		handler.Routes().ServeHTTP(rr, req)
		testutil.AssertResponse(t, rr, http.StatusOK, "SUCCESS_CREATE_PAYMENT_REQUEST")
	})

	t.Run("validation", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := requestWithUser(http.MethodPost, "/project/"+projectID.String()+"/request", strings.NewReader(`{"title":""}`), userID)
		handler.Routes().ServeHTTP(rr, req)
		testutil.AssertResponse(t, rr, http.StatusBadRequest, "VALIDATION_FAILED")
	})
}

func TestFinanceApproveAuditorContract(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()
	orgID := uuid.New()
	requestID := uuid.New()
	handler := NewHandler(fakeFinanceService{
		auditorFn: func(ctx context.Context, p finance_service.ApprovalParams) (finance_service.PaymentRequestReturn, error) {
			if p.RequestID != requestID || p.OrgID != orgID || p.UserID != userID || p.Action != "APPROVE" {
				t.Fatalf("params = %+v, want route/auth/body values", p)
			}
			return finance_service.PaymentRequestReturn{ID: requestID, Status: "AUDITOR_APPROVED"}, nil
		},
	}, fakePolicyStore{
		projectID: projectID,
		orgID:     orgID,
		role:      entity.OrganizationRoleEnumINDEPENDENTAUDITOR,
		member:    entity.MemberRoleEnumMEMBER,
		request:   entity.GetPaymentRequestWithContractRow{ID: requestID, ProjectID: projectID},
	})

	rr := httptest.NewRecorder()
	req := requestWithUser(http.MethodPut, "/request/"+requestID.String()+"/approve-auditor", strings.NewReader(`{"action":"APPROVE"}`), userID)
	handler.Routes().ServeHTTP(rr, req)

	testutil.AssertResponse(t, rr, http.StatusOK, "SUCCESS_APPROVE_BY_AUDITOR")
}

func requestWithUser(method, target string, body io.Reader, userID uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, target, body)
	claims := &token_service.AccessTokenClaims{ID: userID, Email: "user@example.com"}
	return req.WithContext(context.WithValue(req.Context(), internal_middleware.UserContextKey, claims))
}
