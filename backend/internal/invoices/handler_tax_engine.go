package invoices

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type InvoiceQueryService interface {
	GetInvoiceForTaxProcessing(ctx context.Context, invoiceID string, organizationID string) (*InvoiceTaxProcessingView, error)
}

type InvoiceTaxProcessingView struct {
	InvoiceID   string
	Number      string
	Series      string
	AccessKey   string
	IssueDate   time.Time
	Issuer      TaxEnginePartyContext
	Recipient   TaxEnginePartyContext
	Operation   TaxEngineOperationContext
	Items       []ProcessInvoiceTaxItem
}

type TaxEngineHandler struct {
	processor    *InvoiceTaxProcessingService
	invoiceQuery InvoiceQueryService
}

func NewTaxEngineHandler(
	processor *InvoiceTaxProcessingService,
	invoiceQuery InvoiceQueryService,
) (*TaxEngineHandler, error) {
	if processor == nil {
		return nil, errors.New("tax engine handler: processor is required")
	}
	if invoiceQuery == nil {
		return nil, errors.New("tax engine handler: invoice query service is required")
	}

	return &TaxEngineHandler{
		processor:    processor,
		invoiceQuery: invoiceQuery,
	}, nil
}

type ProcessInvoiceTaxesRequest struct {
	TenantID       string `json:"tenant_id"`
	OrganizationID string `json:"organization_id"`
	InvoiceID      string `json:"invoice_id"`
	UserID         string `json:"user_id"`
	RequestID      string `json:"request_id"`
	Source         string `json:"source"`
	SourceFileName string `json:"source_file_name"`
}

type ProcessInvoiceTaxesResponse struct {
	InvoiceID       string                           `json:"invoice_id"`
	ProcessedItems  int                              `json:"processed_items"`
	SuccessfulItems int                              `json:"successful_items"`
	FailedItems     int                              `json:"failed_items"`
	StartedAt       time.Time                        `json:"started_at"`
	FinishedAt      time.Time                        `json:"finished_at"`
	Items           []ProcessInvoiceTaxesItemResult  `json:"items"`
}

type ProcessInvoiceTaxesItemResult struct {
	InvoiceItemID string `json:"invoice_item_id"`
	ItemNumber    int    `json:"item_number"`
	Success       bool   `json:"success"`
	Error         string `json:"error,omitempty"`
}

func (h *TaxEngineHandler) HandleProcessInvoiceTaxes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": "method not allowed",
		})
		return
	}

	var req ProcessInvoiceTaxesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid request body",
		})
		return
	}

	req = sanitizeProcessInvoiceTaxesRequest(req)

	if err := validateProcessInvoiceTaxesRequest(req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": err.Error(),
		})
		return
	}

	invoice, err := h.invoiceQuery.GetInvoiceForTaxProcessing(r.Context(), req.InvoiceID, req.OrganizationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "failed to load invoice for tax processing",
		})
		return
	}
	if invoice == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error": "invoice not found",
		})
		return
	}

	result, err := h.processor.ProcessInvoice(r.Context(), ProcessInvoiceTaxInput{
		TenantID:       req.TenantID,
		OrganizationID: req.OrganizationID,
		Invoice: TaxEngineInvoiceContext{
			InvoiceID: req.InvoiceID,
			Number:    invoice.Number,
			Series:    invoice.Series,
			AccessKey: invoice.AccessKey,
			IssueDate: invoice.IssueDate,
			Issuer:    invoice.Issuer,
			Recipient: invoice.Recipient,
			Operation: invoice.Operation,
		},
		Items:    invoice.Items,
		Metadata: buildTaxEngineMetadata(req),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "failed to process invoice taxes",
		})
		return
	}

	resp := ProcessInvoiceTaxesResponse{
		InvoiceID:       result.InvoiceID,
		ProcessedItems:  result.ProcessedItems,
		SuccessfulItems: result.SuccessfulItems,
		FailedItems:     result.FailedItems,
		StartedAt:       result.StartedAt,
		FinishedAt:      result.FinishedAt,
		Items:           make([]ProcessInvoiceTaxesItemResult, 0, len(result.Items)),
	}

	for _, item := range result.Items {
		resp.Items = append(resp.Items, ProcessInvoiceTaxesItemResult{
			InvoiceItemID: item.InvoiceItemID,
			ItemNumber:    item.ItemNumber,
			Success:       item.Success,
			Error:         item.Error,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func buildTaxEngineMetadata(req ProcessInvoiceTaxesRequest) TaxEngineMetadataContext {
	source := req.Source
	if source == "" {
		source = "invoice_tax_processing"
	}

	return TaxEngineMetadataContext{
		UserID:         req.UserID,
		RequestID:      req.RequestID,
		ImportedAt:     time.Now(),
		Source:         source,
		SourceFileName: req.SourceFileName,
	}
}

func sanitizeProcessInvoiceTaxesRequest(req ProcessInvoiceTaxesRequest) ProcessInvoiceTaxesRequest {
	req.TenantID = strings.TrimSpace(req.TenantID)
	req.OrganizationID = strings.TrimSpace(req.OrganizationID)
	req.InvoiceID = strings.TrimSpace(req.InvoiceID)
	req.UserID = strings.TrimSpace(req.UserID)
	req.RequestID = strings.TrimSpace(req.RequestID)
	req.Source = strings.TrimSpace(req.Source)
	req.SourceFileName = strings.TrimSpace(req.SourceFileName)
	return req
}

func validateProcessInvoiceTaxesRequest(req ProcessInvoiceTaxesRequest) error {
	if req.TenantID == "" {
		return errors.New("tenant_id is required")
	}
	if req.OrganizationID == "" {
		return errors.New("organization_id is required")
	}
	if req.InvoiceID == "" {
		return errors.New("invoice_id is required")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}