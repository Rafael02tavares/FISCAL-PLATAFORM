package integrations

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const CosmosProvider = "cosmos"
const OpenAIProvider = "openai"
const DefaultCosmosBaseURL = "https://api.cosmos.bluesoft.com.br"
const DefaultOpenAIBaseURL = "https://api.openai.com/v1"
const DefaultOpenAIModel = "gpt-5.4-mini"

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type integrationRecord struct {
	ID             string
	OrganizationID string
	Provider       string
	Enabled        bool
	BaseURL        string
	ModelName      string
	APIToken       string
	Notes          string
	UpdatedAt      string
}

func (r *Repository) Get(ctx context.Context, organizationID string, provider string) (integrationRecord, error) {
	query := `
		SELECT
			id,
			organization_id,
			provider,
			enabled,
			base_url,
			COALESCE(model_name, ''),
			api_token,
			notes,
			updated_at::text
		FROM external_integrations
		WHERE organization_id = $1 AND provider = $2
	`

	var item integrationRecord
	err := r.db.QueryRow(ctx, query, organizationID, provider).Scan(
		&item.ID,
		&item.OrganizationID,
		&item.Provider,
		&item.Enabled,
		&item.BaseURL,
		&item.ModelName,
		&item.APIToken,
		&item.Notes,
		&item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return integrationRecord{
			OrganizationID: organizationID,
			Provider:       provider,
			BaseURL:        DefaultCosmosBaseURL,
			ModelName:      defaultModelForProvider(provider),
		}, nil
	}
	if err != nil {
		return integrationRecord{}, fmt.Errorf("get integration setting: %w", err)
	}

	return item, nil
}

type UpsertParams struct {
	OrganizationID string
	Provider       string
	Enabled        bool
	BaseURL        string
	ModelName      string
	APIToken       string
	Notes          string
}

func (r *Repository) Upsert(ctx context.Context, params UpsertParams) (integrationRecord, error) {
	provider := strings.TrimSpace(params.Provider)
	baseURL := strings.TrimRight(strings.TrimSpace(params.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURLForProvider(provider)
	}
	modelName := strings.TrimSpace(params.ModelName)
	if modelName == "" {
		modelName = defaultModelForProvider(provider)
	}

	query := `
		INSERT INTO external_integrations (
			organization_id,
			provider,
			enabled,
			base_url,
			model_name,
			api_token,
			notes
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (organization_id, provider) DO UPDATE
		SET
			enabled = EXCLUDED.enabled,
			base_url = EXCLUDED.base_url,
			model_name = EXCLUDED.model_name,
			api_token = CASE
				WHEN EXCLUDED.api_token = '' THEN external_integrations.api_token
				ELSE EXCLUDED.api_token
			END,
			notes = EXCLUDED.notes,
			updated_at = NOW()
		RETURNING
			id,
			organization_id,
			provider,
			enabled,
			base_url,
			model_name,
			api_token,
			notes,
			updated_at::text
	`

	var item integrationRecord
	if err := r.db.QueryRow(
		ctx,
		query,
		params.OrganizationID,
		provider,
		params.Enabled,
		baseURL,
		modelName,
		strings.TrimSpace(params.APIToken),
		strings.TrimSpace(params.Notes),
	).Scan(
		&item.ID,
		&item.OrganizationID,
		&item.Provider,
		&item.Enabled,
		&item.BaseURL,
		&item.ModelName,
		&item.APIToken,
		&item.Notes,
		&item.UpdatedAt,
	); err != nil {
		return integrationRecord{}, fmt.Errorf("upsert integration setting: %w", err)
	}

	return item, nil
}

func defaultBaseURLForProvider(provider string) string {
	switch provider {
	case OpenAIProvider:
		return DefaultOpenAIBaseURL
	default:
		return DefaultCosmosBaseURL
	}
}

func defaultModelForProvider(provider string) string {
	switch provider {
	case OpenAIProvider:
		return DefaultOpenAIModel
	default:
		return ""
	}
}
