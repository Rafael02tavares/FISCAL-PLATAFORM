CREATE TABLE IF NOT EXISTS fiscal_watcher_sources (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code VARCHAR(80) NOT NULL UNIQUE,
  name VARCHAR(160) NOT NULL,
  authority VARCHAR(120) NOT NULL,
  source_type VARCHAR(40) NOT NULL,
  url TEXT NOT NULL,
  cadence_hours INTEGER NOT NULL DEFAULT 24,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  last_checked_at TIMESTAMPTZ,
  last_status VARCHAR(40) NOT NULL DEFAULT 'idle',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS fiscal_watcher_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source_id UUID NOT NULL REFERENCES fiscal_watcher_sources(id) ON DELETE CASCADE,
  status VARCHAR(40) NOT NULL DEFAULT 'review_required',
  severity VARCHAR(20) NOT NULL DEFAULT 'medium',
  detection_mode VARCHAR(20) NOT NULL DEFAULT 'manual',
  title VARCHAR(220) NOT NULL,
  summary TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fiscal_watcher_sources_active
  ON fiscal_watcher_sources(active, code);

CREATE INDEX IF NOT EXISTS idx_fiscal_watcher_events_status_detected
  ON fiscal_watcher_events(status, detected_at DESC);

INSERT INTO fiscal_watcher_sources (code, name, authority, source_type, url, cadence_hours)
VALUES
  ('planalto_lc87', 'Lei Kandir', 'Planalto', 'legal', 'https://www.planalto.gov.br/ccivil_03/Leis/lcp/Lcp87.htm', 24),
  ('confaz_cest', 'Tabela CEST', 'CONFAZ', 'catalog', 'https://www.confaz.fazenda.gov.br/', 24),
  ('portal_nfe', 'Portal NF-e', 'Portal NF-e', 'operational', 'https://www.nfe.fazenda.gov.br/portal/principal.aspx', 12)
ON CONFLICT (code) DO UPDATE
SET
  name = EXCLUDED.name,
  authority = EXCLUDED.authority,
  source_type = EXCLUDED.source_type,
  url = EXCLUDED.url,
  cadence_hours = EXCLUDED.cadence_hours,
  active = TRUE,
  updated_at = NOW();
