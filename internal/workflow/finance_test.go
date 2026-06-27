package workflow

import (
	"testing"

	"investment_monitoring/internal/repository/entity"
)

func TestFinanceNextStatus(t *testing.T) {
	tests := []struct {
		name    string
		current entity.PaymentRequestStatusEnum
		input   Input
		want    entity.PaymentRequestStatusEnum
		ok      bool
	}{
		{
			name:    "auditor approve submitted",
			current: entity.PaymentRequestStatusEnumSUBMITTED,
			input:   FinanceInputAuditorApprove,
			want:    entity.PaymentRequestStatusEnumAUDITORAPPROVED,
			ok:      true,
		},
		{
			name:    "agency approve auditor approved",
			current: entity.PaymentRequestStatusEnumAUDITORAPPROVED,
			input:   FinanceInputAgencyApprove,
			want:    entity.PaymentRequestStatusEnumAGENCYAPPROVED,
			ok:      true,
		},
		{
			name:    "disburse agency approved",
			current: entity.PaymentRequestStatusEnumAGENCYAPPROVED,
			input:   FinanceInputDisburse,
			want:    entity.PaymentRequestStatusEnumPAID,
			ok:      true,
		},
		{
			name:    "cannot skip auditor",
			current: entity.PaymentRequestStatusEnumSUBMITTED,
			input:   FinanceInputAgencyApprove,
			ok:      false,
		},
		{
			name:    "paid is terminal",
			current: entity.PaymentRequestStatusEnumPAID,
			input:   FinanceInputDisburse,
			ok:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := FinanceNextStatus(tt.current, tt.input)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("status = %s, want %s", got, tt.want)
			}
		})
	}
}
