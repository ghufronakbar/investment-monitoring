package project_controller

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"investment_monitoring/internal/http/internal_middleware"
	project_service "investment_monitoring/internal/module/project/service"
	token_service "investment_monitoring/internal/module/support/token/service"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/internal/testutil"

	"github.com/google/uuid"
)

type fakeProjectService struct {
	createFn                func(context.Context, project_service.CreateProjectParams, uuid.UUID) (project_service.CreateProjectReturn, error)
	listFn                  func(context.Context, uuid.UUID) ([]project_service.ProjectReturn, error)
	getFn                   func(context.Context, project_service.GetProjectParams) (project_service.ProjectDetailReturn, error)
	configureSettingsFn     func(context.Context, project_service.ConfigureSettingsParams) (project_service.ProjectSettingsReturn, error)
	assignExecutingAgencyFn func(context.Context, project_service.AssignExecutingAgencyParams) (project_service.AssignmentReturn, error)
	assignVendorFn          func(context.Context, project_service.AssignVendorParams) (project_service.AssignmentReturn, error)
	assignAuditorFn         func(context.Context, project_service.AssignAuditorParams) (project_service.AssignmentReturn, error)
	assignGovPartnersFn     func(context.Context, project_service.AssignGovPartnersParams) (project_service.AssignmentReturn, error)
}

func (f fakeProjectService) Create(ctx context.Context, p project_service.CreateProjectParams, orgID uuid.UUID) (project_service.CreateProjectReturn, error) {
	return f.createFn(ctx, p, orgID)
}
func (f fakeProjectService) ListAccessible(ctx context.Context, userID uuid.UUID) ([]project_service.ProjectReturn, error) {
	return f.listFn(ctx, userID)
}
func (f fakeProjectService) GetByID(ctx context.Context, p project_service.GetProjectParams) (project_service.ProjectDetailReturn, error) {
	return f.getFn(ctx, p)
}
func (f fakeProjectService) ConfigureSettings(ctx context.Context, p project_service.ConfigureSettingsParams) (project_service.ProjectSettingsReturn, error) {
	return f.configureSettingsFn(ctx, p)
}
func (f fakeProjectService) AssignExecutingAgency(ctx context.Context, p project_service.AssignExecutingAgencyParams) (project_service.AssignmentReturn, error) {
	return f.assignExecutingAgencyFn(ctx, p)
}
func (f fakeProjectService) AssignVendor(ctx context.Context, p project_service.AssignVendorParams) (project_service.AssignmentReturn, error) {
	return f.assignVendorFn(ctx, p)
}
func (f fakeProjectService) AssignAuditor(ctx context.Context, p project_service.AssignAuditorParams) (project_service.AssignmentReturn, error) {
	return f.assignAuditorFn(ctx, p)
}
func (f fakeProjectService) AssignGovPartners(ctx context.Context, p project_service.AssignGovPartnersParams) (project_service.AssignmentReturn, error) {
	return f.assignGovPartnersFn(ctx, p)
}

type fakePolicyStore struct {
	entity.Store
	userID    uuid.UUID
	projectID uuid.UUID
	orgID     uuid.UUID
	role      entity.OrganizationRoleEnum
	member    entity.MemberRoleEnum
}

func (f fakePolicyStore) GetUserMemberships(ctx context.Context, userID uuid.UUID) ([]entity.GetUserMembershipsRow, error) {
	return []entity.GetUserMembershipsRow{{
		OrgID:      f.orgID,
		OrgRole:    f.role,
		MemberRole: f.member,
	}}, nil
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

func TestProjectListContract(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()
	handler := NewHandler(fakeProjectService{
		listFn: func(ctx context.Context, gotUserID uuid.UUID) ([]project_service.ProjectReturn, error) {
			if gotUserID != userID {
				t.Fatalf("user id = %s, want %s", gotUserID, userID)
			}
			return []project_service.ProjectReturn{{ID: projectID, Name: "Project"}}, nil
		},
	}, fakePolicyStore{})

	rr := httptest.NewRecorder()
	req := requestWithUser(http.MethodGet, "/", nil, userID)
	handler.Routes().ServeHTTP(rr, req)

	testutil.AssertResponse(t, rr, http.StatusOK, "SUCCESS_LIST_PROJECTS")
	var body testutil.ResponseBody
	testutil.DecodeBody(t, rr, &body)
	items := body.Data.([]any)
	if len(items) != 1 || items[0].(map[string]any)["name"] != "Project" {
		t.Fatalf("data = %#v, want project list", body.Data)
	}
}

func TestProjectGetByIDContract(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()
	orgID := uuid.New()
	handler := NewHandler(fakeProjectService{
		getFn: func(ctx context.Context, p project_service.GetProjectParams) (project_service.ProjectDetailReturn, error) {
			if p.ProjectID != projectID || p.UserID != userID {
				t.Fatalf("params = %+v, want project/user", p)
			}
			return project_service.ProjectDetailReturn{Project: project_service.ProjectReturn{ID: projectID, Name: "Project"}}, nil
		},
	}, fakePolicyStore{userID: userID, projectID: projectID, orgID: orgID, role: entity.OrganizationRoleEnumCONTRACTOR, member: entity.MemberRoleEnumMEMBER})

	rr := httptest.NewRecorder()
	req := requestWithUser(http.MethodGet, "/"+projectID.String()+"/", nil, userID)
	handler.Routes().ServeHTTP(rr, req)

	testutil.AssertResponse(t, rr, http.StatusOK, "SUCCESS_GET_PROJECT")
}

func TestProjectCreateAndValidationContracts(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	projectID := uuid.New()
	loanProviderID := uuid.New()
	handler := NewHandler(fakeProjectService{
		createFn: func(ctx context.Context, p project_service.CreateProjectParams, callerOrgID uuid.UUID) (project_service.CreateProjectReturn, error) {
			if p.UserID != userID || callerOrgID != orgID || p.LoanProviderID != loanProviderID {
				t.Fatalf("params = %+v org=%s, want user/org/loan provider", p, callerOrgID)
			}
			return project_service.CreateProjectReturn{Project: project_service.ProjectReturn{ID: projectID, Name: p.Name}}, nil
		},
	}, fakePolicyStore{userID: userID, orgID: orgID, role: entity.OrganizationRoleEnumGOVSUPERVISOR, member: entity.MemberRoleEnumADMIN})

	t.Run("success", func(t *testing.T) {
		body := `{"name":"Project","description":"Desc","contractFileUrl":"https://example.com/contract.pdf","loanProviderId":"` + loanProviderID.String() + `","imageUrls":["https://example.com/image.png"]}`
		rr := httptest.NewRecorder()
		req := requestWithUser(http.MethodPost, "/", strings.NewReader(body), userID)
		handler.Routes().ServeHTTP(rr, req)
		testutil.AssertResponse(t, rr, http.StatusOK, "SUCCESS_CREATE_PROJECT")
	})

	t.Run("validation", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := requestWithUser(http.MethodPost, "/", strings.NewReader(`{"name":""}`), userID)
		handler.Routes().ServeHTTP(rr, req)
		testutil.AssertResponse(t, rr, http.StatusBadRequest, "VALIDATION_FAILED")
	})
}

func TestProjectAssignVendorContract(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()
	orgID := uuid.New()
	vendorID := uuid.New()
	handler := NewHandler(fakeProjectService{
		assignVendorFn: func(ctx context.Context, p project_service.AssignVendorParams) (project_service.AssignmentReturn, error) {
			if p.ProjectID != projectID || p.UserID != userID || p.ActorOrgID != orgID || p.VendorOrgID != vendorID {
				t.Fatalf("params = %+v, want route/auth/body values", p)
			}
			return project_service.AssignmentReturn{Success: true, Message: "VENDOR_ASSIGNED"}, nil
		},
	}, fakePolicyStore{userID: userID, projectID: projectID, orgID: orgID, role: entity.OrganizationRoleEnumEXECUTINGAGENCY, member: entity.MemberRoleEnumADMIN})

	body := `{"vendorOrgId":"` + vendorID.String() + `","contractValue":1000,"contractDocUrl":"https://example.com/contract.pdf"}`
	rr := httptest.NewRecorder()
	req := requestWithUser(http.MethodPut, "/"+projectID.String()+"/contract/assign-vendor", strings.NewReader(body), userID)
	handler.Routes().ServeHTTP(rr, req)

	testutil.AssertResponse(t, rr, http.StatusOK, "SUCCESS_ASSIGN_VENDOR")
}

func requestWithUser(method, target string, body io.Reader, userID uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, target, body)
	claims := &token_service.AccessTokenClaims{ID: userID, Email: "user@example.com"}
	return req.WithContext(context.WithValue(req.Context(), internal_middleware.UserContextKey, claims))
}
