package ingest

import (
	"net/http"

	"github.com/blackbox-agentdiff/api/internal/store"
)

type Handler struct {
	store store.Store
}

func NewHandler(store store.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) HTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusAccepted)
}