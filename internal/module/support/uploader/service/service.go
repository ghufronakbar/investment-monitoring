// internal/module/support/uploader/service/service.go
package cloudinary_uploader

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"investment_monitoring/config"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/google/uuid"
)

type UploaderService struct {
	store entity.Store
	cfg   *config.Config
	cld   *cloudinary.Cloudinary
}

func NewService(store entity.Store, cfg *config.Config) (*UploaderService, error) {
	cld, err := cloudinary.NewFromParams(cfg.CLOUDINARY_CLOUD_NAME, cfg.CLOUDINARY_API_KEY, cfg.CLOUDINARY_API_SECRET)
	if err != nil {
		return nil, err
	}
	return &UploaderService{
		store: store,
		cfg:   cfg,
		cld:   cld,
	}, nil
}

// UploadSingle uploads a single file to Cloudinary
func (s *UploaderService) UploadSingle(ctx context.Context, params UploadFileParams) (*UploadResponse, error) {
	metadata, err := s.prepareUpload(ctx, params)
	if err != nil {
		return nil, err
	}

	uploadParams := uploader.UploadParams{
		Folder: s.cfg.APP_NAME,
		Tags:   []string{"source:api"},
	}

	// Set resource type for PDF
	if metadata.resourceType == "pdf" {
		uploadParams.ResourceType = "raw"
		uploadParams.Tags = append(uploadParams.Tags, "type:document")
	} else {
		uploadParams.Tags = append(uploadParams.Tags, "type:image")
	}

	result, err := s.cld.Upload.Upload(ctx, bytes.NewReader(metadata.data), uploadParams)
	if err != nil {
		return nil, errs.NewInternalServerError(err)
	}
	if result == nil || result.SecureURL == "" {
		return nil, errs.NewInternalServerError(errors.New("CLOUDINARY_RETURNED_EMPTY_URL"))
	}

	uploaded, err := s.store.CreateFileMetadata(ctx, entity.CreateFileMetadataParams{
		PublicID:         result.PublicID,
		Url:              result.SecureURL,
		FileName:         metadata.fileName,
		MimeType:         metadata.mimeType,
		Size:             int64(len(metadata.data)),
		Checksum:         metadata.checksum,
		UploadedByUserID: metadata.userID,
		UploadedByOrgID:  metadata.orgID,
		EntityType:       metadata.entityType,
		EntityID:         metadata.entityID,
	})
	if err != nil {
		return nil, errs.NewInternalServerError(err)
	}

	return mapUploadResponse(uploaded, result.Format), nil
}

// UploadMultiple uploads multiple files to Cloudinary
func (s *UploaderService) UploadMultiple(ctx context.Context, files []UploadFileParams) ([]UploadResponse, error) {
	var uploaded []UploadResponse

	for _, file := range files {
		result, err := s.UploadSingle(ctx, file)
		if err != nil {
			return nil, err
		}
		uploaded = append(uploaded, *result)
	}

	return uploaded, nil
}

type preparedUpload struct {
	data         []byte
	fileName     string
	mimeType     string
	checksum     string
	resourceType string
	userID       uuid.UUID
	orgID        uuid.NullUUID
	entityType   sql.NullString
	entityID     uuid.NullUUID
}

func (s *UploaderService) prepareUpload(ctx context.Context, params UploadFileParams) (preparedUpload, error) {
	if params.File == nil {
		return preparedUpload{}, errs.NewBadRequest("FILE_REQUIRED")
	}
	if params.Size > MaxFileSize {
		return preparedUpload{}, errs.NewBadRequest("FILE_TOO_LARGE_5MB_MAX")
	}
	if err := validateEntityBinding(params.EntityType, params.EntityID); err != nil {
		return preparedUpload{}, err
	}

	orgID := uuid.NullUUID{}
	if params.OrgID != nil {
		isMember, err := s.store.CheckMembershipExists(ctx, entity.CheckMembershipExistsParams{
			UserID:         params.UserID,
			OrganizationID: *params.OrgID,
		})
		if err != nil {
			return preparedUpload{}, errs.NewInternalServerError(err)
		}
		if !isMember {
			return preparedUpload{}, errs.NewForbidden("USER_NOT_MEMBER_OF_ORGANIZATION")
		}
		orgID = uuid.NullUUID{UUID: *params.OrgID, Valid: true}
	}

	data, err := io.ReadAll(io.LimitReader(params.File, MaxFileSize+1))
	if err != nil {
		return preparedUpload{}, errs.NewInternalServerError(err)
	}
	if len(data) == 0 {
		return preparedUpload{}, errs.NewBadRequest("FILE_EMPTY")
	}
	if int64(len(data)) > MaxFileSize {
		return preparedUpload{}, errs.NewBadRequest("FILE_TOO_LARGE_5MB_MAX")
	}

	mimeType := detectMimeType(data)
	if !AllowedMimeTypes[mimeType] {
		return preparedUpload{}, errs.NewBadRequest("INVALID_FILE_TYPE")
	}

	hash := sha256.Sum256(data)
	resourceType := "image"
	if mimeType == "application/pdf" {
		resourceType = "pdf"
	}

	return preparedUpload{
		data:         data,
		fileName:     sanitizeFileName(params.FileName),
		mimeType:     mimeType,
		checksum:     hex.EncodeToString(hash[:]),
		resourceType: resourceType,
		userID:       params.UserID,
		orgID:        orgID,
		entityType:   nullableString(params.EntityType),
		entityID:     nullableUUID(params.EntityID),
	}, nil
}

func detectMimeType(data []byte) string {
	sniffLength := 512
	if len(data) < sniffLength {
		sniffLength = len(data)
	}
	return http.DetectContentType(data[:sniffLength])
}

func sanitizeFileName(fileName string) string {
	fileName = strings.TrimSpace(filepath.Base(fileName))
	if fileName == "." || fileName == string(filepath.Separator) || fileName == "" {
		return "upload"
	}
	return fileName
}

func validateEntityBinding(entityType *string, entityID *uuid.UUID) error {
	hasEntityType := entityType != nil && strings.TrimSpace(*entityType) != ""
	hasEntityID := entityID != nil
	if hasEntityType != hasEntityID {
		return errs.NewBadRequest("ENTITY_TYPE_AND_ENTITY_ID_REQUIRED_TOGETHER")
	}
	return nil
}

func nullableString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: strings.ToUpper(trimmed), Valid: true}
}

func nullableUUID(value *uuid.UUID) uuid.NullUUID {
	if value == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *value, Valid: true}
}

func mapUploadResponse(file entity.File, format string) *UploadResponse {
	var orgID *uuid.UUID
	if file.UploadedByOrgID.Valid {
		orgID = &file.UploadedByOrgID.UUID
	}

	var entityType *string
	if file.EntityType.Valid {
		entityType = &file.EntityType.String
	}

	var entityID *uuid.UUID
	if file.EntityID.Valid {
		entityID = &file.EntityID.UUID
	}

	return &UploadResponse{
		ID:               file.ID,
		PublicId:         file.PublicID,
		Url:              file.Url,
		Format:           format,
		FileName:         file.FileName,
		MimeType:         file.MimeType,
		Size:             file.Size,
		Checksum:         file.Checksum,
		UploadedByUserID: file.UploadedByUserID,
		UploadedByOrgID:  orgID,
		EntityType:       entityType,
		EntityID:         entityID,
		CreatedAt:        file.CreatedAt,
	}
}
