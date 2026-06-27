// internal/module/support/uploader/service/return.go
package cloudinary_uploader

import (
	"time"

	"github.com/google/uuid"
)

// UploadResponse for single file upload
type UploadResponse struct {
	ID               uuid.UUID  `json:"id"`
	PublicId         string     `json:"publicId"`
	Url              string     `json:"url"`
	Format           string     `json:"format"`
	FileName         string     `json:"fileName"`
	MimeType         string     `json:"mimeType"`
	Size             int64      `json:"size"`
	Checksum         string     `json:"checksum"`
	UploadedByUserID uuid.UUID  `json:"uploadedByUserId"`
	UploadedByOrgID  *uuid.UUID `json:"uploadedByOrgId,omitempty"`
	EntityType       *string    `json:"entityType,omitempty"`
	EntityID         *uuid.UUID `json:"entityId,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
}
