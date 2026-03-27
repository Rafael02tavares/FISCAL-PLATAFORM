package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rafa/fiscal-platform/backend/internal/taxengine/domain"
)

type ClassificationMemoryRepository struct {
	db *pgxpool.Pool
}

func NewClassificationMemoryRepository(db *pgxpool.Pool) (*ClassificationMemoryRepository, error) {
	if db == nil {
		return nil, errors.New("taxengine postgres: db pool is required")
	}

	return &ClassificationMemoryRepository{db: db}, nil
}

func (r *ClassificationMemoryRepository) FindByGTIN(
	ctx context.Context,
	tenantID string,
	organizationID string,
	gtin string,
) (*domain.ClassificationMemoryEntry, error) {
	tenantID = strings.TrimSpace(tenantID)
	organizationID = strings.TrimSpace(organizationID)
	gtin = strings.TrimSpace(gtin)

	if tenantID == "" || organizationID == "" || gtin == "" {
		return nil, nil
	}

	query := `
		SELECT
			tenant_id,
			organization_id,
			supplier_id,
			supplier_product_code,
			gtin,
			description_normalized,
			ncm,
			extipi,
			cest,
			confidence,
			source,
			last_used_at
		FROM product_classification_memory
		WHERE tenant_id = $1
		  AND organization_id = $2
		  AND gtin = $3
		ORDER BY confidence DESC, last_used_at DESC
		LIMIT 1
	`

	entry, err := r.scanSingleEntry(ctx, query, tenantID, organizationID, gtin)
	if err != nil {
		return nil, fmt.Errorf("taxengine postgres: find classification memory by gtin: %w", err)
	}

	return entry, nil
}

func (r *ClassificationMemoryRepository) FindBySupplierProduct(
	ctx context.Context,
	tenantID string,
	organizationID string,
	supplierID string,
	supplierProductCode string,
) (*domain.ClassificationMemoryEntry, error) {
	tenantID = strings.TrimSpace(tenantID)
	organizationID = strings.TrimSpace(organizationID)
	supplierID = strings.TrimSpace(supplierID)
	supplierProductCode = strings.TrimSpace(supplierProductCode)

	if tenantID == "" || organizationID == "" || supplierID == "" || supplierProductCode == "" {
		return nil, nil
	}

	query := `
		SELECT
			tenant_id,
			organization_id,
			supplier_id,
			supplier_product_code,
			gtin,
			description_normalized,
			ncm,
			extipi,
			cest,
			confidence,
			source,
			last_used_at
		FROM product_classification_memory
		WHERE tenant_id = $1
		  AND organization_id = $2
		  AND supplier_id = $3
		  AND supplier_product_code = $4
		ORDER BY confidence DESC, last_used_at DESC
		LIMIT 1
	`

	entry, err := r.scanSingleEntry(ctx, query, tenantID, organizationID, supplierID, supplierProductCode)
	if err != nil {
		return nil, fmt.Errorf("taxengine postgres: find classification memory by supplier product: %w", err)
	}

	return entry, nil
}

func (r *ClassificationMemoryRepository) FindBestDescriptionMatch(
	ctx context.Context,
	tenantID string,
	organizationID string,
	descriptionNormalized string,
) (*domain.ClassificationMemoryEntry, error) {
	tenantID = strings.TrimSpace(tenantID)
	organizationID = strings.TrimSpace(organizationID)
	descriptionNormalized = normalizeSpaces(strings.TrimSpace(descriptionNormalized))

	if tenantID == "" || organizationID == "" || descriptionNormalized == "" {
		return nil, nil
	}

	query := `
		SELECT
			tenant_id,
			organization_id,
			supplier_id,
			supplier_product_code,
			gtin,
			description_normalized,
			ncm,
			extipi,
			cest,
			confidence,
			source,
			last_used_at
		FROM product_classification_memory
		WHERE tenant_id = $1
		  AND organization_id = $2
		  AND description_normalized = $3
		ORDER BY confidence DESC, last_used_at DESC
		LIMIT 1
	`

	entry, err := r.scanSingleEntry(ctx, query, tenantID, organizationID, descriptionNormalized)
	if err != nil {
		return nil, fmt.Errorf("taxengine postgres: find classification memory by description: %w", err)
	}

	return entry, nil
}

func (r *ClassificationMemoryRepository) Save(
	ctx context.Context,
	entry domain.ClassificationMemoryEntry,
) error {
	entry = sanitizeClassificationMemoryEntry(entry)

	if entry.TenantID == "" {
		return errors.New("taxengine postgres: classification memory tenant_id is required")
	}
	if entry.OrganizationID == "" {
		return errors.New("taxengine postgres: classification memory organization_id is required")
	}
	if entry.NCM == "" {
		return errors.New("taxengine postgres: classification memory ncm is required")
	}
	if entry.GTIN == "" && (entry.SupplierID == "" || entry.SupplierProductCode == "") && entry.DescriptionNormalized == "" {
		return errors.New("taxengine postgres: at least one key must be provided for classification memory")
	}

	query := `
		INSERT INTO product_classification_memory (
			tenant_id,
			organization_id,
			supplier_id,
			supplier_product_code,
			gtin,
			description_normalized,
			ncm,
			extipi,
			cest,
			confidence,
			source,
			last_used_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW()
		)
		ON CONFLICT (tenant_id, organization_id, supplier_id, supplier_product_code, gtin, description_normalized)
		DO UPDATE SET
			ncm = EXCLUDED.ncm,
			extipi = EXCLUDED.extipi,
			cest = EXCLUDED.cest,
			confidence = EXCLUDED.confidence,
			source = EXCLUDED.source,
			last_used_at = NOW()
	`

	_, err := r.db.Exec(
		ctx,
		query,
		nullIfEmpty(entry.TenantID),
		nullIfEmpty(entry.OrganizationID),
		nullIfEmpty(entry.SupplierID),
		nullIfEmpty(entry.SupplierProductCode),
		nullIfEmpty(entry.GTIN),
		nullIfEmpty(entry.DescriptionNormalized),
		nullIfEmpty(entry.NCM),
		entry.EXTIPI,
		entry.CEST,
		clampConfidence(entry.Confidence),
		string(entry.Source),
	)
	if err != nil {
		return fmt.Errorf("taxengine postgres: save classification memory: %w", err)
	}

	return nil
}

func (r *ClassificationMemoryRepository) scanSingleEntry(
	ctx context.Context,
	query string,
	args ...any,
) (*domain.ClassificationMemoryEntry, error) {
	row := r.db.QueryRow(ctx, query, args...)

	var entry domain.ClassificationMemoryEntry
	var extipi *string
	var cest *string
	var source string
	var lastUsedAt time.Time

	err := row.Scan(
		&entry.TenantID,
		&entry.OrganizationID,
		&entry.SupplierID,
		&entry.SupplierProductCode,
		&entry.GTIN,
		&entry.DescriptionNormalized,
		&entry.NCM,
		&extipi,
		&cest,
		&entry.Confidence,
		&source,
		&lastUsedAt,
	)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, err
	}

	entry.TenantID = strings.TrimSpace(entry.TenantID)
	entry.OrganizationID = strings.TrimSpace(entry.OrganizationID)
	entry.SupplierID = strings.TrimSpace(entry.SupplierID)
	entry.SupplierProductCode = strings.TrimSpace(entry.SupplierProductCode)
	entry.GTIN = strings.TrimSpace(entry.GTIN)
	entry.DescriptionNormalized = normalizeSpaces(strings.TrimSpace(entry.DescriptionNormalized))
	entry.NCM = strings.TrimSpace(entry.NCM)
	entry.EXTIPI = normalizeOptionalTrim(extipi)
	entry.CEST = normalizeOptionalTrim(cest)
	entry.Confidence = clampConfidence(entry.Confidence)
	entry.Source = domain.ClassificationSource(strings.TrimSpace(source))
	entry.LastUsedAt = lastUsedAt.Format(time.RFC3339)

	return &entry, nil
}

func sanitizeClassificationMemoryEntry(entry domain.ClassificationMemoryEntry) domain.ClassificationMemoryEntry {
	entry.TenantID = strings.TrimSpace(entry.TenantID)
	entry.OrganizationID = strings.TrimSpace(entry.OrganizationID)
	entry.SupplierID = strings.TrimSpace(entry.SupplierID)
	entry.SupplierProductCode = strings.TrimSpace(entry.SupplierProductCode)
	entry.GTIN = strings.TrimSpace(entry.GTIN)
	entry.DescriptionNormalized = normalizeSpaces(strings.TrimSpace(entry.DescriptionNormalized))
	entry.NCM = strings.TrimSpace(entry.NCM)
	entry.EXTIPI = normalizeOptionalTrim(entry.EXTIPI)
	entry.CEST = normalizeOptionalTrim(entry.CEST)
	entry.Confidence = clampConfidence(entry.Confidence)

	if strings.TrimSpace(string(entry.Source)) == "" {
		entry.Source = domain.ClassificationSourceFallback
	}

	return entry
}

func nullIfEmpty(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func normalizeSpaces(value string) string {
	if value == "" {
		return ""
	}
	return strings.Join(strings.Fields(value), " ")
}

func clampConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func isNoRows(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no rows in result set")
}

var _ domain.ClassificationMemoryRepository = (*ClassificationMemoryRepository)(nil)