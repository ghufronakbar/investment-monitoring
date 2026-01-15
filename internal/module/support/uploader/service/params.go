// internal/module/support/uploader/service/params.go
package cloudinary_uploader

// Single upload params come from multipart form-data, not JSON
// Multi upload params also come from multipart form-data

// Allowed MIME types
var AllowedMimeTypes = map[string]bool{
	"image/jpeg":      true,
	"image/jpg":       true,
	"image/png":       true,
	"application/pdf": true,
}

// Max file size: 5MB
const MaxFileSize int64 = 5 * 1024 * 1024
