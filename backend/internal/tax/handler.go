package tax

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Rafael02tavares/FISCAL-PLATAFORM/backend/internal/auth"
	"github.com/Rafael02tavares/FISCAL-PLATAFORM/backend/internal/organizations"
)

type Handler struct {
	service    *Service
	orgService *organizations.Service
}

func NewHandler(service *Service, orgService *organizations.Service) *Handler {
	return &Handler{
		service:    service,
		orgService: orgService,
	}
}

func (h *Handler) Suggest(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "usuário não autenticado")
		return
	}

	organizationID := strings.TrimSpace(r.Header.Get("X-Organization-ID"))
	if organizationID == "" {
		writeError(w, http.StatusBadRequest, "MISSING_ORGANIZATION_ID", "X-Organization-ID é obrigatório")
		return
	}

	allowed, err := h.orgService.UserBelongsToOrganization(r.Context(), userID, organizationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ORGANIZATION_VALIDATION_FAILED", "não foi possível validar acesso à organização")
		return
	}

	if !allowed {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "usuário sem acesso a esta organização")
		return
	}

	var req SuggestRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "corpo da requisição inválido")
		return
	}

	req.normalize()

	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	resp, err := h.service.Suggest(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidSuggestionInput):
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "dados insuficientes para gerar a sugestão")
		case errors.Is(err, ErrSuggestionNotFound):
			writeError(w, http.StatusNotFound, "SUGGESTION_NOT_FOUND", "não foi possível sugerir perfil tributário para o contexto informado")
		default:
			writeError(w, http.StatusInternalServerError, "SUGGESTION_FAILED", "não foi possível gerar a sugestão tributária")
		}
		return
	}

	if err := h.service.PersistSuggestion(r.Context(), organizationID, req, resp); err != nil {
		writeError(w, http.StatusInternalServerError, "PERSIST_SUGGESTION_FAILED", "a sugestão foi gerada, mas não pôde ser salva")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (r *SuggestRequest) normalize() {
	r.GTIN = normalizeGTIN(r.GTIN)
	r.Description = strings.TrimSpace(r.Description)
	r.OperationCode = strings.TrimSpace(r.OperationCode)
	r.TaxRegime = strings.TrimSpace(r.TaxRegime)
	r.EmitterUF = strings.ToUpper(strings.TrimSpace(r.EmitterUF))
	r.RecipientUF = strings.ToUpper(strings.TrimSpace(r.RecipientUF))
}

func (r *SuggestRequest) validate() error {
	switch {
	case r.OperationCode == "":
		return errors.New("operation_code é obrigatório")
	case r.Description == "" && r.GTIN == "":
		return errors.New("description ou gtin deve ser informado")
	case r.EmitterUF == "":
		return errors.New("emitter_uf é obrigatório")
	case len(r.EmitterUF) != 2:
		return errors.New("emitter_uf deve conter 2 caracteres")
	case r.RecipientUF == "":
		return errors.New("recipient_uf é obrigatório")
	case len(r.RecipientUF) != 2:
		return errors.New("recipient_uf deve conter 2 caracteres")
	default:
		return nil
	}
}

func normalizeGTIN(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "SEM GTIN") {
		return ""
	}
	return value
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}