// internal/module/headless/cloudinary_uploader/service.go
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

type CloudinaryUploaderService struct {
	cfg *config.Config
	cld *cloudinary.Cloudinary
}

func NewService(cfg *config.Config) (*CloudinaryUploaderService, error) {
	cld, err := cloudinary.NewFromParams(cfg.CLOUDINARY_CLOUD_NAME, cfg.CLOUDINARY_API_KEY, cfg.CLOUDINARY_API_SECRET)
	if err != nil {
		return nil, err
	}
	return &CloudinaryUploaderService{
		cfg: cfg,
		cld: cld,
	}, nil
}

func (s *CloudinaryUploaderService) UploadSingleImage(ctx context.Context, file io.Reader) (*CloudinaryUploadSingleImageResponse, error) {
	result, err := s.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder: s.cfg.APP_NAME,
		Tags:   []string{"source:api", "type:image"},
	})
	if err != nil {
		return nil, errs.NewInternalServerError(err)
	}
	if result == nil {
		return nil, errs.NewInternalServerError(errors.New("CLOUDINARY_RETURNED_EMPTY_URL"))
	}
	return &CloudinaryUploadSingleImageResponse{
		PublicId: result.PublicID,
		ImageUrl: result.SecureURL,
		Format:   result.Format,
	}, nil
}
