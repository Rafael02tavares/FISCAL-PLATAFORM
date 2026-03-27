package invoices

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	taxengine "github.com/rafa/fiscal-platform/backend/internal/taxengine/domain"
)

type TaxEngineRunner interface {
	Evaluate(ctx context.Context, input taxengine.EvaluateInput) (*taxengine.EvaluateOutput, error)
}

type TaxDecisionRepository interface {
	UpsertInvoiceItemTaxDecision(ctx context.Context, input UpsertInvoiceItemTaxDecisionInput) error
}

type TaxEngineIntegrationService struct {
	engine   TaxEngineRunner
	decision TaxDecisionRepository
}

func NewTaxEngineIntegrationService(
	engine TaxEngineRunner,
	decisionRepo TaxDecisionRepository,
) (*TaxEngineIntegrationService, error) {
	if engine == nil {
		return nil, errors.New("invoices tax engine integration: engine is required")
	}
	if decisionRepo == nil {
		return nil, errors.New("invoices tax engine integration: decision repository is required")
	}

	return &TaxEngineIntegrationService{
		engine:   engine,
		decision: decisionRepo,
	}, nil
}

type EvaluateInvoiceItemInput struct {
	TenantID       string
	OrganizationID string
	Invoice        TaxEngineInvoiceContext
	Item           TaxEngineInvoiceItemContext
	Metadata       TaxEngineMetadataContext
}

type TaxEngineInvoiceContext struct {
	InvoiceID         string
	Number            string
	Series            string
	AccessKey         string
	IssueDate         time.Time
	Issuer            TaxEnginePartyContext
	Recipient         TaxEnginePartyContext
	Operation         TaxEngineOperationContext
}

type TaxEnginePartyContext struct {
	CNPJ              string
	CPF               string
	UF                string
	CRT               string
	IE                string
	IsContributorICMS bool
	IsFinalConsumer   bool
	TaxRegimeCode     string
	MunicipalityCode  string
	CountryCode       string
}

type TaxEngineOperationContext struct {
	DocumentType          string
	OperationType         string
	OperationScope        string
	CFOP                  string
	FinNFe                string
	PresenceIndicator     string
	PurposeCode           string
	IsReturn              bool
	IsTransfer            bool
	IsResale              bool
	IsImport              bool
	IsExport              bool
	HasInterstateDelivery bool
}

type TaxEngineInvoiceItemContext struct {
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

type TaxEngineMetadataContext struct {
	UserID         string
	RequestID      string
	ImportedAt     time.Time
	Source         string
	SourceFileName string
}

type EvaluateInvoiceItemResult struct {
	Output *taxengine.EvaluateOutput
}

type UpsertInvoiceItemTaxDecisionInput struct {
	TenantID       string
	OrganizationID string
	InvoiceID      string
	InvoiceItemID  string

	ClassificationNCM      string
	ClassificationEXTIPI   *string
	ClassificationCEST     *string
	ClassificationSource   string
	ClassificationConfidence float64
	NeedsReview            bool
	SummaryStatus          string

	ICMSBaseValue   *float64
	ICMSRate        *float64
	ICMSAmount      *float64
	ICMSCST         *string
	ICMSRuleID      *string

	ICMSSTBaseValue *float64
	ICMSSTRate      *float64
	ICMSSTAmount    *float64
	ICMSSTCST       *string
	ICMSSTRuleID    *string

	FCPBaseValue    *float64
	FCPRate         *float64
	FCPAmount       *float64
	FCPRuleID       *string

	DIFALBaseValue          *float64
	DIFALInternalRate       *float64
	DIFALInterstateRate     *float64
	DIFALAmountDestination  *float64
	DIFALAmountOrigin       *float64
	DIFALRuleID             *string

	PISBaseValue   *float64
	PISRate        *float64
	PISAmount      *float64
	PISCST         *string
	PISRuleID      *string

	COFINSBaseValue *float64
	COFINSRate      *float64
	COFINSAmount    *float64
	COFINSCST       *string
	COFINSRuleID    *string

	IPIBaseValue   *float64
	IPIRate        *float64
	IPIAmount      *float64
	IPICST         *string
	IPIRuleID      *string

	Warnings     []string
	Explanations []taxengine.DecisionExplanation
	AuditTrail   []taxengine.AuditStep
}

func (s *TaxEngineIntegrationService) EvaluateAndPersistInvoiceItem(
	ctx context.Context,
	input EvaluateInvoiceItemInput,
) (*EvaluateInvoiceItemResult, error) {
	if err := validateEvaluateInvoiceItemInput(input); err != nil {
		return nil, err
	}

	engineInput := buildTaxEngineEvaluateInput(input)

	output, err := s.engine.Evaluate(ctx, engineInput)
	if err != nil {
		return nil, fmt.Errorf("invoices tax engine integration: evaluate invoice item: %w", err)
	}

	persistInput := buildUpsertInvoiceItemTaxDecisionInput(input, *output)
	if err := s.decision.UpsertInvoiceItemTaxDecision(ctx, persistInput); err != nil {
		return nil, fmt.Errorf("invoices tax engine integration: persist tax decision: %w", err)
	}

	return &EvaluateInvoiceItemResult{
		Output: output,
	}, nil
}

func buildTaxEngineEvaluateInput(input EvaluateInvoiceItemInput) taxengine.EvaluateInput {
	invoiceID := strings.TrimSpace(input.Invoice.InvoiceID)
	invoiceItemID := strings.TrimSpace(input.Item.InvoiceItemID)

	return taxengine.EvaluateInput{
		TenantID:       strings.TrimSpace(input.TenantID),
		OrganizationID: strings.TrimSpace(input.OrganizationID),
		InvoiceID:      stringPtrOrNil(invoiceID),
		InvoiceItemID:  stringPtrOrNil(invoiceItemID),
		Issuer: taxengine.PartyContext{
			CNPJ:              strings.TrimSpace(input.Invoice.Issuer.CNPJ),
			CPF:               strings.TrimSpace(input.Invoice.Issuer.CPF),
			UF:                strings.TrimSpace(input.Invoice.Issuer.UF),
			CRT:               strings.TrimSpace(input.Invoice.Issuer.CRT),
			IE:                strings.TrimSpace(input.Invoice.Issuer.IE),
			IsContributorICMS: input.Invoice.Issuer.IsContributorICMS,
			IsFinalConsumer:   input.Invoice.Issuer.IsFinalConsumer,
			TaxRegimeCode:     strings.TrimSpace(input.Invoice.Issuer.TaxRegimeCode),
			MunicipalityCode:  strings.TrimSpace(input.Invoice.Issuer.MunicipalityCode),
			CountryCode:       strings.TrimSpace(input.Invoice.Issuer.CountryCode),
		},
		Recipient: taxengine.PartyContext{
			CNPJ:              strings.TrimSpace(input.Invoice.Recipient.CNPJ),
			CPF:               strings.TrimSpace(input.Invoice.Recipient.CPF),
			UF:                strings.TrimSpace(input.Invoice.Recipient.UF),
			CRT:               strings.TrimSpace(input.Invoice.Recipient.CRT),
			IE:                strings.TrimSpace(input.Invoice.Recipient.IE),
			IsContributorICMS: input.Invoice.Recipient.IsContributorICMS,
			IsFinalConsumer:   input.Invoice.Recipient.IsFinalConsumer,
			TaxRegimeCode:     strings.TrimSpace(input.Invoice.Recipient.TaxRegimeCode),
			MunicipalityCode:  strings.TrimSpace(input.Invoice.Recipient.MunicipalityCode),
			CountryCode:       strings.TrimSpace(input.Invoice.Recipient.CountryCode),
		},
		Operation: taxengine.OperationContext{
			DocumentType:          taxengine.DocumentType(strings.TrimSpace(input.Invoice.Operation.DocumentType)),
			OperationType:         taxengine.OperationType(strings.TrimSpace(input.Invoice.Operation.OperationType)),
			OperationScope:        taxengine.OperationScope(strings.TrimSpace(input.Invoice.Operation.OperationScope)),
			CFOP:                  strings.TrimSpace(input.Invoice.Operation.CFOP),
			FinNFe:                strings.TrimSpace(input.Invoice.Operation.FinNFe),
			PresenceIndicator:     strings.TrimSpace(input.Invoice.Operation.PresenceIndicator),
			PurposeCode:           strings.TrimSpace(input.Invoice.Operation.PurposeCode),
			IsReturn:              input.Invoice.Operation.IsReturn,
			IsTransfer:            input.Invoice.Operation.IsTransfer,
			IsResale:              input.Invoice.Operation.IsResale,
			IsImport:              input.Invoice.Operation.IsImport,
			IsExport:              input.Invoice.Operation.IsExport,
			HasInterstateDelivery: input.Invoice.Operation.HasInterstateDelivery,
		},
		Product: taxengine.ProductContext{
			ItemNumber:            input.Item.ItemNumber,
			SupplierID:            strings.TrimSpace(input.Item.SupplierID),
			SupplierProductCode:   strings.TrimSpace(input.Item.SupplierProductCode),
			Description:           strings.TrimSpace(input.Item.Description),
			GTIN:                  strings.TrimSpace(input.Item.GTIN),
			NCM:                   strings.TrimSpace(input.Item.NCM),
			EXTIPI:                strings.TrimSpace(input.Item.EXTIPI),
			CEST:                  strings.TrimSpace(input.Item.CEST),
			OriginCode:            strings.TrimSpace(input.Item.OriginCode),
			Unit:                  strings.TrimSpace(input.Item.Unit),
			Quantity:              input.Item.Quantity,
			CommercialUnit:        strings.TrimSpace(input.Item.CommercialUnit),
			TributaryUnit:         strings.TrimSpace(input.Item.TributaryUnit),
			AdditionalDescription: strings.TrimSpace(input.Item.AdditionalDescription),
		},
		Values: taxengine.ValueContext{
			UnitValue:          input.Item.UnitValue,
			GrossValue:         input.Item.GrossValue,
			DiscountValue:      input.Item.DiscountValue,
			FreightValue:       input.Item.FreightValue,
			InsuranceValue:     input.Item.InsuranceValue,
			OtherExpensesValue: input.Item.OtherExpensesValue,
			IPIValue:           input.Item.IPIValue,
			ICMSBaseValue:      input.Item.ICMSBaseValue,
			TotalValue:         input.Item.TotalValue,
		},
		Metadata: taxengine.MetadataContext{
			InvoiceNumber:  strings.TrimSpace(input.Invoice.Number),
			InvoiceSeries:  strings.TrimSpace(input.Invoice.Series),
			InvoiceKey:     strings.TrimSpace(input.Invoice.AccessKey),
			IssueDate:      input.Invoice.IssueDate,
			ImportedAt:     input.Metadata.ImportedAt,
			Source:         strings.TrimSpace(input.Metadata.Source),
			SourceFileName: strings.TrimSpace(input.Metadata.SourceFileName),
			UserID:         strings.TrimSpace(input.Metadata.UserID),
			RequestID:      strings.TrimSpace(input.Metadata.RequestID),
		},
	}
}

func buildUpsertInvoiceItemTaxDecisionInput(
	input EvaluateInvoiceItemInput,
	output taxengine.EvaluateOutput,
) UpsertInvoiceItemTaxDecisionInput {
	result := UpsertInvoiceItemTaxDecisionInput{
		TenantID:                 strings.TrimSpace(input.TenantID),
		OrganizationID:           strings.TrimSpace(input.OrganizationID),
		InvoiceID:                strings.TrimSpace(input.Invoice.InvoiceID),
		InvoiceItemID:            strings.TrimSpace(input.Item.InvoiceItemID),
		ClassificationNCM:        strings.TrimSpace(output.Classification.NCM),
		ClassificationEXTIPI:     cloneOptionalString(output.Classification.EXTIPI),
		ClassificationCEST:       cloneOptionalString(output.Classification.CEST),
		ClassificationSource:     strings.TrimSpace(string(output.Classification.Source)),
		ClassificationConfidence: output.Classification.Confidence,
		NeedsReview:              output.Classification.NeedsReview || output.Summary.RequiresManualReview,
		SummaryStatus:            strings.TrimSpace(string(output.Summary.Status)),
		Warnings:                 cloneStrings(output.Warnings),
		Explanations:             cloneExplanations(output.Explanations),
		AuditTrail:               cloneAuditTrail(output.AuditTrail),
	}

	if output.Taxes.ICMS != nil {
		result.ICMSBaseValue = floatPtr(output.Taxes.ICMS.BaseValue)
		result.ICMSRate = floatPtr(output.Taxes.ICMS.Rate)
		result.ICMSAmount = floatPtr(output.Taxes.ICMS.Amount)
		result.ICMSCST = stringPtrOrNil(strings.TrimSpace(output.Taxes.ICMS.CST))
		result.ICMSRuleID = stringPtrOrNil(strings.TrimSpace(output.Taxes.ICMS.RuleID))
	}

	if output.Taxes.ICMSST != nil {
		result.ICMSSTBaseValue = floatPtr(output.Taxes.ICMSST.BaseValue)
		result.ICMSSTRate = floatPtr(output.Taxes.ICMSST.Rate)
		result.ICMSSTAmount = floatPtr(output.Taxes.ICMSST.Amount)
		result.ICMSSTCST = stringPtrOrNil(strings.TrimSpace(output.Taxes.ICMSST.CST))
		result.ICMSSTRuleID = stringPtrOrNil(strings.TrimSpace(output.Taxes.ICMSST.RuleID))
	}

	if output.Taxes.FCP != nil {
		result.FCPBaseValue = floatPtr(output.Taxes.FCP.BaseValue)
		result.FCPRate = floatPtr(output.Taxes.FCP.Rate)
		result.FCPAmount = floatPtr(output.Taxes.FCP.Amount)
		result.FCPRuleID = stringPtrOrNil(strings.TrimSpace(output.Taxes.FCP.RuleID))
	}

	if output.Taxes.DIFAL != nil {
		result.DIFALBaseValue = floatPtr(output.Taxes.DIFAL.BaseValue)
		result.DIFALInternalRate = floatPtr(output.Taxes.DIFAL.InternalRate)
		result.DIFALInterstateRate = floatPtr(output.Taxes.DIFAL.InterstateRate)
		result.DIFALAmountDestination = floatPtr(output.Taxes.DIFAL.AmountDestinationUF)
		result.DIFALAmountOrigin = floatPtr(output.Taxes.DIFAL.AmountOriginUF)
		result.DIFALRuleID = stringPtrOrNil(strings.TrimSpace(output.Taxes.DIFAL.RuleID))
	}

	if output.Taxes.PIS != nil {
		result.PISBaseValue = floatPtr(output.Taxes.PIS.BaseValue)
		result.PISRate = floatPtr(output.Taxes.PIS.Rate)
		result.PISAmount = floatPtr(output.Taxes.PIS.Amount)
		result.PISCST = stringPtrOrNil(strings.TrimSpace(output.Taxes.PIS.CST))
		result.PISRuleID = stringPtrOrNil(strings.TrimSpace(output.Taxes.PIS.RuleID))
	}

	if output.Taxes.COFINS != nil {
		result.COFINSBaseValue = floatPtr(output.Taxes.COFINS.BaseValue)
		result.COFINSRate = floatPtr(output.Taxes.COFINS.Rate)
		result.COFINSAmount = floatPtr(output.Taxes.COFINS.Amount)
		result.COFINSCST = stringPtrOrNil(strings.TrimSpace(output.Taxes.COFINS.CST))
		result.COFINSRuleID = stringPtrOrNil(strings.TrimSpace(output.Taxes.COFINS.RuleID))
	}

	if output.Taxes.IPI != nil {
		result.IPIBaseValue = floatPtr(output.Taxes.IPI.BaseValue)
		result.IPIRate = floatPtr(output.Taxes.IPI.Rate)
		result.IPIAmount = floatPtr(output.Taxes.IPI.Amount)
		result.IPICST = stringPtrOrNil(strings.TrimSpace(output.Taxes.IPI.CST))
		result.IPIRuleID = stringPtrOrNil(strings.TrimSpace(output.Taxes.IPI.RuleID))
	}

	return result
}

func validateEvaluateInvoiceItemInput(input EvaluateInvoiceItemInput) error {
	if strings.TrimSpace(input.TenantID) == "" {
		return errors.New("invoices tax engine integration: tenant_id is required")
	}
	if strings.TrimSpace(input.OrganizationID) == "" {
		return errors.New("invoices tax engine integration: organization_id is required")
	}
	if strings.TrimSpace(input.Invoice.InvoiceID) == "" {
		return errors.New("invoices tax engine integration: invoice_id is required")
	}
	if strings.TrimSpace(input.Item.InvoiceItemID) == "" {
		return errors.New("invoices tax engine integration: invoice_item_id is required")
	}
	if strings.TrimSpace(input.Item.Description) == "" {
		return errors.New("invoices tax engine integration: item description is required")
	}

	return nil
}

func stringPtrOrNil(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func floatPtr(value float64) *float64 {
	return &value
}

func cloneOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	v := strings.TrimSpace(*value)
	if v == "" {
		return nil
	}
	return &v
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, len(values))
	copy(out, values)
	return out
}

func cloneExplanations(values []taxengine.DecisionExplanation) []taxengine.DecisionExplanation {
	if len(values) == 0 {
		return nil
	}

	out := make([]taxengine.DecisionExplanation, len(values))
	copy(out, values)
	return out
}

func cloneAuditTrail(values []taxengine.AuditStep) []taxengine.AuditStep {
	if len(values) == 0 {
		return nil
	}

	out := make([]taxengine.AuditStep, len(values))
	copy(out, values)
	return out
}