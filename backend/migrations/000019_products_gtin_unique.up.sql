WITH duplicate_products AS (
    SELECT
        id AS duplicate_id,
        FIRST_VALUE(id) OVER (
            PARTITION BY normalized_gtin
            ORDER BY created_at ASC, id ASC
        ) AS canonical_id
    FROM products
    WHERE NULLIF(TRIM(normalized_gtin), '') IS NOT NULL
),
duplicate_map AS (
    SELECT duplicate_id, canonical_id
    FROM duplicate_products
    WHERE duplicate_id <> canonical_id
)
UPDATE product_tax_profiles ptp
SET product_id = duplicate_map.canonical_id
FROM duplicate_map
WHERE ptp.product_id = duplicate_map.duplicate_id;

WITH duplicate_products AS (
    SELECT
        id AS duplicate_id,
        FIRST_VALUE(id) OVER (
            PARTITION BY normalized_gtin
            ORDER BY created_at ASC, id ASC
        ) AS canonical_id
    FROM products
    WHERE NULLIF(TRIM(normalized_gtin), '') IS NOT NULL
),
duplicate_map AS (
    SELECT duplicate_id, canonical_id
    FROM duplicate_products
    WHERE duplicate_id <> canonical_id
)
UPDATE tax_suggestions_log tsl
SET product_id = duplicate_map.canonical_id
FROM duplicate_map
WHERE tsl.product_id = duplicate_map.duplicate_id;

WITH duplicate_products AS (
    SELECT
        id AS duplicate_id,
        FIRST_VALUE(id) OVER (
            PARTITION BY normalized_gtin
            ORDER BY created_at ASC, id ASC
        ) AS canonical_id
    FROM products
    WHERE NULLIF(TRIM(normalized_gtin), '') IS NOT NULL
),
duplicate_map AS (
    SELECT duplicate_id
    FROM duplicate_products
    WHERE duplicate_id <> canonical_id
)
DELETE FROM products p
USING duplicate_map
WHERE p.id = duplicate_map.duplicate_id;

CREATE UNIQUE INDEX IF NOT EXISTS idx_products_normalized_gtin_unique
    ON products(normalized_gtin)
    WHERE NULLIF(TRIM(normalized_gtin), '') IS NOT NULL;
