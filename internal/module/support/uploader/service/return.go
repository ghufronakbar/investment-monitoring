// internal/module/headless/cloudinary_uploader/viewmodel.go
package cloudinary_uploader

type CloudinaryUploadSingleImageResponse struct {
	PublicId string `json:"publicId"`
	ImageUrl string `json:"imageUrl"`
	Format   string `json:"format"`
}
