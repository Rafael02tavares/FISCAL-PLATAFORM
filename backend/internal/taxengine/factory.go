package taxengine

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rafa/fiscal-platform/backend/internal/taxengine/audit"
	"github.com/rafa/fiscal-platform/backend/internal/taxengine/calculators"
	"github.com/rafa/fiscal-platform/backend/internal/taxengine/classifier"
	"github.com/rafa/fiscal-platform/backend/internal/taxengine/domain"
	"github.com/rafa/fiscal-platform/backend/internal/taxengine/explain"
	"github.com/rafa/fiscal-platform/backend/internal/taxengine/normalizer"
	"github.com/rafa/fiscal-platform/backend/internal/taxengine/repository/postgres"
	"github.com/rafa/fiscal-platform/backend/internal/taxengine/rules"
)

type Dependencies struct {
	DB *pgxpool.Pool
}

func New(deps Dependencies) (domain.Engine, error) {
	if deps.DB == nil {
		return nil, errors.New("taxengine: db is required")
	}

	ruleRepo, err := postgres.NewRuleRepository(deps.DB)
	if err != nil {
		return nil, err
	}

	classificationMemoryRepo, err := postgres.NewClassificationMemoryRepository(deps.DB)
	if err != nil {
		return nil, err
	}

	taxEngineRunRepo, err := postgres.NewTaxEngineRunRepository(deps.DB)
	if err != nil {
		return nil, err
	}

	normalizerService := normalizer.NewService()

	classifierService, err := classifier.NewService(classificationMemoryRepo)
	if err != nil {
		return nil, err
	}

	matcher := rules.NewMatcher()
	conflictResolver := rules.NewConflictResolver()

	evaluator, err := rules.NewEvaluator(
		ruleRepo,
		matcher,
		conflictResolver,
		nil,
	)
	if err != nil {
		return nil, err
	}

	calculatorService := calculators.NewService()
	explainerService := explain.NewBuilder(nil)

	auditorService, err := audit.NewService(taxEngineRunRepo)
	if err != nil {
		return nil, err
	}

	engine, err := NewService(
		normalizerService,
		classifierService,
		evaluator,
		calculatorService,
		explainerService,
		auditorService,
	)
	if err != nil {
		return nil, err
	}

	return engine, nil
}