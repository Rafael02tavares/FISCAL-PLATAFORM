package invoices

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type InvoiceTaxProcessingService struct {
	integration *TaxEngineIntegrationService
}

func NewInvoiceTaxProcessingService(
	integration *TaxEngineIntegrationService,
) (*InvoiceTaxProcessingService, error) {
	if integration == nil {
		return nil, errors.New("invoice tax processing service: integration is required")
	}

	return &InvoiceTaxProcessingService{
		integration: integration,
	}, nil
}

type ProcessInvoiceTaxInput struct {
	TenantID       string
	OrganizationID string
	Invoice        TaxEngineInvoiceContext
	Items          []ProcessInvoiceTaxItem
	Metadata       TaxEngineMetadataContext
}

type ProcessInvoiceTaxItem struct {
	InvoiceItemID         string
	ItemNumber            int
	SupplierID            string
	SupplierProductCode   string
	Description           string
	AdditionalDescription string
	GTIN                  string
	NCM                   string
	EXTIPI                string
	CEST                  string
	OriginCode            string
	Unit                  string
	CommercialUnit        string
	TributaryUnit         string
	Quantity              float64
	UnitValue             float64
	GrossValue            float64
	DiscountValue         float64
	FreightValue          float64
	InsuranceValue        float64
	OtherExpensesValue    float64
	IPIValue              float64
	ICMSBaseValue         float64
	TotalValue            float64
}

type ProcessInvoiceTaxResult struct {
	InvoiceID        string
	ProcessedItems   int
	SuccessfulItems  int
	FailedItems      int
	Items            []ProcessedInvoiceTaxItemResult
	StartedAt        time.Time
	FinishedAt       time.Time
}

type ProcessedInvoiceTaxItemResult struct {
	InvoiceItemID string
	ItemNumber    int
	Success       bool
	Output        *EvaluateInvoiceItemResult
	Error         string
}

func (s *InvoiceTaxProcessingService) ProcessInvoice(
	ctx context.Context,
	input ProcessInvoiceTaxInput,
) (*ProcessInvoiceTaxResult, error) {
	if err := validateProcessInvoiceTaxInput(input); err != nil {
		return nil, err
	}

	startedAt := time.Now()

	result := &ProcessInvoiceTaxResult{
		InvoiceID:      strings.TrimSpace(input.Invoice.InvoiceID),
		ProcessedItems: len(input.Items),
		Items:          make([]ProcessedInvoiceTaxItemResult, 0, len(input.Items)),
		StartedAt:      startedAt,
	}

	for _, item := range input.Items {
		evaluateInput := EvaluateInvoiceItemInput{
			TenantID:       strings.TrimSpace(input.TenantID),
			OrganizationID: strings.TrimSpace(input.OrganizationID),
			Invoice:        input.Invoice,
			Item: TaxEngineInvoiceItemContext{
				InvoiceItemID:         strings.TrimSpace(item.InvoiceItemID),
				ItemNumber:            item.ItemNumber,
				SupplierID:            strings.TrimSpace(item.SupplierID),
				SupplierProductCode:   strings.TrimSpace(item.SupplierProductCode),
				Description:           strings.TrimSpace(item.Description),
				AdditionalDescription: strings.TrimSpace(item.AdditionalDescription),
				GTIN:                  strings.TrimSpace(item.GTIN),
				NCM:                   strings.TrimSpace(item.NCM),
				EXTIPI:                strings.TrimSpace(item.EXTIPI),
				CEST:                  strings.TrimSpace(item.CEST),
				OriginCode:            strings.TrimSpace(item.OriginCode),
				Unit:                  strings.TrimSpace(item.Unit),
				CommercialUnit:        strings.TrimSpace(item.CommercialUnit),
				TributaryUnit:         strings.TrimSpace(item.TributaryUnit),
				Quantity:              item.Quantity,
				UnitValue:             item.UnitValue,
				GrossValue:            item.GrossValue,
				DiscountValue:         item.DiscountValue,
				FreightValue:          item.FreightValue,
				InsuranceValue:        item.InsuranceValue,
				OtherExpensesValue:    item.OtherExpensesValue,
				IPIValue:              item.IPIValue,
				ICMSBaseValue:         item.ICMSBaseValue,
				TotalValue:            item.TotalValue,
			},
			Metadata: input.Metadata,
		}

		itemResult := ProcessedInvoiceTaxItemResult{
			InvoiceItemID: strings.TrimSpace(item.InvoiceItemID),
			ItemNumber:    item.ItemNumber,
		}

		output, err := s.integration.EvaluateAndPersistInvoiceItem(ctx, evaluateInput)
		if err != nil {
			itemResult.Success = false
			itemResult.Error = err.Error()
			result.FailedItems++
		} else {
			itemResult.Success = true
			itemResult.Output = output
			result.SuccessfulItems++
		}

		result.Items = append(result.Items, itemResult)
	}

	result.FinishedAt = time.Now()

	return result, nil
}

func validateProcessInvoiceTaxInput(input ProcessInvoiceTaxInput) error {
	if strings.TrimSpace(input.TenantID) == "" {
		return errors.New("invoice tax processing service: tenant_id is required")
	}
	if strings.TrimSpace(input.OrganizationID) == "" {
		return errors.New("invoice tax processing service: organization_id is required")
	}
	if strings.TrimSpace(input.Invoice.InvoiceID) == "" {
		return errors.New("invoice tax processing service: invoice_id is required")
	}
	if len(input.Items) == 0 {
		return errors.New("invoice tax processing service: at least one item is required")
	}

	for i, item := range input.Items {
		if strings.TrimSpace(item.InvoiceItemID) == "" {
			return fmt.Errorf("invoice tax processing service: item at index %d has empty invoice_item_id", i)
		}
		if strings.TrimSpace(item.Description) == "" {
			return fmt.Errorf("invoice tax processing service: item at index %d has empty description", i)
		}
	}

	return nil
}