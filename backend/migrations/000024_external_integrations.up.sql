CREATE TABLE IF NOT EXISTS external_integrations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  provider VARCHAR(40) NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT FALSE,
  base_url TEXT NOT NULL,
  api_token TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (organization_id, provider)
);

CREATE INDEX IF NOT EXISTS idx_external_integrations_org_provider
  ON external_integrations (organization_id, provider);
