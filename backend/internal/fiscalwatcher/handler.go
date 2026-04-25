package fiscalwatcher

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListSources(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListSources(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{
				"code":    "WATCHER_SOURCES_LIST_FAILED",
				"message": "nao foi possivel listar as fontes monitoradas",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	items, err := h.service.ListEvents(r.Context(), r.URL.Query().Get("status"), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{
				"code":    "WATCHER_EVENTS_LIST_FAILED",
				"message": "nao foi possivel listar os eventos de verificacao",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) RunCheck(w http.ResponseWriter, r *http.Request) {
	var input struct {
		SourceCode string `json:"source_code"`
	}

	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&input)
	}

	items, err := h.service.RunCheck(r.Context(), input.SourceCode)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"code":    "WATCHER_RUN_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "verificacao registrada com sucesso",
		"items":   items,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
