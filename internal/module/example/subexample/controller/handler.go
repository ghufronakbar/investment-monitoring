// internal/module/example/subexample_controller/handler.go
package subexample_controller

import (
	"fmt"
	"investment_monitoring/internal/http/internal_middleware"
	subexample_service "investment_monitoring/internal/module/example/subexample/service"
	"net/http"

	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/response"
	"investment_monitoring/pkg/utils"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc *subexample_service.SubExampleService
}

func NewHandler(svc *subexample_service.SubExampleService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.GetList)
	r.Get("/{exampleId}", h.Detail)
	r.Delete("/{exampleId}", h.Delete)
	r.Post("/", h.Create)
	r.Patch("/{exampleId}", h.Edit)

	return r
}

func (h *Handler) GetList(w http.ResponseWriter, r *http.Request) {
	_, _ = internal_middleware.GetUserFromContext(r.Context())

	filter := internal_middleware.GetFilterFromContext(r.Context())

	filterQuery := subexample_service.SubExampleGetListParams{
		Search:  filter.Search,
		SortBy:  filter.SortByDB(),
		Offset:  filter.Offset(),
		Limit:   filter.Limit,
		SortDir: filter.Sort,
		Page:    filter.Page,
	}

	res, pagination, err := h.svc.GetList(r.Context(), filterQuery)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.LIST(w, r, "SUCCESS_GET_LIST_EXAMPLE", res, &filter, pagination)
}

func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {

	_, _ = internal_middleware.GetUserFromContext(r.Context())

	exampleId, err := uuid.Parse(chi.URLParam(r, "exampleId"))
	if err != nil {
		response.Error(w, r, errs.NewValidationFailed(map[string]string{"exampleId": "Invalid UUID"}), nil)
		return
	}

	res, err := h.svc.GetDetail(r.Context(), exampleId)
	if err != nil {
		response.Error(w, r, err, res)
		return
	}

	response.OK(w, r, "SUCCESS_GET_DETAIL_EXAMPLE", res)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req subexample_service.SubExampleCreateParams

	prof, _ := internal_middleware.GetUserFromContext(r.Context())
	req.UserID = prof.ID

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	res, err := h.svc.Create(r.Context(), req)
	if err != nil {
		fmt.Println(err)
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_CREATE_EXAMPLE", res)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	var req subexample_service.SubExampleEditParams

	prof, _ := internal_middleware.GetUserFromContext(r.Context())
	req.UserID = prof.ID

	if appErr := utils.ValidateStruct(r.Body, &req); appErr != nil {
		response.ValidationFailed(w, r, appErr.ValidationErrors)
		return
	}

	res, err := h.svc.Edit(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_EDIT_EXAMPLE", res)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	var req subexample_service.SubExampleDeleteParams
	prof, _ := internal_middleware.GetUserFromContext(r.Context())

	exampleId := chi.URLParam(r, "exampleId")

	req.ID = uuid.Must(uuid.Parse(exampleId))
	req.UserID = prof.ID

	res, err := h.svc.Delete(r.Context(), req)
	if err != nil {
		response.Error(w, r, err, nil)
		return
	}

	response.OK(w, r, "SUCCESS_DELETE_EXAMPLE", res)
}
