package cloudinary_uploader

import (
	"database/sql"
	"testing"
	"time"

	"investment_monitoring/internal/repository/entity"

	"github.com/google/uuid"
)

func TestValidateEntityBindingRequiresTypeAndIDTogether(t *testing.T) {
	entityType := "ticket"
	entityID := uuid.New()

	tests := []struct {
		name       string
		entityType *string
		entityID   *uuid.UUID
		wantErr    bool
	}{
		{name: "none", wantErr: false},
		{name: "both", entityType: &entityType, entityID: &entityID, wantErr: false},
		{name: "type only", entityType: &entityType, wantErr: true},
		{name: "id only", entityID: &entityID, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEntityBinding(tt.entityType, tt.entityID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "base name only", in: "../../invoice.pdf", want: "invoice.pdf"},
		{name: "trim spaces", in: "  proof.png  ", want: "proof.png"},
		{name: "empty", in: "   ", want: "upload"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeFileName(tt.in); got != tt.want {
				t.Fatalf("sanitizeFileName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestUploaderNullableHelpers(t *testing.T) {
	entityType := " ticket "
	gotString := nullableString(&entityType)
	if !gotString.Valid || gotString.String != "TICKET" {
		t.Fatalf("nullableString = %+v, want uppercase valid TICKET", gotString)
	}

	blank := "  "
	if nullableString(&blank).Valid {
		t.Fatal("expected blank string to be invalid null")
	}

	id := uuid.New()
	gotUUID := nullableUUID(&id)
	if !gotUUID.Valid || gotUUID.UUID != id {
		t.Fatalf("nullableUUID = %+v, want %s", gotUUID, id)
	}
}

func TestMapUploadResponseMapsOptionalFields(t *testing.T) {
	orgID := uuid.New()
	entityID := uuid.New()
	createdAt := time.Now()
	file := entity.File{
		ID:               uuid.New(),
		PublicID:         "public-id",
		Url:              "https://example.com/file.png",
		FileName:         "file.png",
		MimeType:         "image/png",
		Size:             123,
		Checksum:         "checksum",
		UploadedByUserID: uuid.New(),
		UploadedByOrgID:  uuid.NullUUID{UUID: orgID, Valid: true},
		EntityType:       sql.NullString{String: "TICKET", Valid: true},
		EntityID:         uuid.NullUUID{UUID: entityID, Valid: true},
		CreatedAt:        createdAt,
	}

	got := mapUploadResponse(file, "png")
	if got.ID != file.ID || got.PublicId != file.PublicID || got.Url != file.Url {
		t.Fatalf("response identity fields = %+v, want file values", got)
	}
	if got.UploadedByOrgID == nil || *got.UploadedByOrgID != orgID {
		t.Fatalf("uploaded org id = %v, want %s", got.UploadedByOrgID, orgID)
	}
	if got.EntityType == nil || *got.EntityType != "TICKET" {
		t.Fatalf("entity type = %v, want TICKET", got.EntityType)
	}
	if got.EntityID == nil || *got.EntityID != entityID {
		t.Fatalf("entity id = %v, want %s", got.EntityID, entityID)
	}
	if !got.CreatedAt.Equal(createdAt) {
		t.Fatalf("created at = %v, want %v", got.CreatedAt, createdAt)
	}
}
