// internal/module/support/uploader/service/return.go
package cloudinary_uploader

// UploadResponse for single file upload
type UploadResponse struct {
	PublicId string `json:"publicId"`
	Url      string `json:"url"`
	Format   string `json:"format"`
}
