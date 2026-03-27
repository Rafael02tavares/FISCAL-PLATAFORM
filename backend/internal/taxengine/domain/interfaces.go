package domain

import "context"

type Engine interface {
	Evaluate(ctx context.Context, input EvaluateInput) (*EvaluateOutput, error)
}

type Normalizer interface {
	Normalize(ctx context.Context, input EvaluateInput) (*NormalizedContext, error)
}

type Classifier interface {
	Resolve(ctx context.Context, normalized NormalizedContext) (*ClassificationDecision, error)
}

type RuleEvaluator interface {
	Evaluate(ctx context.Context, normalized NormalizedContext, classification ClassificationDecision) (*RuleEvaluationResult, error)
}

type Calculator interface {
	Calculate(
		ctx context.Context,
		normalized NormalizedContext,
		classification ClassificationDecision,
		evaluation RuleEvaluationResult,
	) (*TaxDecisionSet, error)
}

type Explainer interface {
	Build(
		normalized NormalizedContext,
		classification ClassificationDecision,
		evaluation RuleEvaluationResult,
		taxes TaxDecisionSet,
	) []DecisionExplanation
}

type Auditor interface {
	RecordRun(
		ctx context.Context,
		input EvaluateInput,
		normalized NormalizedContext,
		classification ClassificationDecision,
		evaluation RuleEvaluationResult,
		taxes TaxDecisionSet,
		output EvaluateOutput,
	) error
}

type Clock interface {
	Now() string
}

type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type RuleRepository interface {
	FindCandidateRules(
		ctx context.Context,
		filter RuleFilter,
	) ([]Rule, error)
}

type LegalBasisRepository interface {
	GetLegalReferencesByIDs(
		ctx context.Context,
		ids []string,
	) (map[string]string, error)
}

type ClassificationMemoryRepository interface {
	FindByGTIN(
		ctx context.Context,
		tenantID string,
		organizationID string,
		gtin string,
	) (*ClassificationMemoryEntry, error)

	FindBySupplierProduct(
		ctx context.Context,
		tenantID string,
		organizationID string,
		supplierID string,
		supplierProductCode string,
	) (*ClassificationMemoryEntry, error)

	FindBestDescriptionMatch(
		ctx context.Context,
		tenantID string,
		organizationID string,
		descriptionNormalized string,
	) (*ClassificationMemoryEntry, error)

	Save(
		ctx context.Context,
		entry ClassificationMemoryEntry,
	) error
}

type TaxEngineRunRepository interface {
	CreateRun(
		ctx context.Context,
		run TaxEngineRun,
	) (string, error)

	CreateAuditSteps(
		ctx context.Context,
		runID string,
		steps []AuditStepRecord,
	) error
}

type RuleFilter struct {
	TaxType         TaxType
	Jurisdiction    JurisdictionType
	UF              *string
	ReferenceDate   string
	OnlyActive      bool
}

type ClassificationMemoryEntry struct {
	TenantID              string
	OrganizationID        string
	SupplierID            string
	SupplierProductCode   string
	GTIN                  string
	DescriptionNormalized string
	NCM                   string
	EXTIPI                *string
	CEST                  *string
	Confidence            float64
	Source                ClassificationSource
	LastUsedAt            string
}

type TaxEngineRun struct {
	TenantID            string
	OrganizationID      string
	InvoiceID           *string
	InvoiceItemID       *string
	InputPayload        []byte
	NormalizedPayload   []byte
	ClassificationJSON  []byte
	EvaluationJSON      []byte
	TaxesJSON           []byte
	OutputPayload       []byte
	Status              string
}

type AuditStepRecord struct {
	Order         int
	Step          string
	Status        AuditStepStatus
	Message       string
	MatchedRuleID *string
	PayloadJSON   []byte
}