package workflow

import (
	"testing"

	"investment_monitoring/internal/repository/entity"
)

func TestTrackerNextStatus(t *testing.T) {
	tests := []struct {
		name    string
		current entity.TicketStatusEnum
		input   Input
		want    entity.TicketStatusEnum
		ok      bool
	}{
		{
			name:    "assign open ticket",
			current: entity.TicketStatusEnumOPEN,
			input:   TrackerInputAssign,
			want:    entity.TicketStatusEnumASSIGNED,
			ok:      true,
		},
		{
			name:    "move assigned to in progress",
			current: entity.TicketStatusEnumASSIGNED,
			input:   TrackerInputSetProgress,
			want:    entity.TicketStatusEnumINPROGRESS,
			ok:      true,
		},
		{
			name:    "resolve in progress ticket",
			current: entity.TicketStatusEnumINPROGRESS,
			input:   TrackerInputSetResolved,
			want:    entity.TicketStatusEnumRESOLVED,
			ok:      true,
		},
		{
			name:    "close resolved ticket",
			current: entity.TicketStatusEnumRESOLVED,
			input:   TrackerInputClose,
			want:    entity.TicketStatusEnumCLOSED,
			ok:      true,
		},
		{
			name:    "closed is terminal",
			current: entity.TicketStatusEnumCLOSED,
			input:   TrackerInputComment,
			ok:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := TrackerNextStatus(tt.current, tt.input)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("status = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestTrackerInputForStatus(t *testing.T) {
	input, ok := TrackerInputForStatus(entity.TicketStatusEnumINPROGRESS)
	if !ok {
		t.Fatal("expected status to map to input")
	}
	if input != TrackerInputSetProgress {
		t.Fatalf("input = %s, want %s", input, TrackerInputSetProgress)
	}

	if _, ok := TrackerInputForStatus(entity.TicketStatusEnum("UNKNOWN")); ok {
		t.Fatal("expected unknown status to be invalid")
	}
}
