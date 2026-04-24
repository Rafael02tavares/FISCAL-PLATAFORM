package adminpartilha

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

type createRuleRequest struct {
	Code                 string   `json:"code"`
	Name                 string   `json:"name"`
	UF                   string   `json:"uf"`
	Priority             int      `json:"priority"`
	Status               string   `json:"status"`
	ValidFrom            string   `json:"valid_from"`
	ValidTo              string   `json:"valid_to"`
	LegalBasisIDs        []string `json:"legal_basis_ids"`
	IssuerUF             string   `json:"issuer_uf"`
	RecipientUF          string   `json:"recipient_uf"`
	OperationScope       string   `json:"operation_scope"`
	OperationType        string   `json:"operation_type"`
	FinalConsumerMode    string   `json:"final_consumer_mode"`
	RecipientContributor string   `json:"recipient_contributor"`
	CRT                  string   `json:"crt"`
	CFOPPrefix           string   `json:"cfop_prefix"`
	NCMPrefix            string   `json:"ncm_prefix"`
	InternalRate         string   `json:"internal_rate"`
	InterstateRate       string   `json:"interstate_rate"`
	FCPRate              string   `json:"fcp_rate"`
	Applies              bool     `json:"applies"`
	Reason               string   `json:"reason"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeRequest(w, r); !ok {
		return
	}

	limit := 120
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	items, err := h.service.ListDIFALRules(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{
				"code":    "LIST_DIFAL_RULES_FAILED",
				"message": "nao foi possivel listar as regras de partilha de ICMS",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeRequest(w, r); !ok {
		return
	}

	var req createRuleRequest
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

	id, err := h.service.CreateDIFALRule(r.Context(), CreateDIFALRuleParams{
		Code:                 req.Code,
		Name:                 req.Name,
		UF:                   req.UF,
		Priority:             req.Priority,
		Status:               req.Status,
		ValidFrom:            req.ValidFrom,
		ValidTo:              req.ValidTo,
		LegalBasisIDs:        req.LegalBasisIDs,
		IssuerUF:             req.IssuerUF,
		RecipientUF:          req.RecipientUF,
		OperationScope:       req.OperationScope,
		OperationType:        req.OperationType,
		FinalConsumerMode:    req.FinalConsumerMode,
		RecipientContributor: req.RecipientContributor,
		CRT:                  req.CRT,
		CFOPPrefix:           req.CFOPPrefix,
		NCMPrefix:            req.NCMPrefix,
		InternalRate:         req.InternalRate,
		InterstateRate:       req.InterstateRate,
		FCPRate:              req.FCPRate,
		Applies:              req.Applies,
		Reason:               req.Reason,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"code":    "CREATE_DIFAL_RULE_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":      id,
		"message": "regra de partilha de ICMS criada com sucesso",
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
