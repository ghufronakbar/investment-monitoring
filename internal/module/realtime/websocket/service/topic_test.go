package websocket_service

import (
	"context"
	"testing"

	"investment_monitoring/config"

	"github.com/google/uuid"
)

func TestParseTopic(t *testing.T) {
	projectID := uuid.New()

	tests := []struct {
		name         string
		raw          string
		wantOK       bool
		wantKind     string
		wantID       uuid.UUID
		wantSubtopic string
	}{
		{
			name:     "user topic",
			raw:      "user." + projectID.String(),
			wantOK:   true,
			wantKind: TopicKindUser,
			wantID:   projectID,
		},
		{
			name:         "project monitoring topic",
			raw:          "project." + projectID.String() + ".monitoring",
			wantOK:       true,
			wantKind:     TopicKindProject,
			wantID:       projectID,
			wantSubtopic: TopicProjectMonitoring,
		},
		{
			name:     "ticket topic",
			raw:      "ticket." + projectID.String(),
			wantOK:   true,
			wantKind: TopicKindTicket,
			wantID:   projectID,
		},
		{
			name:   "invalid uuid",
			raw:    "project.not-a-uuid",
			wantOK: false,
		},
		{
			name:   "invalid project subtopic",
			raw:    "project." + projectID.String() + ".unknown",
			wantOK: false,
		},
		{
			name:   "too many segments",
			raw:    "ticket." + projectID.String() + ".extra",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseTopic(tt.raw)
			if ok != tt.wantOK {
				t.Fatalf("ParseTopic() ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if got.Kind != tt.wantKind {
				t.Fatalf("kind = %s, want %s", got.Kind, tt.wantKind)
			}
			if got.ID != tt.wantID {
				t.Fatalf("id = %s, want %s", got.ID, tt.wantID)
			}
			if got.Subtopic != tt.wantSubtopic {
				t.Fatalf("subtopic = %s, want %s", got.Subtopic, tt.wantSubtopic)
			}
		})
	}
}

func TestAuthorizeUserTopic(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	svc := NewService(&config.Config{}, nil, nil)

	if err := svc.AuthorizeTopic(context.Background(), userID, "user."+userID.String()); err != nil {
		t.Fatalf("AuthorizeTopic own user topic returned error: %v", err)
	}

	if err := svc.AuthorizeTopic(context.Background(), userID, "user."+otherUserID.String()); err == nil {
		t.Fatal("AuthorizeTopic other user topic returned nil error")
	}
}
