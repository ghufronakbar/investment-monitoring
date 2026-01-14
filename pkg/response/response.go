// pkg/response/response.go
package response

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/filter"
	"investment_monitoring/pkg/pagination"

	chimw "github.com/go-chi/chi/v5/middleware"
)

type MetaData struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
	Path      string `json:"path"`
	Method    string `json:"method"`
}

type BaseResponse struct {
	MetaData         MetaData               `json:"metaData"`
	ResponseMessage  string                 `json:"responseMessage"`
	Data             interface{}            `json:"data"`
	ValidationErrors map[string]string      `json:"validationErrors"`
	FilterQuery      *filter.ReqFilter      `json:"filterQuery"`
	Pagination       *pagination.Pagination `json:"pagination"`
}

type writeOpts struct {
	validationErrors map[string]string
	filterQuery      *filter.ReqFilter
	pagination       *pagination.Pagination
}

func write(w http.ResponseWriter, r *http.Request, status int, message string, data interface{}, opts writeOpts) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	metaMsg := http.StatusText(status)

	reqID := chimw.GetReqID(r.Context())
	// opsional: expose juga di header
	// w.Header().Set("X-Request-ID", reqID)

	resp := BaseResponse{
		MetaData: MetaData{
			Code:      status,
			Message:   metaMsg,
			RequestID: reqID,
			Path:      r.URL.Path, // atau r.RequestURI kalau mau include query string
			Method:    r.Method,
		},
		ResponseMessage:  message,
		Data:             normalizeNilSlice(data),
		ValidationErrors: opts.validationErrors,
		FilterQuery:      opts.filterQuery,
		Pagination:       opts.pagination,
	}

	// kalau gagal encode, minimal log (karena header sudah terlanjur ditulis)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Println("encode response error:", err)
	}
}

func OK(w http.ResponseWriter, r *http.Request, message string, data interface{}) {
	write(w, r, http.StatusOK, message, data, writeOpts{})
}

func LIST(w http.ResponseWriter, r *http.Request, message string, data interface{}, filterQuery *filter.ReqFilter, pagination *pagination.Pagination) {
	write(w, r, http.StatusOK, message, data, writeOpts{
		filterQuery: filterQuery,
		pagination:  pagination,
	})
}

func ValidationFailed(w http.ResponseWriter, r *http.Request, errsMap map[string]string) {
	write(w, r, http.StatusBadRequest, "VALIDATION_FAILED", nil, writeOpts{
		validationErrors: errsMap,
	})
}

func Error(w http.ResponseWriter, r *http.Request, err error, data interface{}) {
	fmt.Println(err)
	code := http.StatusInternalServerError
	msg := "INTERNAL_SERVER_ERROR"

	var appErr *errs.AppError
	if errors.As(err, &appErr) {
		code = appErr.Code
		msg = appErr.Message

		if appErr.Code == http.StatusInternalServerError {
			// TODO: log server error
		}

		if appErr.ValidationErrors != nil {
			ValidationFailed(w, r, appErr.ValidationErrors)
			return
		}
	}

	write(w, r, code, msg, data, writeOpts{})
}

func InvalidJsonFormat(w http.ResponseWriter, r *http.Request) {
	write(w, r, http.StatusBadRequest, "INVALID_JSON_FORMAT", nil, writeOpts{})
}

func normalizeNilSlice(data interface{}) interface{} {
	if data == nil {
		return nil
	}
	rv := reflect.ValueOf(data)
	rt := rv.Type()
	if rt.Kind() == reflect.Slice && rv.IsNil() {
		empty := reflect.MakeSlice(rt, 0, 0)
		return empty.Interface()
	}
	return data
}
