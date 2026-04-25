package tax

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/rafa/fiscal-platform/backend/internal/auth"
	"github.com/rafa/fiscal-platform/backend/internal/organizations"
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
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "usuario nao autenticado")
		return
	}

	organizationID := strings.TrimSpace(r.Header.Get("X-Organization-ID"))
	if organizationID == "" {
		writeError(w, http.StatusBadRequest, "MISSING_ORGANIZATION_ID", "X-Organization-ID e obrigatorio")
		return
	}

	allowed, err := h.orgService.UserBelongsToOrganization(r.Context(), userID, organizationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ORGANIZATION_VALIDATION_FAILED", "nao foi possivel validar acesso a organizacao")
		return
	}

	if !allowed {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "usuario sem acesso a esta organizacao")
		return
	}

	var req SuggestRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "corpo da requisicao invalido")
		return
	}

	req.normalize()
	req.OrganizationID = organizationID

	org, err := h.orgService.GetOrganizationByID(r.Context(), organizationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ORGANIZATION_LOAD_FAILED", "nao foi possivel carregar o contexto fiscal da organizacao")
		return
	}

	if strings.TrimSpace(req.TaxRegime) == "" {
		req.TaxRegime = strings.TrimSpace(org.TaxRegime)
	}
	if strings.TrimSpace(req.TargetCRT) == "" {
		req.TargetCRT = strings.TrimSpace(org.CRT)
	}
	if strings.TrimSpace(req.OperationCode) == "" {
		req.OperationCode = "sale_consumer_final"
	}
	if strings.TrimSpace(req.EmitterUF) == "" {
		req.EmitterUF = strings.ToUpper(strings.TrimSpace(org.HomeUF))
	}
	if strings.TrimSpace(req.RecipientUF) == "" {
		req.RecipientUF = strings.ToUpper(strings.TrimSpace(org.HomeUF))
	}

	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	resp, err := h.service.Suggest(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusNotFound, "SUGGESTION_NOT_FOUND", "nao foi possivel sugerir perfil tributario para o contexto informado")
		return
	}

	if err := h.service.PersistSuggestion(r.Context(), organizationID, req, resp); err != nil {
		log.Printf("tax suggestion persistence warning: org=%s operation=%s gtin=%s err=%v", organizationID, req.OperationCode, req.GTIN, err)
		resp.Warnings = append(resp.Warnings, "A sugestao foi gerada, mas o log interno nao pode ser salvo nesta tentativa.")
	}

	writeJSON(w, http.StatusOK, resp)
}

func (r *SuggestRequest) normalize() {
	r.GTIN = strings.TrimSpace(r.GTIN)
	r.Description = strings.TrimSpace(r.Description)
	r.NCMCode = strings.TrimSpace(r.NCMCode)
	r.OperationCode = strings.TrimSpace(r.OperationCode)
	r.TaxRegime = strings.TrimSpace(r.TaxRegime)
	r.TargetCRT = strings.TrimSpace(r.TargetCRT)
	r.EmitterUF = strings.ToUpper(strings.TrimSpace(r.EmitterUF))
	r.RecipientUF = strings.ToUpper(strings.TrimSpace(r.RecipientUF))
	r.SourceICMSCST = strings.TrimSpace(r.SourceICMSCST)
	r.SourceICMSRate = strings.TrimSpace(r.SourceICMSRate)
	r.SourcePISCST = strings.TrimSpace(r.SourcePISCST)
	r.SourcePISRate = strings.TrimSpace(r.SourcePISRate)
	r.SourceCOFINSCST = strings.TrimSpace(r.SourceCOFINSCST)
	r.SourceCOFINSRate = strings.TrimSpace(r.SourceCOFINSRate)
	r.SourceCFOP = strings.TrimSpace(r.SourceCFOP)
}

func (r *SuggestRequest) validate() error {
	switch {
	case r.OperationCode == "":
		return errors.New("operation_code e obrigatorio")
	case r.Description == "" && r.GTIN == "" && r.NCMCode == "":
		return errors.New("description, gtin ou ncm_code deve ser informado")
	case r.EmitterUF == "":
		return errors.New("emitter_uf e obrigatorio")
	case len(r.EmitterUF) != 2:
		return errors.New("emitter_uf deve conter 2 caracteres")
	case r.RecipientUF == "":
		return errors.New("recipient_uf e obrigatorio")
	case len(r.RecipientUF) != 2:
		return errors.New("recipient_uf deve conter 2 caracteres")
	default:
		return nil
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
