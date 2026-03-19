package tax

import (
	"encoding/json"
	"net/http"

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
		http.Error(w, "user not authenticated", http.StatusUnauthorized)
		return
	}

	organizationID := r.Header.Get("X-Organization-ID")
	if organizationID == "" {
		http.Error(w, "X-Organization-ID is required", http.StatusBadRequest)
		return
	}

	allowed, err := h.orgService.UserBelongsToOrganization(r.Context(), userID, organizationID)
	if err != nil {
		http.Error(w, "cannot validate organization access: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !allowed {
		http.Error(w, "forbidden for this organization", http.StatusForbidden)
		return
	}

	var req SuggestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.Suggest(r.Context(), req)
	if err != nil {
		http.Error(w, "cannot suggest tax profile: "+err.Error(), http.StatusNotFound)
		return
	}

	_ = h.service.PersistSuggestion(r.Context(), organizationID, req, resp)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
