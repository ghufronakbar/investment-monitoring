package audit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestToCreateParamsMapsActorEntityMetadataAndData(t *testing.T) {
	svc := NewService(nil)
	actorID := uuid.New()
	orgID := uuid.New()
	entityID := uuid.New()

	ctx := context.WithValue(context.Background(), requestMetadataContextKey{}, RequestMetadata{
		RequestID: "req-1",
		IPAddress: "127.0.0.1",
		UserAgent: "go-test",
	})

	got := svc.toCreateParams(ctx, RecordParams{
		ActorUserID: actorID,
		ActorOrgID:  orgID,
		Action:      ActionCreateProject,
		EntityType:  EntityProject,
		EntityID:    entityID,
		BeforeData:  map[string]string{"status": "draft"},
		AfterData:   map[string]string{"status": "active"},
	})

	if !got.ActorUserID.Valid || got.ActorUserID.UUID != actorID {
		t.Fatalf("actor user id = %+v, want %s", got.ActorUserID, actorID)
	}
	if !got.ActorOrgID.Valid || got.ActorOrgID.UUID != orgID {
		t.Fatalf("actor org id = %+v, want %s", got.ActorOrgID, orgID)
	}
	if got.Action != ActionCreateProject {
		t.Fatalf("action = %s, want %s", got.Action, ActionCreateProject)
	}
	if got.EntityType != EntityProject {
		t.Fatalf("entity type = %s, want %s", got.EntityType, EntityProject)
	}
	if !got.EntityID.Valid || got.EntityID.UUID != entityID {
		t.Fatalf("entity id = %+v, want %s", got.EntityID, entityID)
	}
	if got.RequestID.String != "req-1" || !got.RequestID.Valid {
		t.Fatalf("request id = %+v, want req-1", got.RequestID)
	}
	if got.IpAddress.String != "127.0.0.1" || !got.IpAddress.Valid {
		t.Fatalf("ip address = %+v, want 127.0.0.1", got.IpAddress)
	}
	if got.UserAgent.String != "go-test" || !got.UserAgent.Valid {
		t.Fatalf("user agent = %+v, want go-test", got.UserAgent)
	}

	assertJSONField(t, got.BeforeData, "status", "draft")
	assertJSONField(t, got.AfterData, "status", "active")
}

func TestToCreateParamsUsesEmptyObjectsAndInvalidNullsForNilValues(t *testing.T) {
	svc := NewService(nil)

	got := svc.toCreateParams(context.Background(), RecordParams{
		Action:     ActionCreateTicket,
		EntityType: EntityTicket,
	})

	if got.ActorUserID.Valid {
		t.Fatalf("actor user id valid = true, want false")
	}
	if got.ActorOrgID.Valid {
		t.Fatalf("actor org id valid = true, want false")
	}
	if got.EntityID.Valid {
		t.Fatalf("entity id valid = true, want false")
	}
	if got.RequestID.Valid || got.IpAddress.Valid || got.UserAgent.Valid {
		t.Fatalf("metadata should be invalid nulls, got request=%+v ip=%+v ua=%+v", got.RequestID, got.IpAddress, got.UserAgent)
	}
	if string(got.BeforeData) != "{}" {
		t.Fatalf("before data = %s, want {}", got.BeforeData)
	}
	if string(got.AfterData) != "{}" {
		t.Fatalf("after data = %s, want {}", got.AfterData)
	}
}

func TestMarshalDataFallsBackWhenJSONMarshalFails(t *testing.T) {
	got := marshalData(func() {})
	if string(got) != `{"marshal_error":true}` {
		t.Fatalf("marshal data = %s, want marshal error marker", got)
	}
}

func assertJSONField(t *testing.T, raw []byte, key string, want string) {
	t.Helper()

	var got map[string]string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("json unmarshal %s: %v", raw, err)
	}
	if got[key] != want {
		t.Fatalf("json field %s = %s, want %s", key, got[key], want)
	}
}
