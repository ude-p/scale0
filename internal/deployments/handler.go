package deployments

import "net/http"

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/deployments", h.Create)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
}
