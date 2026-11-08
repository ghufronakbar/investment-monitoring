//go:build integration

package testutil

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"testing"

	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// OpenTestDB connects to the disposable database named by TEST_DATABASE_URL.
// Integration tests write fixtures with random identifiers and never clean up,
// so the URL must point to a throwaway database (see scripts/test-integration.sh).
func OpenTestDB(t *testing.T) (*sql.DB, *entity.SQLStore) {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, entity.NewStore(db)
}

// StatusOf maps a policy or service error to the HTTP status the handler
// would return, so integration tests can assert on the API contract.
func StatusOf(err error) int {
	if err == nil {
		return http.StatusOK
	}
	var appErr *errs.AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return http.StatusInternalServerError
}

// Actor is a user acting on behalf of one organization membership.
type Actor struct {
	UserID uuid.UUID
	OrgID  uuid.UUID
}

// Fixture builds an isolated cross-institutional project ecosystem.
type Fixture struct {
	t   *testing.T
	db  *sql.DB
	ctx context.Context
}

func NewFixture(t *testing.T, db *sql.DB) *Fixture {
	return &Fixture{t: t, db: db, ctx: context.Background()}
}

func (f *Fixture) exec(query string, args ...any) {
	f.t.Helper()
	if _, err := f.db.ExecContext(f.ctx, query, args...); err != nil {
		f.t.Fatalf("fixture exec: %v\nquery: %s", err, query)
	}
}

func (f *Fixture) Org(role entity.OrganizationRoleEnum) uuid.UUID {
	f.t.Helper()
	id := uuid.New()
	f.exec(`INSERT INTO organizations (id, name, role) VALUES ($1, $2, $3)`, id, "Org "+string(role)+" "+id.String()[:8], role)
	return id
}

func (f *Fixture) User() uuid.UUID {
	f.t.Helper()
	id := uuid.New()
	f.exec(`INSERT INTO users (id, name, email) VALUES ($1, $2, $3)`, id, "User "+id.String()[:8], id.String()+"@test.local")
	return id
}

// Member adds a user to an organization and returns the acting identity.
func (f *Fixture) Member(orgID uuid.UUID, role entity.MemberRoleEnum) Actor {
	f.t.Helper()
	userID := f.User()
	f.exec(`INSERT INTO organization_members (role, user_id, organization_id) VALUES ($1, $2, $3)`, role, userID, orgID)
	return Actor{UserID: userID, OrgID: orgID}
}

// FormerMember adds a membership that has been soft-deleted.
func (f *Fixture) FormerMember(orgID uuid.UUID) Actor {
	f.t.Helper()
	userID := f.User()
	f.exec(`INSERT INTO organization_members (role, user_id, organization_id, deleted_at) VALUES ('ADMIN', $1, $2, NOW())`, userID, orgID)
	return Actor{UserID: userID, OrgID: orgID}
}

// Participants are the organizations holding each contractual position.
type Participants struct {
	GovSupervisor   uuid.UUID
	LoanProvider    uuid.UUID
	ExecutingAgency uuid.UUID
	Vendor          uuid.UUID
	Auditor         uuid.UUID
	GovPartners     []uuid.UUID
}

type Project struct {
	ID         uuid.UUID
	ContractID uuid.UUID
	TaskID     uuid.UUID
}

func (f *Fixture) Project(p Participants) Project {
	f.t.Helper()
	pr := Project{ID: uuid.New(), ContractID: uuid.New(), TaskID: uuid.New()}
	f.exec(`INSERT INTO projects (id, name, description, contract_file_url) VALUES ($1, $2, 'fixture', 'https://example.test/contract.pdf')`,
		pr.ID, "Project "+pr.ID.String()[:8])
	f.exec(`INSERT INTO contracts (id, description, project_id, gov_supervisor_participant_id, loan_provider_participant_id,
		executing_agency_participant_id, vendor_participant_id, auditor_participant_id)
		VALUES ($1, 'fixture', $2, $3, $4, $5, $6, $7)`,
		pr.ContractID, pr.ID, p.GovSupervisor, p.LoanProvider, p.ExecutingAgency, p.Vendor, p.Auditor)
	for _, gp := range p.GovPartners {
		f.exec(`INSERT INTO gov_participants (contract_id, gov_participant_id) VALUES ($1, $2)`, pr.ContractID, gp)
	}
	f.exec(`INSERT INTO task_items (id, title, description, deadline, weight, contract_id) VALUES ($1, 'Task', 'fixture', NOW() + INTERVAL '30 days', 100, $2)`,
		pr.TaskID, pr.ContractID)
	return pr
}

func (f *Fixture) PaymentRequest(contractID uuid.UUID, status entity.PaymentRequestStatusEnum) uuid.UUID {
	f.t.Helper()
	id := uuid.New()
	f.exec(`INSERT INTO payment_requests (id, title, amount, status, invoice_file_url, contract_id) VALUES ($1, 'Termin', 1000000, $2, 'https://example.test/invoice.pdf', $3)`,
		id, status, contractID)
	return id
}

func (f *Fixture) ProgressLog(taskID, vendorOrgID uuid.UUID) uuid.UUID {
	f.t.Helper()
	id := uuid.New()
	f.exec(`INSERT INTO progress_logs (id, report, progress_percentage, snapshot_title, snapshot_description, snapshot_weight, task_item_id, vendor_id)
		VALUES ($1, 'fixture', 10, 'Task', 'fixture', 100, $2, $3)`,
		id, taskID, vendorOrgID)
	return id
}

// Ticket creates an obstacle ticket raised by the contractor, optionally assigned.
func (f *Fixture) Ticket(contractID uuid.UUID, reporter Actor, assignTo uuid.NullUUID) uuid.UUID {
	f.t.Helper()
	id := uuid.New()
	f.exec(`INSERT INTO tickets (id, title, description, location, contract_id, ticket_assigner_contractor_id, ticket_assigner_user_id, ticket_assign_to_id)
		VALUES ($1, 'Obstacle', 'fixture', 'Site', $2, $3, $4, $5)`,
		id, contractID, reporter.OrgID, reporter.UserID, assignTo)
	f.TicketStatus(id, reporter, entity.TicketStatusEnumOPEN)
	return id
}

// TicketStatus appends a status log; the latest log is the ticket's current status.
func (f *Fixture) TicketStatus(ticketID uuid.UUID, actor Actor, status entity.TicketStatusEnum) {
	f.t.Helper()
	f.exec(`INSERT INTO ticket_logs (action, description, new_status, ticket_id, organization_id, user_id, created_at)
		VALUES ('STATUS_CHANGED', 'fixture', $1, $2, $3, $4, clock_timestamp())`,
		status, ticketID, actor.OrgID, actor.UserID)
}
