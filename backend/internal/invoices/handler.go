package invoices

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path"
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
	orgID, ok := h.authorizeOrganizationRequest(w, r)
	if !ok {
		return
	}

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_MULTIPART_FORM", "formulário multipart inválido")
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "FILE_REQUIRED", "arquivo é obrigatório no campo 'file'")
		return
	}
	defer file.Close()

	filename := strings.TrimSpace(fileHeader.Filename)
	if filename == "" {
		writeError(w, http.StatusBadRequest, "INVALID_FILENAME", "nome do arquivo é obrigatório")
		return
	}

	xmlBytes, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "FILE_READ_FAILED", "não foi possível ler o arquivo enviado")
		return
	}

	if len(bytes.TrimSpace(xmlBytes)) == 0 {
		writeError(w, http.StatusBadRequest, "EMPTY_FILE", "o arquivo enviado está vazio")
		return
	}

	result, err := h.service.ProcessXML(
		r.Context(),
		orgID,
		string(xmlBytes),
		bytes.NewReader(xmlBytes),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "PROCESS_XML_FAILED", "não foi possível processar o XML enviado")
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.authorizeOrganizationRequest(w, r)
	if !ok {
		return
	}

	invoices, err := h.service.ListInvoices(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "LIST_INVOICES_FAILED", "não foi possível listar as notas fiscais")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"invoices": invoices,
	})
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.authorizeOrganizationRequest(w, r)
	if !ok {
		return
	}

	invoiceID, err := extractInvoiceID(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusNotFound, "INVOICE_NOT_FOUND", err.Error())
		return
	}

	invoice, err := h.service.GetInvoiceByID(r.Context(), orgID, invoiceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "GET_INVOICE_FAILED", "não foi possível buscar a nota fiscal")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"invoice": invoice,
	})
}

func (h *Handler) authorizeOrganizationRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "usuário não autenticado")
		return "", false
	}

	orgID := strings.TrimSpace(r.Header.Get("X-Organization-ID"))
	if orgID == "" {
		writeError(w, http.StatusBadRequest, "MISSING_ORGANIZATION_ID", "X-Organization-ID é obrigatório")
		return "", false
	}

	allowed, err := h.organizationService.UserBelongsToOrganization(r.Context(), userID, orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ORGANIZATION_VALIDATION_FAILED", "não foi possível validar acesso à organização")
		return "", false
	}

	if !allowed {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "usuário sem acesso a esta organização")
		return "", false
	}

	return orgID, true
}

func extractInvoiceID(urlPath string) (string, error) {
	cleanPath := strings.TrimSpace(urlPath)
	if cleanPath == "" {
		return "", errors.New("identificador da nota fiscal não informado")
	}

	last := strings.TrimSpace(path.Base(cleanPath))
	if last == "" || last == "." || last == "invoices" || last == "upload" {
		return "", errors.New("identificador da nota fiscal inválido")
	}

	return last, nil
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