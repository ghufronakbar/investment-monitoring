// internal/module/support/uploader/controller/handler.go
package uploader_controller

import (
	"io"
	"net/http"
	"strings"

	cloudinary_uploader "investment_monitoring/internal/module/support/uploader/service"
	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/response"

	"github.com/go-chi/chi/v5"
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

	// Validate file size
	if header.Size > cloudinary_uploader.MaxFileSize {
		response.Error(w, r, errs.NewBadRequest("FILE_TOO_LARGE_5MB_MAX"), nil)
		return
	}

	// Validate MIME type
	contentType := header.Header.Get("Content-Type")
	if !cloudinary_uploader.AllowedMimeTypes[contentType] {
		response.Error(w, r, errs.NewBadRequest("INVALID_FILE_TYPE"), nil)
		return
	}

	// Determine resource type
	resourceType := "image"
	if contentType == "application/pdf" {
		resourceType = "pdf"
	}

	result, err := h.uploaderSvc.UploadSingle(r.Context(), file, resourceType)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_UPLOAD", result)
}

// UploadMultiple handles POST /support/upload/multiple
// Accepts multipart form-data with field "files" (multiple)
func (h *Handler) UploadMultiple(w http.ResponseWriter, r *http.Request) {
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

	var readers []io.Reader
	var resourceTypes []string

	for _, fileHeader := range files {
		// Validate file size
		if fileHeader.Size > cloudinary_uploader.MaxFileSize {
			response.Error(w, r, errs.NewBadRequest("FILE_TOO_LARGE_5MB_MAX"), nil)
			return
		}

		// Validate MIME type
		contentType := fileHeader.Header.Get("Content-Type")
		if !cloudinary_uploader.AllowedMimeTypes[contentType] {
			response.Error(w, r, errs.NewBadRequest("INVALID_FILE_TYPE"), nil)
			return
		}

		file, err := fileHeader.Open()
		if err != nil {
			response.Error(w, r, errs.NewBadRequest("FAILED_TO_OPEN_FILE"), nil)
			return
		}
		defer file.Close()

		// Determine resource type
		resourceType := "image"
		if strings.Contains(contentType, "pdf") {
			resourceType = "pdf"
		}

		readers = append(readers, file)
		resourceTypes = append(resourceTypes, resourceType)
	}

	result, err := h.uploaderSvc.UploadMultiple(r.Context(), readers, resourceTypes)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_UPLOAD_MULTIPLE", result)
}
