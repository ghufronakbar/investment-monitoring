// internal/module/support/uploader/controller/handler.go
package uploader_controller

import (
	"net/http"
	"strings"

	"investment_monitoring/internal/http/internal_middleware"
	cloudinary_uploader "investment_monitoring/internal/module/support/uploader/service"
	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	uploaderSvc *cloudinary_uploader.UploaderService
}

func NewHandler(uploaderSvc *cloudinary_uploader.UploaderService) *Handler {
	return &Handler{
		uploaderSvc: uploaderSvc,
	}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/single", h.UploadSingle)
	r.Post("/multiple", h.UploadMultiple)

	return r
}

// UploadSingle handles POST /support/upload/single
// Accepts multipart form-data with field "file"
func (h *Handler) UploadSingle(w http.ResponseWriter, r *http.Request) {
	userClaims, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	// Parse multipart form with max size
	if err := r.ParseMultipartForm(cloudinary_uploader.MaxFileSize); err != nil {
		response.Error(w, r, errs.NewBadRequest("FILE_TOO_LARGE"), nil)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, r, errs.NewBadRequest("FILE_REQUIRED"), nil)
		return
	}
	defer file.Close()

	if header.Size > cloudinary_uploader.MaxFileSize {
		response.Error(w, r, errs.NewBadRequest("FILE_TOO_LARGE_5MB_MAX"), nil)
		return
	}

	uploadParams, err := h.parseUploadMetadata(r, userClaims.ID)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}
	uploadParams.File = file
	uploadParams.FileName = header.Filename
	uploadParams.Size = header.Size

	result, err := h.uploaderSvc.UploadSingle(r.Context(), uploadParams)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_UPLOAD", result)
}

// UploadMultiple handles POST /support/upload/multiple
// Accepts multipart form-data with field "files" (multiple)
func (h *Handler) UploadMultiple(w http.ResponseWriter, r *http.Request) {
	userClaims, err := internal_middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("AUTHENTICATION_REQUIRED"), nil)
		return
	}

	// Parse multipart form with max size (10 files * 5MB = 50MB max total)
	if err := r.ParseMultipartForm(cloudinary_uploader.MaxFileSize * 10); err != nil {
		response.Error(w, r, errs.NewBadRequest("TOTAL_SIZE_TOO_LARGE"), nil)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		response.Error(w, r, errs.NewBadRequest("FILES_REQUIRED"), nil)
		return
	}

	if len(files) > 10 {
		response.Error(w, r, errs.NewBadRequest("MAX_10_FILES_ALLOWED"), nil)
		return
	}

	baseParams, err := h.parseUploadMetadata(r, userClaims.ID)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	var uploadFiles []cloudinary_uploader.UploadFileParams

	for _, fileHeader := range files {
		if fileHeader.Size > cloudinary_uploader.MaxFileSize {
			response.Error(w, r, errs.NewBadRequest("FILE_TOO_LARGE_5MB_MAX"), nil)
			return
		}

		file, err := fileHeader.Open()
		if err != nil {
			response.Error(w, r, errs.NewBadRequest("FAILED_TO_OPEN_FILE"), nil)
			return
		}
		defer file.Close()

		params := baseParams
		params.File = file
		params.FileName = fileHeader.Filename
		params.Size = fileHeader.Size
		uploadFiles = append(uploadFiles, params)
	}

	result, err := h.uploaderSvc.UploadMultiple(r.Context(), uploadFiles)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_UPLOAD_MULTIPLE", result)
}

func (h *Handler) parseUploadMetadata(r *http.Request, userID uuid.UUID) (cloudinary_uploader.UploadFileParams, error) {
	var orgID *uuid.UUID
	if rawOrgID := strings.TrimSpace(r.FormValue("organizationId")); rawOrgID != "" {
		parsed, err := uuid.Parse(rawOrgID)
		if err != nil {
			return cloudinary_uploader.UploadFileParams{}, errs.NewValidationFailed(map[string]string{"organizationId": "Invalid UUID"})
		}
		orgID = &parsed
	}

	var entityType *string
	if rawEntityType := strings.TrimSpace(r.FormValue("entityType")); rawEntityType != "" {
		entityType = &rawEntityType
	}

	var entityID *uuid.UUID
	if rawEntityID := strings.TrimSpace(r.FormValue("entityId")); rawEntityID != "" {
		parsed, err := uuid.Parse(rawEntityID)
		if err != nil {
			return cloudinary_uploader.UploadFileParams{}, errs.NewValidationFailed(map[string]string{"entityId": "Invalid UUID"})
		}
		entityID = &parsed
	}

	return cloudinary_uploader.UploadFileParams{
		UserID:     userID,
		OrgID:      orgID,
		EntityType: entityType,
		EntityID:   entityID,
	}, nil
}
