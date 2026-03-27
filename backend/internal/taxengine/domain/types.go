package domain

import "time"

type DocumentType string
type OperationType string
type OperationScope string
type DecisionStatus string
type TaxType string
type RuleStatus string
type JurisdictionType string
type ClassificationSource string
type AuditStepStatus string

const (
	DocumentTypeNFe DocumentType = "NFE"
)

const (
	OperationTypeEntry OperationType = "ENTRY"
	OperationTypeExit  OperationType = "EXIT"
)

const (
	OperationScopeInternal      OperationScope = "INTERNAL"
	OperationScopeInterstate    OperationScope = "INTERSTATE"
	OperationScopeInternational OperationScope = "INTERNATIONAL"
)

const (
	DecisionStatusAutomatic         DecisionStatus = "AUTOMATIC"
	DecisionStatusSuggested         DecisionStatus = "SUGGESTED"
	DecisionStatusManualReview      DecisionStatus = "MANUAL_REVIEW_REQUIRED"
	DecisionStatusBlockedMissingData DecisionStatus = "BLOCKED_BY_MISSING_DATA"
)

const (
	TaxTypeICMS   TaxType = "ICMS"
	TaxTypeICMSST TaxType = "ICMS_ST"
	TaxTypeFCP    TaxType = "FCP"
	TaxTypeDIFAL  TaxType = "DIFAL"
	TaxTypePIS    TaxType = "PIS"
	TaxTypeCOFINS TaxType = "COFINS"
	TaxTypeIPI    TaxType = "IPI"
)

const (
	RuleStatusActive   RuleStatus = "ACTIVE"
	RuleStatusInactive RuleStatus = "INACTIVE"
	RuleStatusDraft    RuleStatus = "DRAFT"
)

const (
	JurisdictionFederal JurisdictionType = "FEDERAL"
	JurisdictionState   JurisdictionType = "STATE"
)

const (
	ClassificationSourceXML              ClassificationSource = "XML"
	ClassificationSourceGTINMemory       ClassificationSource = "GTIN_MEMORY"
	ClassificationSourceSupplierMemory   ClassificationSource = "SUPPLIER_MEMORY"
	ClassificationSourceDescriptionMatch ClassificationSource = "DESCRIPTION_MATCH"
	ClassificationSourceManualRule       ClassificationSource = "MANUAL_RULE"
	ClassificationSourceFallback         ClassificationSource = "FALLBACK"
)

const (
	AuditStepStatusSuccess AuditStepStatus = "SUCCESS"
	AuditStepStatusSkipped AuditStepStatus = "SKIPPED"
	AuditStepStatusWarning AuditStepStatus = "WARNING"
	AuditStepStatusFailed  AuditStepStatus = "FAILED"
)

type PartyContext struct {
	CNPJ                    string
	CPF                     string
	UF                      string
	CRT                     string
	IE                      string
	IsContributorICMS       bool
	IsFinalConsumer         bool
	HasSuframa              bool
	TaxRegimeCode           string
	MunicipalityCode        string
	CountryCode             string
}

type OperationContext struct {
	DocumentType            DocumentType
	OperationType           OperationType
	OperationScope          OperationScope
	CFOP                    string
	FinNFe                  string
	PresenceIndicator       string
	PurposeCode             string
	IsReturn                bool
	IsTransfer              bool
	IsResale                bool
	IsImport                bool
	IsExport                bool
	HasInterstateDelivery   bool
}

type ProductContext struct {
	ItemNumber              int
	SupplierID              string
	SupplierProductCode     string
	Description             string
	GTIN                    string
	NCM                     string
	EXTIPI                  string
	CEST                    string
	OriginCode              string
	Unit                    string
	Quantity                float64
	CommercialUnit          string
	TributaryUnit           string
	AdditionalDescription   string
}

type ValueContext struct {
	UnitValue               float64
	GrossValue              float64
	DiscountValue           float64
	FreightValue            float64
	InsuranceValue          float64
	OtherExpensesValue      float64
	IPIValue                float64
	ICMSBaseValue           float64
	TotalValue              float64
}

type MetadataContext struct {
	InvoiceNumber           string
	InvoiceSeries           string
	InvoiceKey              string
	IssueDate               time.Time
	ImportedAt              time.Time
	Source                  string
	SourceFileName          string
	UserID                  string
	RequestID               string
}

type EvaluateInput struct {
	TenantID                string
	OrganizationID          string
	InvoiceID               *string
	InvoiceItemID           *string
	Issuer                  PartyContext
	Recipient               PartyContext
	Operation               OperationContext
	Product                 ProductContext
	Values                  ValueContext
	Metadata                MetadataContext
}

type NormalizedContext struct {
	TenantID                string
	OrganizationID          string
	InvoiceID               *string
	InvoiceItemID           *string

	DocumentType            DocumentType
	OperationType           OperationType
	OperationScope          OperationScope
	CFOP                    string
	FinNFe                  string
	PresenceIndicator       string
	PurposeCode             string

	IssuerUF                string
	RecipientUF             string
	IssuerCRT               string
	RecipientContributor    bool
	FinalConsumer           bool

	ProductDescription      string
	ProductDescriptionNorm  string
	GTIN                    string
	NCM                     string
	NCMChapter              string
	NCMPosition             string
	EXTIPI                  string
	CEST                    string
	OriginCode              string

	SupplierID              string
	SupplierProductCode     string

	Quantity                float64
	UnitValue               float64
	GrossValue              float64
	DiscountValue           float64
	FreightValue            float64
	InsuranceValue          float64
	OtherExpensesValue      float64
	TotalValue              float64

	IsReturn                bool
	IsTransfer              bool
	IsResale                bool
	IsImport                bool
	IsExport                bool
	HasInterstateDelivery   bool

	AdditionalTags          []string
}

type ClassificationDecision struct {
	NCM                     string
	EXTIPI                  *string
	CEST                    *string
	Confidence              float64
	Source                  ClassificationSource
	NeedsReview             bool
	Reasons                 []string
}

type Rule struct {
	ID                      string
	Code                    string
	Name                    string
	TaxType                 TaxType
	JurisdictionType        JurisdictionType
	UF                      *string
	Priority                int
	SpecificityHint         int
	ValidFrom               time.Time
	ValidTo                 *time.Time
	Status                  RuleStatus
	Conditions              []Condition
	Actions                 []Action
	LegalBasisIDs           []string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type Condition struct {
	Field                   string
	Operator                string
	ValueText               *string
	ValueNumber             *float64
	ValueBool               *bool
	ValueList               []string
	ValueMin                *float64
	ValueMax                *float64
	Weight                  int
}

type Action struct {
	Type                    string
	Target                  string
	ValueText               *string
	ValueNumber             *float64
	ValueBool               *bool
	ValueJSON               []byte
}

type MatchedRule struct {
	RuleID                   string
	RuleCode                 string
	RuleName                 string
	TaxType                  TaxType
	Score                    int
	Priority                 int
	MatchedFacts             []string
	LegalBasisIDs            []string
}

type PreliminaryTaxDecisions struct {
	ICMS                    *ICMSDecision
	ICMSST                  *ICMSSTDecision
	FCP                     *FCPDecision
	DIFAL                   *DIFALDecision
	PIS                     *PISDecision
	COFINS                  *COFINSDecision
	IPI                     *IPIDecision
}

type RuleEvaluationResult struct {
	Decisions               PreliminaryTaxDecisions
	MatchedRules            []MatchedRule
	Warnings                []string
	AuditTrail              []AuditStep
}

type TaxDecisionSet struct {
	ICMS                    *ICMSDecision
	ICMSST                  *ICMSSTDecision
	FCP                     *FCPDecision
	DIFAL                   *DIFALDecision
	PIS                     *PISDecision
	COFINS                  *COFINSDecision
	IPI                     *IPIDecision
}

type ICMSDecision struct {
	Applies                 bool
	CST                     string
	CSOSN                   string
	ModBC                   *string
	BaseValue               float64
	Rate                    float64
	Amount                  float64
	ReducedBaseRate         *float64
	DeferredRate            *float64
	RuleID                  string
	LegalBasisIDs           []string
	Reason                  string
}

type ICMSSTDecision struct {
	Applies                 bool
	CST                     string
	ModBCST                 *string
	MVARate                 *float64
	BaseValue               float64
	Rate                    float64
	Amount                  float64
	FCPSTRate               *float64
	FCPSTAmount             *float64
	RuleID                  string
	LegalBasisIDs           []string
	Reason                  string
}

type FCPDecision struct {
	Applies                 bool
	BaseValue               float64
	Rate                    float64
	Amount                  float64
	RuleID                  string
	LegalBasisIDs           []string
	Reason                  string
}

type DIFALDecision struct {
	Applies                 bool
	BaseValue               float64
	InternalRate            float64
	InterstateRate          float64
	FCPRate                 *float64
	AmountDestinationUF     float64
	AmountOriginUF          float64
	FCPAmount               *float64
	RuleID                  string
	LegalBasisIDs           []string
	Reason                  string
}

type PISDecision struct {
	Applies                 bool
	CST                     string
	BaseValue               float64
	Rate                    float64
	Amount                  float64
	NatureCode              *string
	RuleID                  string
	LegalBasisIDs           []string
	Reason                  string
}

type COFINSDecision struct {
	Applies                 bool
	CST                     string
	BaseValue               float64
	Rate                    float64
	Amount                  float64
	NatureCode              *string
	RuleID                  string
	LegalBasisIDs           []string
	Reason                  string
}

type IPIDecision struct {
	Applies                 bool
	CST                     string
	BaseValue               float64
	Rate                    float64
	Amount                  float64
	EnquadramentoCode       *string
	RuleID                  string
	LegalBasisIDs           []string
	Reason                  string
}

type DecisionExplanation struct {
	TaxType                 TaxType
	RuleCode                string
	RuleName                string
	MatchedFacts            []string
	LegalReferences         []string
	Conclusion              string
}

type DecisionSummary struct {
	Status                  DecisionStatus
	Confidence              float64
	RequiresManualReview    bool
	HasWarnings             bool
}

type AuditStep struct {
	Order                   int
	Step                    string
	Status                  AuditStepStatus
	Message                 string
	MatchedRuleID           *string
	CreatedAt               time.Time
}

type EvaluateOutput struct {
	Classification          ClassificationDecision
	Taxes                   TaxDecisionSet
	Summary                 DecisionSummary
	Explanations            []DecisionExplanation
	Warnings                []string
	AuditTrail              []AuditStep
}