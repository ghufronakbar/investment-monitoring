//go:build integration

// Package integration exercises the governance core against a real
// PostgreSQL database. Run it through scripts/test-integration.sh.
package integration

import (
	"context"
	"net/http"
	"testing"

	"investment_monitoring/internal/policy"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/internal/testutil"

	"github.com/google/uuid"
)

// TestOrganizationCentricAuthorizationVersusRBAC runs each access request
// through the policy layer and through an RBAC baseline that only asks whether
// the user holds an active membership in an organization of the required type.
// Cases where the two disagree are what organization-centric authorization adds.
func TestOrganizationCentricAuthorizationVersusRBAC(t *testing.T) {
	db, store := testutil.OpenTestDB(t)
	f := testutil.NewFixture(t, db)
	ctx := context.Background()
	pol := policy.NewService(store)

	gs := f.Org(entity.OrganizationRoleEnumGOVSUPERVISOR)
	lp := f.Org(entity.OrganizationRoleEnumLOANPROVIDER)
	ea := f.Org(entity.OrganizationRoleEnumEXECUTINGAGENCY)
	ct := f.Org(entity.OrganizationRoleEnumCONTRACTOR)
	au := f.Org(entity.OrganizationRoleEnumINDEPENDENTAUDITOR)
	gpAssigned := f.Org(entity.OrganizationRoleEnumGOVPARTNER)
	gpUnassigned := f.Org(entity.OrganizationRoleEnumGOVPARTNER)
	p1 := f.Project(testutil.Participants{
		GovSupervisor: gs, LoanProvider: lp, ExecutingAgency: ea, Vendor: ct, Auditor: au,
		GovPartners: []uuid.UUID{gpAssigned, gpUnassigned},
	})

	// Project 2 is run by different organizations. Its auditor position is held
	// by an organization whose registered type is CONTRACTOR.
	ea2 := f.Org(entity.OrganizationRoleEnumEXECUTINGAGENCY)
	ct2 := f.Org(entity.OrganizationRoleEnumCONTRACTOR)
	ctAsAuditor := f.Org(entity.OrganizationRoleEnumCONTRACTOR)
	p2 := f.Project(testutil.Participants{
		GovSupervisor: gs, LoanProvider: lp, ExecutingAgency: ea2, Vendor: ct2, Auditor: ctAsAuditor,
	})

	// Organizations with the right type but no position in any contract.
	auOutside := f.Org(entity.OrganizationRoleEnumINDEPENDENTAUDITOR)
	lpOutside := f.Org(entity.OrganizationRoleEnumLOANPROVIDER)
	gpOutside := f.Org(entity.OrganizationRoleEnumGOVPARTNER)
	gsOutside := f.Org(entity.OrganizationRoleEnumGOVSUPERVISOR)

	admin := entity.MemberRoleEnumADMIN
	ctAdmin := f.Member(ct, admin)
	ct2Admin := f.Member(ct2, admin)
	ctFormer := f.FormerMember(ct)
	noOrg := testutil.Actor{UserID: f.User()}
	auAdmin := f.Member(au, admin)
	auOutsideAdmin := f.Member(auOutside, admin)
	ctAsAuditorAdmin := f.Member(ctAsAuditor, admin)
	lpAdmin := f.Member(lp, admin)
	lpOutsideAdmin := f.Member(lpOutside, admin)
	eaAdmin := f.Member(ea, admin)
	eaMember := f.Member(ea, entity.MemberRoleEnumMEMBER)
	ea2Admin := f.Member(ea2, admin)
	gpAssignedAdmin := f.Member(gpAssigned, admin)
	gpUnassignedAdmin := f.Member(gpUnassigned, admin)
	gpOutsideAdmin := f.Member(gpOutside, admin)
	gsOutsideAdmin := f.Member(gsOutside, admin)

	p1Log := f.ProgressLog(p1.TaskID, ct)
	p2Log := f.ProgressLog(p2.TaskID, ct2)
	p1Payment := f.PaymentRequest(p1.ContractID, entity.PaymentRequestStatusEnumSUBMITTED)
	p2Payment := f.PaymentRequest(p2.ContractID, entity.PaymentRequestStatusEnumSUBMITTED)
	p1Ticket := f.Ticket(p1.ContractID, ctAdmin, uuid.NullUUID{UUID: gpAssigned, Valid: true})

	// rbac is the baseline decision: an active membership in any organization
	// of the required type, regardless of project or contract.
	rbac := func(a testutil.Actor, roles ...entity.OrganizationRoleEnum) bool {
		for _, role := range roles {
			if _, ok, err := pol.UserHasOrgRole(ctx, a.UserID, role, false); err != nil {
				t.Fatalf("rbac baseline: %v", err)
			} else if ok {
				return true
			}
		}
		return false
	}
	anyMembership := func(a testutil.Actor) bool {
		memberships, err := store.GetUserMemberships(ctx, a.UserID)
		if err != nil {
			t.Fatalf("memberships: %v", err)
		}
		return len(memberships) > 0
	}
	errOnly := func(_ *policy.ProjectAccess, err error) error { return err }

	cases := []struct {
		id       string
		name     string
		decide   func() error
		baseline func() bool
		want     int
	}{
		{"AUTH-01", "contractor submits progress on its own project",
			func() error { return errOnly(pol.CanSubmitProgress(ctx, ctAdmin.UserID, p1.TaskID)) },
			func() bool { return rbac(ctAdmin, entity.OrganizationRoleEnumCONTRACTOR) }, http.StatusOK},
		{"AUTH-02", "contractor submits progress on another project (cross-project)",
			func() error { return errOnly(pol.CanSubmitProgress(ctx, ct2Admin.UserID, p1.TaskID)) },
			func() bool { return rbac(ct2Admin, entity.OrganizationRoleEnumCONTRACTOR) }, http.StatusForbidden},
		{"AUTH-03", "former member of the contractor submits progress",
			func() error { return errOnly(pol.CanSubmitProgress(ctx, ctFormer.UserID, p1.TaskID)) },
			func() bool { return rbac(ctFormer, entity.OrganizationRoleEnumCONTRACTOR) }, http.StatusForbidden},
		{"AUTH-04", "user without any organization submits progress",
			func() error { return errOnly(pol.CanSubmitProgress(ctx, noOrg.UserID, p1.TaskID)) },
			func() bool { return rbac(noOrg, entity.OrganizationRoleEnumCONTRACTOR) }, http.StatusForbidden},
		{"AUTH-05", "contracted auditor validates progress",
			func() error { return errOnly(pol.CanValidateProgress(ctx, auAdmin.UserID, p1Log)) },
			func() bool { return rbac(auAdmin, entity.OrganizationRoleEnumINDEPENDENTAUDITOR) }, http.StatusOK},
		{"AUTH-06", "auditor-type organization without a contract validates progress",
			func() error { return errOnly(pol.CanValidateProgress(ctx, auOutsideAdmin.UserID, p1Log)) },
			func() bool { return rbac(auOutsideAdmin, entity.OrganizationRoleEnumINDEPENDENTAUDITOR) }, http.StatusForbidden},
		{"AUTH-07", "contractor-type organization holding the auditor position validates progress",
			func() error { return errOnly(pol.CanValidateProgress(ctx, ctAsAuditorAdmin.UserID, p2Log)) },
			func() bool { return rbac(ctAsAuditorAdmin, entity.OrganizationRoleEnumINDEPENDENTAUDITOR) }, http.StatusOK},
		{"AUTH-08", "same organization submits progress on the project it audits",
			func() error { return errOnly(pol.CanSubmitProgress(ctx, ctAsAuditorAdmin.UserID, p2.TaskID)) },
			func() bool { return rbac(ctAsAuditorAdmin, entity.OrganizationRoleEnumCONTRACTOR) }, http.StatusForbidden},
		{"AUTH-09", "same organization validates progress on a project it does not audit",
			func() error { return errOnly(pol.CanValidateProgress(ctx, ctAsAuditorAdmin.UserID, p1Log)) },
			func() bool { return rbac(ctAsAuditorAdmin, entity.OrganizationRoleEnumINDEPENDENTAUDITOR) }, http.StatusForbidden},
		{"AUTH-10", "auditor-type organization without a contract approves a payment",
			func() error { return errOnly(pol.CanApprovePaymentAsAuditor(ctx, auOutsideAdmin.UserID, p1Payment)) },
			func() bool { return rbac(auOutsideAdmin, entity.OrganizationRoleEnumINDEPENDENTAUDITOR) }, http.StatusForbidden},
		{"AUTH-11", "contractor-type organization holding the auditor position approves a payment",
			func() error { return errOnly(pol.CanApprovePaymentAsAuditor(ctx, ctAsAuditorAdmin.UserID, p2Payment)) },
			func() bool { return rbac(ctAsAuditorAdmin, entity.OrganizationRoleEnumINDEPENDENTAUDITOR) }, http.StatusOK},
		{"AUTH-12", "contracted lender disburses a payment",
			func() error { return errOnly(pol.CanDisbursePayment(ctx, lpAdmin.UserID, p1Payment)) },
			func() bool { return rbac(lpAdmin, entity.OrganizationRoleEnumLOANPROVIDER) }, http.StatusOK},
		{"AUTH-13", "lender-type organization without a contract disburses a payment",
			func() error { return errOnly(pol.CanDisbursePayment(ctx, lpOutsideAdmin.UserID, p1Payment)) },
			func() bool { return rbac(lpOutsideAdmin, entity.OrganizationRoleEnumLOANPROVIDER) }, http.StatusForbidden},
		{"AUTH-14", "executing agency admin assigns a ticket",
			func() error { return errOnly(pol.CanAssignTicket(ctx, eaAdmin.UserID, p1Ticket)) },
			func() bool { return rbac(eaAdmin, entity.OrganizationRoleEnumEXECUTINGAGENCY) }, http.StatusOK},
		{"AUTH-15", "executing agency non-admin member assigns a ticket",
			func() error { return errOnly(pol.CanAssignTicket(ctx, eaMember.UserID, p1Ticket)) },
			func() bool { return rbac(eaMember, entity.OrganizationRoleEnumEXECUTINGAGENCY) }, http.StatusForbidden},
		{"AUTH-16", "executing agency of another project assigns a ticket",
			func() error { return errOnly(pol.CanAssignTicket(ctx, ea2Admin.UserID, p1Ticket)) },
			func() bool { return rbac(ea2Admin, entity.OrganizationRoleEnumEXECUTINGAGENCY) }, http.StatusForbidden},
		{"AUTH-17", "assigned government partner updates a ticket",
			func() error { return errOnly(pol.CanAddTicketLog(ctx, gpAssignedAdmin.UserID, p1Ticket)) },
			func() bool {
				return rbac(gpAssignedAdmin, entity.OrganizationRoleEnumGOVPARTNER, entity.OrganizationRoleEnumEXECUTINGAGENCY)
			}, http.StatusOK},
		{"AUTH-18", "government partner on the contract but not assigned updates a ticket",
			func() error { return errOnly(pol.CanAddTicketLog(ctx, gpUnassignedAdmin.UserID, p1Ticket)) },
			func() bool {
				return rbac(gpUnassignedAdmin, entity.OrganizationRoleEnumGOVPARTNER, entity.OrganizationRoleEnumEXECUTINGAGENCY)
			}, http.StatusForbidden},
		{"AUTH-19", "government partner outside the contract updates a ticket",
			func() error { return errOnly(pol.CanAddTicketLog(ctx, gpOutsideAdmin.UserID, p1Ticket)) },
			func() bool {
				return rbac(gpOutsideAdmin, entity.OrganizationRoleEnumGOVPARTNER, entity.OrganizationRoleEnumEXECUTINGAGENCY)
			}, http.StatusForbidden},
		{"AUTH-20", "contractor that raised the ticket updates its status",
			func() error { return errOnly(pol.CanAddTicketLog(ctx, ctAdmin.UserID, p1Ticket)) },
			func() bool {
				return rbac(ctAdmin, entity.OrganizationRoleEnumGOVPARTNER, entity.OrganizationRoleEnumEXECUTINGAGENCY)
			}, http.StatusForbidden},
		{"AUTH-21", "contractor of another project views the monitoring data",
			func() error { return pol.CanViewMonitoringProject(ctx, ct2Admin.UserID, p1.ID) },
			func() bool { return anyMembership(ct2Admin) }, http.StatusForbidden},
		{"AUTH-22", "government supervisor outside the contract views the project (supervisory fallback)",
			func() error { return pol.CanViewProject(ctx, gsOutsideAdmin.UserID, p1.ID) },
			func() bool { return rbac(gsOutsideAdmin, entity.OrganizationRoleEnumGOVSUPERVISOR) }, http.StatusOK},
	}

	disagreements := 0
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			got := testutil.StatusOf(tc.decide())
			if got != tc.want {
				t.Fatalf("%s: status = %d, want %d", tc.name, got, tc.want)
			}
			baselineAllows := tc.baseline()
			policyAllows := got == http.StatusOK
			if baselineAllows != policyAllows {
				disagreements++
			}
			t.Logf("%s | %-85s | policy=%d | rbac_baseline_allows=%v | differs=%v",
				tc.id, tc.name, got, baselineAllows, baselineAllows != policyAllows)
		})
	}
	t.Logf("AUTH summary: cases=%d differ_from_rbac_baseline=%d", len(cases), disagreements)
}
