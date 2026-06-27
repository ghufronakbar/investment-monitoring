package websocket_controller

import (
	"net/http"

	websocket_service "investment_monitoring/internal/module/realtime/websocket/service"
	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/response"

	"github.com/go-chi/chi/v5"
	"nhooyr.io/websocket"
)

type Handler struct {
	svc *websocket_service.Service
}

func NewHandler(svc *websocket_service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.Connect)
	return r
}

func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
	if !h.svc.Enabled() {
		response.Error(w, r, errs.NewBadRequest("WEBSOCKET_DISABLED"), nil)
		return
	}

	claims, err := h.svc.AuthenticateRequest(r)
	if err != nil {
		response.Error(w, r, errs.NewUnauthorized("INVALID_OR_EXPIRED_TOKEN"), nil)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: h.svc.AllowAllOrigins(),
		OriginPatterns:     h.svc.OriginPatterns(),
	})
	if err != nil {
		return
	}

	h.svc.HandleConnection(r.Context(), conn, claims)
}
