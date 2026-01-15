// internal/module/support/uploader/service/service.go
package cloudinary_uploader

import (
	"context"
	"errors"
	"investment_monitoring/config"
	"investment_monitoring/pkg/errs"
	"io"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type UploaderService struct {
	cfg *config.Config
	cld *cloudinary.Cloudinary
}

func NewService(cfg *config.Config) (*UploaderService, error) {
	cld, err := cloudinary.NewFromParams(cfg.CLOUDINARY_CLOUD_NAME, cfg.CLOUDINARY_API_KEY, cfg.CLOUDINARY_API_SECRET)
	if err != nil {
		return nil, err
	}
	return &UploaderService{
		cfg: cfg,
		cld: cld,
	}, nil
}

// UploadSingle uploads a single file to Cloudinary
func (s *UploaderService) UploadSingle(ctx context.Context, file io.Reader, resourceType string) (*UploadResponse, error) {
	params := uploader.UploadParams{
		Folder: s.cfg.APP_NAME,
		Tags:   []string{"source:api"},
	}

	// Set resource type for PDF
	if resourceType == "pdf" {
		params.ResourceType = "raw"
		params.Tags = append(params.Tags, "type:document")
	} else {
		params.Tags = append(params.Tags, "type:image")
	}

	result, err := s.cld.Upload.Upload(ctx, file, params)
	if err != nil {
		return nil, errs.NewInternalServerError(err)
	}
	if result == nil || result.SecureURL == "" {
		return nil, errs.NewInternalServerError(errors.New("CLOUDINARY_RETURNED_EMPTY_URL"))
	}

	return &UploadResponse{
		PublicId: result.PublicID,
		Url:      result.SecureURL,
		Format:   result.Format,
	}, nil
}

// UploadMultiple uploads multiple files to Cloudinary
func (s *UploaderService) UploadMultiple(ctx context.Context, files []io.Reader, resourceTypes []string) ([]UploadResponse, error) {
	var uploaded []UploadResponse

	for i, file := range files {
		resourceType := "image"
		if i < len(resourceTypes) {
			resourceType = resourceTypes[i]
		}

		result, err := s.UploadSingle(ctx, file, resourceType)
		if err != nil {
			return nil, err
		}
		uploaded = append(uploaded, *result)
	}

	return uploaded, nil
}
