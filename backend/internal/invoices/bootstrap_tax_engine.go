package invoices

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rafa/fiscal-platform/backend/internal/taxengine"
	taxenginedomain "github.com/rafa/fiscal-platform/backend/internal/taxengine/domain"
)

type TaxEngineModule struct {
	DecisionRepository *InvoiceItemTaxDecisionRepository
	IntegrationService *TaxEngineIntegrationService
	ProcessingService  *InvoiceTaxProcessingService
	QueryRepository    *InvoiceTaxProcessingQueryRepository
	Handler            *TaxEngineHandler
	Engine             taxenginedomain.Engine
}

type TaxEngineModuleDependencies struct {
	DB *pgxpool.Pool
}

func NewTaxEngineModule(deps TaxEngineModuleDependencies) (*TaxEngineModule, error) {
	if deps.DB == nil {
		return nil, errors.New("invoice tax engine module: db is required")
	}

	decisionRepo, err := NewInvoiceItemTaxDecisionRepository(deps.DB)
	if err != nil {
		return nil, err
	}

	engine, err := taxengine.New(taxengine.Dependencies{
		DB: deps.DB,
	})
	if err != nil {
		return nil, err
	}

	integrationService, err := NewTaxEngineIntegrationService(engine, decisionRepo)
	if err != nil {
		return nil, err
	}

	processingService, err := NewInvoiceTaxProcessingService(integrationService)
	if err != nil {
		return nil, err
	}

	queryRepository, err := NewInvoiceTaxProcessingQueryRepository(deps.DB)
	if err != nil {
		return nil, err
	}

	handler, err := NewTaxEngineHandler(processingService, queryRepository)
	if err != nil {
		return nil, err
	}

	return &TaxEngineModule{
		DecisionRepository: decisionRepo,
		IntegrationService: integrationService,
		ProcessingService:  processingService,
		QueryRepository:    queryRepository,
		Handler:            handler,
		Engine:             engine,
	}, nil
}