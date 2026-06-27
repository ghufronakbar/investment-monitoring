package workflow

import (
	"testing"

	"investment_monitoring/internal/repository/entity"
)

func TestFinanceNextStatusRejectsRejectedAndRoleMismatchPaths(t *testing.T) {
	tests := []struct {
		name    string
		current entity.PaymentRequestStatusEnum
		input   Input
	}{
		{
			name:    "auditor rejected terminal",
			current: entity.PaymentRequestStatusEnumAUDITORREJECTED,
			input:   FinanceInputAgencyApprove,
		},
		{
			name:    "agency rejected terminal",
			current: entity.PaymentRequestStatusEnumAGENCYREJECTED,
			input:   FinanceInputDisburse,
		},
		{
			name:    "agency cannot approve submitted",
			current: entity.PaymentRequestStatusEnumSUBMITTED,
			input:   FinanceInputAgencyApprove,
		},
		{
			name:    "auditor cannot approve agency approved",
			current: entity.PaymentRequestStatusEnumAGENCYAPPROVED,
			input:   FinanceInputAuditorApprove,
		},
		{
			name:    "invalid state",
			current: entity.PaymentRequestStatusEnum("UNKNOWN"),
			input:   FinanceInputAuditorApprove,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, ok := FinanceNextStatus(tt.current, tt.input); ok {
				t.Fatalf("status = %s, ok = true; want invalid transition", got)
			}
		})
	}
}

func TestFinanceApprovalInput(t *testing.T) {
	tests := []struct {
		name         string
		action       string
		auditorStage bool
		want         Input
	}{
		{name: "auditor approve", action: "APPROVE", auditorStage: true, want: FinanceInputAuditorApprove},
		{name: "auditor reject", action: "REJECT", auditorStage: true, want: FinanceInputAuditorReject},
		{name: "agency approve", action: "APPROVE", auditorStage: false, want: FinanceInputAgencyApprove},
		{name: "agency reject", action: "REJECT", auditorStage: false, want: FinanceInputAgencyReject},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FinanceApprovalInput(tt.action, tt.auditorStage); got != tt.want {
				t.Fatalf("input = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestMonitoringValidationState(t *testing.T) {
	tests := []struct {
		name               string
		isAlreadyValidated bool
		isAccepted         bool
		want               State
	}{
		{name: "pending", want: MonitoringStatePending},
		{name: "accepted", isAlreadyValidated: true, isAccepted: true, want: MonitoringStateAccepted},
		{name: "rejected", isAlreadyValidated: true, isAccepted: false, want: MonitoringStateRejected},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MonitoringValidationState(tt.isAlreadyValidated, tt.isAccepted); got != tt.want {
				t.Fatalf("state = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestTrackerNextStatusRejectsClosedReopenAndUnknownState(t *testing.T) {
	if got, ok := TrackerNextStatus(entity.TicketStatusEnumCLOSED, TrackerInputSetOpen); ok {
		t.Fatalf("status = %s, ok = true; want closed terminal", got)
	}
	if got, ok := TrackerNextStatus(entity.TicketStatusEnum("UNKNOWN"), TrackerInputComment); ok {
		t.Fatalf("status = %s, ok = true; want unknown state invalid", got)
	}
}
