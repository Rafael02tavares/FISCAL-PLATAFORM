package invoices

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/rafa/fiscal-platform/backend/internal/auth"
	"github.com/rafa/fiscal-platform/backend/internal/organizations"
)

type Handler struct {
	service             *Service
	organizationService *organizations.Service
}

func NewHandler(service *Service, organizationService *organizations.Service) *Handler {
	return &Handler{
		service:             service,
		organizationService: organizationService,
	}
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		http.Error(w, "invalid multipart form: "+err.Error(), http.StatusBadRequest)
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "user not authenticated", http.StatusUnauthorized)
		return
	}

	orgID := r.Header.Get("X-Organization-ID")
	if orgID == "" {
		http.Error(w, "X-Organization-ID is required", http.StatusBadRequest)
		return
	}

	allowed, err := h.organizationService.UserBelongsToOrganization(r.Context(), userID, orgID)
	if err != nil {
		http.Error(w, "cannot validate organization access: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !allowed {
		http.Error(w, "forbidden for this organization", http.StatusForbidden)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	xmlBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "cannot read file", http.StatusInternalServerError)
		return
	}

	result, err := h.service.ProcessXML(
		r.Context(),
		orgID,
		string(xmlBytes),
		bytes.NewReader(xmlBytes),
	)
	if err != nil {
		http.Error(w, "cannot process xml: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(result)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "user not authenticated", http.StatusUnauthorized)
		return
	}

	orgID := r.Header.Get("X-Organization-ID")
	if orgID == "" {
		http.Error(w, "X-Organization-ID is required", http.StatusBadRequest)
		return
	}

	allowed, err := h.organizationService.UserBelongsToOrganization(r.Context(), userID, orgID)
	if err != nil {
		http.Error(w, "cannot validate organization access: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !allowed {
		http.Error(w, "forbidden for this organization", http.StatusForbidden)
		return
	}

	invoices, err := h.service.ListInvoices(r.Context(), orgID)
	if err != nil {
		http.Error(w, "cannot list invoices: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"invoices": invoices,
	})
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "user not authenticated", http.StatusUnauthorized)
		return
	}

	orgID := r.Header.Get("X-Organization-ID")
	if orgID == "" {
		http.Error(w, "X-Organization-ID is required", http.StatusBadRequest)
		return
	}

	allowed, err := h.organizationService.UserBelongsToOrganization(r.Context(), userID, orgID)
	if err != nil {
		http.Error(w, "cannot validate organization access: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !allowed {
		http.Error(w, "forbidden for this organization", http.StatusForbidden)
		return
	}

	prefix := "/invoices/"
	path := r.URL.Path

	if !strings.HasPrefix(path, prefix) {
		http.NotFound(w, r)
		return
	}

	invoiceID := strings.TrimPrefix(path, prefix)
	if invoiceID == "" || invoiceID == "upload" {
		http.NotFound(w, r)
		return
	}

	invoice, err := h.service.GetInvoiceByID(r.Context(), orgID, invoiceID)
	if err != nil {
		http.Error(w, "cannot get invoice: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"invoice": invoice,
	})
}
