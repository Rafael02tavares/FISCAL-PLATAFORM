DROP INDEX IF EXISTS idx_import_batches_imported_at;
DROP INDEX IF EXISTS idx_import_batches_source_name;

DROP INDEX IF EXISTS idx_ncm_fiscal_enrichment_ncm_id;

DROP INDEX IF EXISTS idx_ncm_catalog_end_date;
DROP INDEX IF EXISTS idx_ncm_catalog_start_date;
DROP INDEX IF EXISTS idx_ncm_catalog_heading_code;
DROP INDEX IF EXISTS idx_ncm_catalog_chapter_code;
DROP INDEX IF EXISTS idx_ncm_catalog_parent_code;
DROP INDEX IF EXISTS idx_ncm_catalog_code;
DROP INDEX IF EXISTS idx_ncm_catalog_code_ex_active;

DROP TABLE IF EXISTS ncm_fiscal_enrichment;
DROP TABLE IF EXISTS ncm_catalog;
DROP TABLE IF EXISTS import_batches;