package workflow

import "testing"

func TestMonitoringValidationInput(t *testing.T) {
	tests := []struct {
		name    string
		claimed int
		real    int
		want    Input
		wantErr bool
	}{
		{
			name:    "accept equal progress",
			claimed: 40,
			real:    40,
			want:    MonitoringInputAccept,
		},
		{
			name:    "reject lower progress",
			claimed: 40,
			real:    30,
			want:    MonitoringInputReject,
		},
		{
			name:    "real progress cannot exceed claimed",
			claimed: 40,
			real:    50,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MonitoringValidationInput(tt.claimed, tt.real)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("input = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestMonitoringNextValidationState(t *testing.T) {
	got, ok := MonitoringNextValidationState(MonitoringStatePending, MonitoringInputAccept)
	if !ok {
		t.Fatal("expected transition to be valid")
	}
	if got != MonitoringStateAccepted {
		t.Fatalf("state = %s, want %s", got, MonitoringStateAccepted)
	}

	if _, ok := MonitoringNextValidationState(MonitoringStateAccepted, MonitoringInputReject); ok {
		t.Fatal("expected accepted state to be terminal")
	}
}
