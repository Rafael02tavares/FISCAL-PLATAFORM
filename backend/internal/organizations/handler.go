package organizations

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Rafael02tavares/FISCAL-PLATAFORM/backend/internal/auth"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type createOrganizationRequest struct {
	Name              string `json:"name"`
	CNPJ              string `json:"cnpj"`
	TaxRegime         string `json:"tax_regime"`
	CRT               string `json:"crt"`
	StateRegistration string `json:"state_registration"`
	HomeUF            string `json:"home_uf"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "usuário não autenticado")
		return
	}

	var req createOrganizationRequest

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

	org, err := h.service.CreateOrganization(
		r.Context(),
		userID,
		req.Name,
		req.CNPJ,
		req.TaxRegime,
		req.CRT,
		req.StateRegistration,
		req.HomeUF,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidOrganizationData):
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "dados da organização inválidos")
		default:
			writeError(w, http.StatusInternalServerError, "CREATE_ORGANIZATION_FAILED", "não foi possível criar a organização")
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"organization": org,
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "usuário não autenticado")
		return
	}

	orgs, err := h.service.ListOrganizations(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "LIST_ORGANIZATIONS_FAILED", "não foi possível listar as organizações")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"organizations": orgs,
	})
}

func (r *createOrganizationRequest) normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.CNPJ = onlyDigits(r.CNPJ)
	r.TaxRegime = strings.TrimSpace(r.TaxRegime)
	r.CRT = strings.TrimSpace(r.CRT)
	r.StateRegistration = strings.TrimSpace(r.StateRegistration)
	r.HomeUF = strings.ToUpper(strings.TrimSpace(r.HomeUF))
}

func (r *createOrganizationRequest) validate() error {
	switch {
	case r.Name == "":
		return errors.New("nome é obrigatório")
	case len([]rune(r.Name)) < 2:
		return errors.New("nome deve ter pelo menos 2 caracteres")
	case len([]rune(r.Name)) > 150:
		return errors.New("nome deve ter no máximo 150 caracteres")
	case r.CNPJ != "" && len(r.CNPJ) != 14:
		return errors.New("cnpj deve conter 14 dígitos")
	case r.HomeUF != "" && len(r.HomeUF) != 2:
		return errors.New("uf deve conter 2 caracteres")
	default:
		return nil
	}
}

func onlyDigits(value string) string {
	var b strings.Builder
	b.Grow(len(value))

	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}

	return b.String()
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