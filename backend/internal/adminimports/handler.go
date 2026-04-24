package adminimports

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

func (h *Handler) ListImportBatches(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	sourceName := strings.TrimSpace(r.URL.Query().Get("source_name"))

	items, err := h.service.ListImportBatches(r.Context(), sourceName, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{
				"code":    "IMPORT_BATCHES_LIST_FAILED",
				"message": "nao foi possivel consultar os lotes de importacao",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

func (h *Handler) UploadNCM(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"code":    "INVALID_MULTIPART_FORM",
				"message": "formulario de upload invalido",
			},
		})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"code":    "MISSING_FILE",
				"message": "arquivo CSV obrigatorio",
			},
		})
		return
	}
	defer file.Close()

	sourceName := strings.TrimSpace(r.FormValue("source_name"))

	if err := h.service.ImportNCMCSV(r.Context(), ImportNCMParams{
		File:         file,
		FileName:     header.Filename,
		SourceName:   sourceName,
		VersionLabel: r.FormValue("version_label"),
	}); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"code":    "NCM_IMPORT_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	items, _ := h.service.ListImportBatches(r.Context(), sourceName, 10)

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "importacao de NCM concluida com sucesso",
		"items":   items,
	})
}

func (h *Handler) UploadCFOP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"code":    "INVALID_MULTIPART_FORM",
				"message": "formulario de upload invalido",
			},
		})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"code":    "MISSING_FILE",
				"message": "arquivo CSV obrigatorio",
			},
		})
		return
	}
	defer file.Close()

	sourceName := strings.TrimSpace(r.FormValue("source_name"))

	if err := h.service.ImportCFOPCSV(r.Context(), ImportCFOPParams{
		File:         file,
		FileName:     header.Filename,
		SourceName:   sourceName,
		VersionLabel: r.FormValue("version_label"),
	}); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"code":    "CFOP_IMPORT_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	items, _ := h.service.ListImportBatches(r.Context(), sourceName, 10)

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "importacao de CFOP concluida com sucesso",
		"items":   items,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
