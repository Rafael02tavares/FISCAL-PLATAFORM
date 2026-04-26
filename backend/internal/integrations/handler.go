package integrations

import (
	"encoding/json"
	"errors"
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

type saveCosmosRequest struct {
	Enabled  bool   `json:"enabled"`
	BaseURL  string `json:"base_url"`
	APIToken string `json:"api_token"`
	Notes    string `json:"notes"`
}

type testCosmosRequest struct {
	GTIN     string `json:"gtin"`
	APIToken string `json:"api_token"`
}

type searchCosmosRequest struct {
	Query    string `json:"query"`
	APIToken string `json:"api_token"`
	Limit    int    `json:"limit"`
}

type saveOpenAIRequest struct {
	Enabled   bool   `json:"enabled"`
	BaseURL   string `json:"base_url"`
	ModelName string `json:"model_name"`
	APIToken  string `json:"api_token"`
	Notes     string `json:"notes"`
}

type testOpenAIRequest struct {
	APIToken    string `json:"api_token"`
	ModelName   string `json:"model_name"`
	Description string `json:"description"`
	GTIN        string `json:"gtin"`
	NCM         string `json:"ncm"`
	CEST        string `json:"cest"`
	UF          string `json:"uf"`
	TaxRegime   string `json:"tax_regime"`
	Operation   string `json:"operation"`
}

func (h *Handler) GetCosmos(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	item, err := h.service.GetCosmos(r.Context(), organizationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "GET_INTEGRATION_FAILED", "nao foi possivel consultar a integracao Cosmos")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) SaveCosmos(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	var req saveCosmosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "corpo da requisicao invalido")
		return
	}

	item, err := h.service.SaveCosmos(r.Context(), SaveCosmosParams{
		OrganizationID: organizationID,
		Enabled:        req.Enabled,
		BaseURL:        req.BaseURL,
		APIToken:       req.APIToken,
		Notes:          req.Notes,
	})
	if err != nil {
		status := http.StatusInternalServerError
		code := "SAVE_INTEGRATION_FAILED"
		message := "nao foi possivel salvar a integracao Cosmos"
		if errors.Is(err, ErrInvalidIntegrationInput) {
			status = http.StatusBadRequest
			code = "INVALID_INTEGRATION_INPUT"
			message = "dados da integracao invalidos"
		}
		writeError(w, status, code, message)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "integracao Cosmos salva com sucesso",
		"item":    item,
	})
}

func (h *Handler) TestCosmos(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	var req testCosmosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "corpo da requisicao invalido")
		return
	}

	result, err := h.service.TestCosmosGTIN(r.Context(), organizationID, req.GTIN, req.APIToken)
	if err != nil {
		status := http.StatusInternalServerError
		code := "COSMOS_TEST_FAILED"
		message := "nao foi possivel testar a consulta Cosmos"
		if errors.Is(err, ErrInvalidIntegrationInput) {
			status = http.StatusBadRequest
			code = "INVALID_COSMOS_TEST"
			message = "informe um GTIN valido para testar a integracao"
		}
		if errors.Is(err, ErrIntegrationTokenMissing) {
			status = http.StatusBadRequest
			code = "COSMOS_TOKEN_MISSING"
			message = "configure o token Cosmos antes de testar"
		}
		writeError(w, status, code, message)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) SearchCosmos(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	var req searchCosmosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "corpo da requisicao invalido")
		return
	}

	result, err := h.service.SearchCosmosProducts(r.Context(), organizationID, req.Query, req.APIToken, req.Limit)
	if err != nil {
		status := http.StatusInternalServerError
		code := "COSMOS_SEARCH_FAILED"
		message := "nao foi possivel buscar produtos na Cosmos"
		if errors.Is(err, ErrInvalidIntegrationInput) {
			status = http.StatusBadRequest
			code = "INVALID_COSMOS_SEARCH"
			message = "informe uma descricao valida para buscar produtos"
		}
		if errors.Is(err, ErrIntegrationTokenMissing) {
			status = http.StatusBadRequest
			code = "COSMOS_TOKEN_MISSING"
			message = "configure o token Cosmos antes de buscar produtos"
		}
		writeError(w, status, code, message)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetOpenAI(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	item, err := h.service.GetOpenAI(r.Context(), organizationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "GET_INTEGRATION_FAILED", "nao foi possivel consultar a integracao OpenAI")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) SaveOpenAI(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	var req saveOpenAIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "corpo da requisicao invalido")
		return
	}

	item, err := h.service.SaveOpenAI(r.Context(), SaveOpenAIParams{
		OrganizationID: organizationID,
		Enabled:        req.Enabled,
		BaseURL:        req.BaseURL,
		ModelName:      req.ModelName,
		APIToken:       req.APIToken,
		Notes:          req.Notes,
	})
	if err != nil {
		status := http.StatusInternalServerError
		code := "SAVE_INTEGRATION_FAILED"
		message := "nao foi possivel salvar a integracao OpenAI"
		if errors.Is(err, ErrInvalidIntegrationInput) {
			status = http.StatusBadRequest
			code = "INVALID_INTEGRATION_INPUT"
			message = "dados da integracao invalidos"
		}
		writeError(w, status, code, message)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "integracao OpenAI salva com sucesso",
		"item":    item,
	})
}

func (h *Handler) TestOpenAI(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	var req testOpenAIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "corpo da requisicao invalido")
		return
	}

	result, err := h.service.TestOpenAI(r.Context(), organizationID, req.APIToken, req.ModelName, OpenAIClassificationInput{
		Description: req.Description,
		GTIN:        req.GTIN,
		NCM:         req.NCM,
		CEST:        req.CEST,
		UF:          req.UF,
		TaxRegime:   req.TaxRegime,
		Operation:   req.Operation,
	})
	if err != nil {
		status := http.StatusInternalServerError
		code := "OPENAI_TEST_FAILED"
		message := "nao foi possivel testar a integracao OpenAI"
		if errors.Is(err, ErrInvalidIntegrationInput) {
			status = http.StatusBadRequest
			code = "INVALID_OPENAI_TEST"
			message = "dados de teste invalidos"
		}
		if errors.Is(err, ErrIntegrationTokenMissing) {
			status = http.StatusBadRequest
			code = "OPENAI_TOKEN_MISSING"
			message = "configure o token OpenAI antes de testar"
		}
		writeError(w, status, code, message)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) authorizeRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "usuario nao autenticado")
		return "", false
	}

	organizationID := strings.TrimSpace(r.Header.Get("X-Organization-ID"))
	if organizationID == "" {
		writeError(w, http.StatusBadRequest, "MISSING_ORGANIZATION_ID", "X-Organization-ID e obrigatorio")
		return "", false
	}

	allowed, err := h.orgService.UserBelongsToOrganization(r.Context(), userID, organizationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ORGANIZATION_VALIDATION_FAILED", "nao foi possivel validar acesso a organizacao")
		return "", false
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "usuario sem acesso a esta organizacao")
		return "", false
	}

	return organizationID, true
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
