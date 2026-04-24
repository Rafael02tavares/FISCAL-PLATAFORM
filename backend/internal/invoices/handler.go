package invoices

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
		writeError(w, http.StatusBadRequest, "INVALID_MULTIPART_FORM", "formulario multipart invalido")
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		files = r.MultipartForm.File["file"]
	}

	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "FILE_REQUIRED", "arquivo e obrigatorio no campo 'files' ou 'file'")
		return
	}

	results := make([]BatchUploadItemResult, 0, len(files))
	successCount := 0
	failedCount := 0

	for _, fileHeader := range files {
		filename := strings.TrimSpace(fileHeader.Filename)
		if filename == "" {
			results = append(results, BatchUploadItemResult{
				FileName: "arquivo-sem-nome",
				Success:  false,
				Error:    "nome do arquivo e obrigatorio",
			})
			failedCount++
			continue
		}

		file, err := fileHeader.Open()
		if err != nil {
			results = append(results, BatchUploadItemResult{
				FileName: filename,
				Success:  false,
				Error:    "nao foi possivel abrir o arquivo enviado",
			})
			failedCount++
			continue
		}

		xmlBytes, err := io.ReadAll(file)
		_ = file.Close()
		if err != nil {
			results = append(results, BatchUploadItemResult{
				FileName: filename,
				Success:  false,
				Error:    "nao foi possivel ler o arquivo enviado",
			})
			failedCount++
			continue
		}

		if len(bytes.TrimSpace(xmlBytes)) == 0 {
			results = append(results, BatchUploadItemResult{
				FileName: filename,
				Success:  false,
				Error:    "o arquivo enviado esta vazio",
			})
			failedCount++
			continue
		}

		result, err := h.service.ProcessXML(
			r.Context(),
			orgID,
			string(xmlBytes),
			bytes.NewReader(xmlBytes),
		)
		if err != nil {
			results = append(results, BatchUploadItemResult{
				FileName: filename,
				Success:  false,
				Error:    fmt.Sprintf("nao foi possivel processar o XML enviado: %v", err),
			})
			failedCount++
			continue
		}

		results = append(results, BatchUploadItemResult{
			FileName:   filename,
			InvoiceID:  result.InvoiceID,
			ItemsCount: result.ItemsCount,
			Success:    true,
		})
		successCount++
	}

	status := http.StatusCreated
	if successCount == 0 {
		status = http.StatusBadRequest
	}

	writeJSON(w, status, BatchUploadResult{
		TotalFiles:   len(files),
		SuccessCount: successCount,
		FailedCount:  failedCount,
		Results:      results,
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.authorizeOrganizationRequest(w, r)
	if !ok {
		return
	}

	invoices, err := h.service.ListInvoices(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "LIST_INVOICES_FAILED", "nao foi possivel listar as notas fiscais")
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
		writeError(w, http.StatusInternalServerError, "GET_INVOICE_FAILED", "nao foi possivel buscar a nota fiscal")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"invoice": invoice,
	})
}

func (h *Handler) authorizeOrganizationRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "usuario nao autenticado")
		return "", false
	}

	orgID := strings.TrimSpace(r.Header.Get("X-Organization-ID"))
	if orgID == "" {
		writeError(w, http.StatusBadRequest, "MISSING_ORGANIZATION_ID", "X-Organization-ID e obrigatorio")
		return "", false
	}

	allowed, err := h.organizationService.UserBelongsToOrganization(r.Context(), userID, orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ORGANIZATION_VALIDATION_FAILED", "nao foi possivel validar acesso a organizacao")
		return "", false
	}

	if !allowed {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "usuario sem acesso a esta organizacao")
		return "", false
	}

	return orgID, true
}

func extractInvoiceID(urlPath string) (string, error) {
	cleanPath := strings.TrimSpace(urlPath)
	if cleanPath == "" {
		return "", errors.New("identificador da nota fiscal nao informado")
	}

	last := strings.TrimSpace(path.Base(cleanPath))
	if last == "" || last == "." || last == "invoices" || last == "upload" {
		return "", errors.New("identificador da nota fiscal invalido")
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
