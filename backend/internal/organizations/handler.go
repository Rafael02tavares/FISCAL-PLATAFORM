package organizations

import (
	"encoding/json"
	"net/http"

	"github.com/rafa/fiscal-platform/backend/internal/auth"
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
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "user not authenticated", http.StatusUnauthorized)
		return
	}

	var req createOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
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
		http.Error(w, "cannot create organization: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"organization": org,
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "user not authenticated", http.StatusUnauthorized)
		return
	}

	orgs, err := h.service.ListOrganizations(r.Context(), userID)
	if err != nil {
		http.Error(w, "cannot list organizations: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"organizations": orgs,
	})
}
