package deployments

import (
	"encoding/json"
	"net/http"
)

const defaultEnvironment = "default"

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
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Environment == "" {
		req.Environment = defaultEnvironment
	}

	if err := h.svc.Validate(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := h.svc.Resolve(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}
