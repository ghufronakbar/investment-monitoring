// internal/module/support/uploader/service/params.go
package cloudinary_uploader

import (
	"io"

	"github.com/google/uuid"
)

// Single upload params come from multipart form-data, not JSON
// Multi upload params also come from multipart form-data

type UploadFileParams struct {
	File       io.Reader
	FileName   string
	Size       int64
	UserID     uuid.UUID
	OrgID      *uuid.UUID
	EntityID   *uuid.UUID
	EntityType *string
}

// Allowed MIME types
var AllowedMimeTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"application/pdf": true,
}

// Max file size: 5MB
const MaxFileSize int64 = 5 * 1024 * 1024
