package icmsrates

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rafa/fiscal-platform/backend/internal/auth"
	"github.com/rafa/fiscal-platform/backend/internal/organizations"
)

type Handler struct {
	service    *Service
	orgService *organizations.Service
}

type upsertStateRateRequest struct {
	UF              string `json:"uf"`
	InternalRate    string `json:"internal_rate"`
	FCPRate         string `json:"fcp_rate"`
	ValidFrom       string `json:"valid_from"`
	ValidTo         string `json:"valid_to"`
	SourceReference string `json:"source_reference"`
	SourceURL       string `json:"source_url"`
	Notes           string `json:"notes"`
}

func NewHandler(service *Service, orgService *organizations.Service) *Handler {
	return &Handler{service: service, orgService: orgService}
}

func (h *Handler) ListStateRates(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeRequest(w, r); !ok {
		return
	}

	items, err := h.service.ListStateRates(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{
				"code":    "LIST_ICMS_RATES_FAILED",
				"message": "nao foi possivel listar as aliquotas internas de ICMS",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) UpsertStateRate(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeRequest(w, r); !ok {
		return
	}

	var req upsertStateRateRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"code":    "INVALID_BODY",
				"message": "corpo da requisicao invalido",
			},
		})
		return
	}

	id, err := h.service.UpsertStateRate(r.Context(), UpsertStateRateParams{
		UF:              req.UF,
		InternalRate:    req.InternalRate,
		FCPRate:         req.FCPRate,
		ValidFrom:       req.ValidFrom,
		ValidTo:         req.ValidTo,
		SourceReference: req.SourceReference,
		SourceURL:       req.SourceURL,
		Notes:           req.Notes,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"code":    "UPSERT_ICMS_RATE_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":      id,
		"message": "aliquota interna de ICMS salva com sucesso",
	})
}

func (h *Handler) ResolveReference(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeRequest(w, r); !ok {
		return
	}

	issuerUF := strings.TrimSpace(r.URL.Query().Get("issuer_uf"))
	recipientUF := strings.TrimSpace(r.URL.Query().Get("recipient_uf"))

	item, err := h.service.ResolveReference(r.Context(), issuerUF, recipientUF)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{
				"code":    "RESOLVE_ICMS_REFERENCE_FAILED",
				"message": "nao foi possivel resolver a referencia de partilha",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"item": item,
	})
}

func (h *Handler) authorizeRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": map[string]string{
				"code":    "UNAUTHORIZED",
				"message": "usuario nao autenticado",
			},
		})
		return "", false
	}

	organizationID := strings.TrimSpace(r.Header.Get("X-Organization-ID"))
	if organizationID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"code":    "MISSING_ORGANIZATION_ID",
				"message": "X-Organization-ID e obrigatorio",
			},
		})
		return "", false
	}

	allowed, err := h.orgService.UserBelongsToOrganization(r.Context(), userID, organizationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{
				"code":    "ORGANIZATION_VALIDATION_FAILED",
				"message": "nao foi possivel validar acesso a organizacao",
			},
		})
		return "", false
	}

	if !allowed {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error": map[string]string{
				"code":    "FORBIDDEN",
				"message": "usuario sem acesso a esta organizacao",
			},
		})
		return "", false
	}

	return organizationID, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
