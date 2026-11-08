//go:build integration

package integration

import (
	"context"
	"database/sql"
	"net/http"
	"testing"

	finance_service "investment_monitoring/internal/module/finance/service"
	monitoring_service "investment_monitoring/internal/module/monitoring/service"
	tracker_service "investment_monitoring/internal/module/tracker/service"
	"investment_monitoring/internal/policy"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/internal/testutil"

	"github.com/google/uuid"
)

// matrixAction is one governed action: the policy check the handler performs
// first, followed by the service call that applies the FSM transition.
type matrixAction struct {
	name    string
	roles   []string        // contractual positions allowed to perform it
	allowed map[string]bool // source states from which the FSM accepts it
	run     func(actor testutil.Actor, entityID uuid.UUID) error
}

type matrixResult struct {
	cells, ok, badRequest, forbidden int
}

// runMatrix executes every (state, action, actor) cell on a fresh entity and
// compares the outcome with the specification: 403 when the actor's position
// is not authorized, otherwise 200 or 400 depending on the transition matrix.
func runMatrix(t *testing.T, workflow string, states []string, actors map[string]testutil.Actor,
	actions []matrixAction, newEntity func(state string) uuid.UUID) matrixResult {
	t.Helper()

	var res matrixResult
	for _, state := range states {
		for _, action := range actions {
			for position, actor := range actors {
				want := http.StatusForbidden
				if contains(action.roles, position) {
					want = http.StatusBadRequest
					if action.allowed[state] {
						want = http.StatusOK
					}
				}

				got := testutil.StatusOf(action.run(actor, newEntity(state)))
				res.cells++
				switch got {
				case http.StatusOK:
					res.ok++
				case http.StatusBadRequest:
					res.badRequest++
				case http.StatusForbidden:
					res.forbidden++
				}
				if got != want {
					t.Errorf("%s: state=%s action=%s actor=%s status=%d want=%d", workflow, state, action.name, position, got, want)
				}
			}
		}
	}
	t.Logf("MATRIX %-10s states=%d actions=%d actors=%d cells=%d ok=%d rejected_fsm_400=%d rejected_policy_403=%d",
		workflow, len(states), len(actions), len(actors), res.cells, res.ok, res.badRequest, res.forbidden)
	return res
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func set(states ...string) map[string]bool {
	m := make(map[string]bool, len(states))
	for _, s := range states {
		m[s] = true
	}
	return m
}

// projectWithAllPositions creates one project whose six contractual positions
// are each held by a separate organization, with one admin acting for each.
func projectWithAllPositions(f *testutil.Fixture) (testutil.Project, map[string]testutil.Actor) {
	orgs := map[string]uuid.UUID{
		"GOV_SUPERVISOR":      f.Org(entity.OrganizationRoleEnumGOVSUPERVISOR),
		"LOAN_PROVIDER":       f.Org(entity.OrganizationRoleEnumLOANPROVIDER),
		"EXECUTING_AGENCY":    f.Org(entity.OrganizationRoleEnumEXECUTINGAGENCY),
		"CONTRACTOR":          f.Org(entity.OrganizationRoleEnumCONTRACTOR),
		"INDEPENDENT_AUDITOR": f.Org(entity.OrganizationRoleEnumINDEPENDENTAUDITOR),
		"GOV_PARTNER":         f.Org(entity.OrganizationRoleEnumGOVPARTNER),
	}
	project := f.Project(testutil.Participants{
		GovSupervisor:   orgs["GOV_SUPERVISOR"],
		LoanProvider:    orgs["LOAN_PROVIDER"],
		ExecutingAgency: orgs["EXECUTING_AGENCY"],
		Vendor:          orgs["CONTRACTOR"],
		Auditor:         orgs["INDEPENDENT_AUDITOR"],
		GovPartners:     []uuid.UUID{orgs["GOV_PARTNER"]},
	})
	actors := make(map[string]testutil.Actor, len(orgs))
	for position, org := range orgs {
		actors[position] = f.Member(org, entity.MemberRoleEnumADMIN)
	}
	return project, actors
}

// TestWorkflowStateActionActorMatrix exhaustively combines every workflow
// state, governed action, and contractual position for the three workflows.
func TestWorkflowStateActionActorMatrix(t *testing.T) {
	db, store := testutil.OpenTestDB(t)
	f := testutil.NewFixture(t, db)
	ctx := context.Background()
	pol := policy.NewService(store)
	project, actors := projectWithAllPositions(f)

	var total matrixResult
	add := func(r matrixResult) {
		total.cells += r.cells
		total.ok += r.ok
		total.badRequest += r.badRequest
		total.forbidden += r.forbidden
	}

	t.Run("finance", func(t *testing.T) {
		svc := finance_service.NewService(store, nil, nil)
		approve := func(check func(context.Context, uuid.UUID, uuid.UUID) (*policy.ProjectAccess, error),
			apply func(context.Context, finance_service.ApprovalParams) (finance_service.PaymentRequestReturn, error),
			decision string) func(testutil.Actor, uuid.UUID) error {
			return func(a testutil.Actor, id uuid.UUID) error {
				access, err := check(ctx, a.UserID, id)
				if err != nil {
					return err
				}
				_, err = apply(ctx, finance_service.ApprovalParams{RequestID: id, OrgID: access.OrganizationID, UserID: a.UserID, Action: decision})
				return err
			}
		}
		actions := []matrixAction{
			{"AUDITOR_APPROVE", []string{"INDEPENDENT_AUDITOR"}, set("SUBMITTED"),
				approve(pol.CanApprovePaymentAsAuditor, svc.ApproveByAuditor, "APPROVE")},
			{"AUDITOR_REJECT", []string{"INDEPENDENT_AUDITOR"}, set("SUBMITTED"),
				approve(pol.CanApprovePaymentAsAuditor, svc.ApproveByAuditor, "REJECT")},
			{"AGENCY_APPROVE", []string{"EXECUTING_AGENCY"}, set("AUDITOR_APPROVED"),
				approve(pol.CanApprovePaymentAsAgency, svc.ApproveByAgency, "APPROVE")},
			{"AGENCY_REJECT", []string{"EXECUTING_AGENCY"}, set("AUDITOR_APPROVED"),
				approve(pol.CanApprovePaymentAsAgency, svc.ApproveByAgency, "REJECT")},
			{"DISBURSE", []string{"LOAN_PROVIDER"}, set("AGENCY_APPROVED"),
				func(a testutil.Actor, id uuid.UUID) error {
					access, err := pol.CanDisbursePayment(ctx, a.UserID, id)
					if err != nil {
						return err
					}
					_, err = svc.Disburse(ctx, finance_service.DisburseParams{
						RequestID: id, OrgID: access.OrganizationID, UserID: a.UserID,
						PaymentProofImageUrls: []string{"https://example.test/proof.png"},
					})
					return err
				}},
		}
		states := []string{"SUBMITTED", "AUDITOR_APPROVED", "AUDITOR_REJECTED", "AGENCY_APPROVED", "AGENCY_REJECTED", "PAID"}
		add(runMatrix(t, "finance", states, actors, actions, func(state string) uuid.UUID {
			return f.PaymentRequest(project.ContractID, entity.PaymentRequestStatusEnum(state))
		}))
	})

	t.Run("tracker", func(t *testing.T) {
		svc := tracker_service.NewService(store, nil, nil)
		gp := actors["GOV_PARTNER"]
		status := func(s string) *string { return &s }
		addLog := func(newStatus *string) func(testutil.Actor, uuid.UUID) error {
			return func(a testutil.Actor, id uuid.UUID) error {
				access, err := pol.CanAddTicketLog(ctx, a.UserID, id)
				if err != nil {
					return err
				}
				_, err = svc.AddLog(ctx, tracker_service.AddTicketLogParams{
					TicketID: id, OrgID: access.OrganizationID, UserID: a.UserID,
					Description: "matrix", NewStatus: newStatus,
				})
				return err
			}
		}
		updaters := []string{"EXECUTING_AGENCY", "GOV_PARTNER"}
		actions := []matrixAction{
			{"ASSIGN", []string{"EXECUTING_AGENCY"}, set("OPEN", "ASSIGNED", "IN_PROGRESS"),
				func(a testutil.Actor, id uuid.UUID) error {
					access, err := pol.CanAssignTicket(ctx, a.UserID, id)
					if err != nil {
						return err
					}
					_, err = svc.AssignTicket(ctx, tracker_service.AssignTicketParams{
						TicketID: id, AssignToOrgID: gp.OrgID, OrgID: access.OrganizationID, UserID: a.UserID,
					})
					return err
				}},
			{"START_WORK", updaters, set("ASSIGNED", "RESOLVED"), addLog(status("IN_PROGRESS"))},
			{"RESOLVE", updaters, set("IN_PROGRESS"), addLog(status("RESOLVED"))},
			{"COMMENT", updaters, set("OPEN", "ASSIGNED", "IN_PROGRESS", "RESOLVED"), addLog(nil)},
			{"CLOSE", []string{"EXECUTING_AGENCY"}, set("RESOLVED"),
				func(a testutil.Actor, id uuid.UUID) error {
					access, err := pol.CanCloseTicket(ctx, a.UserID, id)
					if err != nil {
						return err
					}
					_, err = svc.CloseTicket(ctx, tracker_service.CloseTicketParams{TicketID: id, OrgID: access.OrganizationID, UserID: a.UserID})
					return err
				}},
		}
		states := []string{"OPEN", "ASSIGNED", "IN_PROGRESS", "RESOLVED", "CLOSED"}
		add(runMatrix(t, "tracker", states, actors, actions, func(state string) uuid.UUID {
			id := f.Ticket(project.ContractID, actors["CONTRACTOR"], uuid.NullUUID{UUID: gp.OrgID, Valid: true})
			if state != "OPEN" {
				f.TicketStatus(id, actors["EXECUTING_AGENCY"], entity.TicketStatusEnum(state))
			}
			return id
		}))
	})

	t.Run("monitoring", func(t *testing.T) {
		svc := monitoring_service.NewService(store, nil, nil)
		comment := "real progress is lower than claimed"
		validate := func(real int, comment *string) func(testutil.Actor, uuid.UUID) error {
			return func(a testutil.Actor, id uuid.UUID) error {
				access, err := pol.CanValidateProgress(ctx, a.UserID, id)
				if err != nil {
					return err
				}
				_, err = svc.ValidateProgress(ctx, monitoring_service.ValidateProgressParams{
					ProgressLogID: id, AuditorID: access.OrganizationID, UserID: a.UserID,
					RealProgressPercentage: real, Comment: comment,
				})
				return err
			}
		}
		auditor := []string{"INDEPENDENT_AUDITOR"}
		actions := []matrixAction{
			{"ACCEPT", auditor, set("PENDING_VALIDATION"), validate(10, nil)},
			{"REJECT", auditor, set("PENDING_VALIDATION"), validate(5, &comment)},
		}
		states := []string{"PENDING_VALIDATION", "VALIDATED"}
		add(runMatrix(t, "monitoring", states, actors, actions, func(state string) uuid.UUID {
			id := f.ProgressLog(project.TaskID, actors["CONTRACTOR"].OrgID)
			if state == "VALIDATED" {
				insertValidation(t, db, id, actors["INDEPENDENT_AUDITOR"].OrgID)
			}
			return id
		}))
	})

	t.Logf("MATRIX total cells=%d ok=%d rejected_fsm_400=%d rejected_policy_403=%d",
		total.cells, total.ok, total.badRequest, total.forbidden)
}

func insertValidation(t *testing.T, db *sql.DB, logID, auditorOrgID uuid.UUID) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO progress_validations (is_accepted, real_progress_percentage, progress_log_id, auditor_id) VALUES (true, 10, $1, $2)`,
		logID, auditorOrgID); err != nil {
		t.Fatalf("insert validation: %v", err)
	}
}
