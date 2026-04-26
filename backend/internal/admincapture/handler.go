package admincapture

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/rafa/fiscal-platform/backend/internal/auth"
	"github.com/rafa/fiscal-platform/backend/internal/organizations"
)

type Handler struct {
	service    *Service
	orgService *organizations.Service
}

func NewHandler(service *Service, orgService *organizations.Service) *Handler {
	return &Handler{service: service, orgService: orgService}
}

type acceptCandidateRequest struct {
	InvoiceItemID string `json:"invoice_item_id"`
}

type acceptProductReviewsRequest struct {
	ProductIDs    []string `json:"product_ids"`
	AcceptAll     bool     `json:"accept_all"`
	MinConfidence float64  `json:"min_confidence"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	limit := 120
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	items, err := h.service.ListCandidates(r.Context(), organizationID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{
				"code":    "LIST_CAPTURE_CANDIDATES_FAILED",
				"message": "nao foi possivel listar a triagem de regras capturadas",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	var req acceptCandidateRequest
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

	org, err := h.orgService.GetOrganizationByID(r.Context(), organizationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{
				"code":    "ORGANIZATION_LOOKUP_FAILED",
				"message": "nao foi possivel carregar o regime da organizacao",
			},
		})
		return
	}

	if err := h.service.AcceptCandidate(
		r.Context(),
		organizationID,
		req.InvoiceItemID,
		strings.TrimSpace(org.TaxRegime),
		strings.TrimSpace(org.CRT),
	); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"code":    "ACCEPT_CAPTURE_CANDIDATE_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "regra capturada integrada ao motor tributario com sucesso",
	})
}

func (h *Handler) ProductReviews(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	limit := 150
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	org, err := h.orgService.GetOrganizationByID(r.Context(), organizationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{
				"code":    "ORGANIZATION_LOOKUP_FAILED",
				"message": "nao foi possivel carregar o regime da organizacao",
			},
		})
		return
	}

	items, err := h.service.ReviewCatalogProducts(
		r.Context(),
		organizationID,
		strings.TrimSpace(org.TaxRegime),
		strings.TrimSpace(org.CRT),
		strings.TrimSpace(org.HomeUF),
		limit,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{
				"code":    "LIST_PRODUCT_REVIEWS_FAILED",
				"message": "nao foi possivel revisar os produtos cadastrados",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

func (h *Handler) AcceptProductReviews(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	var req acceptProductReviewsRequest
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

	org, err := h.orgService.GetOrganizationByID(r.Context(), organizationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{
				"code":    "ORGANIZATION_LOOKUP_FAILED",
				"message": "nao foi possivel carregar o regime da organizacao",
			},
		})
		return
	}

	accepted, failures, err := h.service.AcceptProductReviews(
		r.Context(),
		organizationID,
		req.ProductIDs,
		req.AcceptAll,
		req.MinConfidence,
		strings.TrimSpace(org.TaxRegime),
		strings.TrimSpace(org.CRT),
		strings.TrimSpace(org.HomeUF),
	)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"code":    "ACCEPT_PRODUCT_REVIEWS_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"message":  "revisoes de produtos aceitas com sucesso",
		"accepted": accepted,
		"failures": failures,
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
