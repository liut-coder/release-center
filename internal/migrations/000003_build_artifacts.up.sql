CREATE TABLE IF NOT EXISTS app_build_artifacts (
  id UUID PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  build_id UUID NOT NULL REFERENCES app_builds(id) ON DELETE CASCADE,
  name VARCHAR(128) NOT NULL,
  artifact_type VARCHAR(32) NOT NULL DEFAULT 'artifact',
  artifact_path TEXT NOT NULL DEFAULT '',
  file_name TEXT NOT NULL DEFAULT '',
  artifact_size BIGINT NOT NULL DEFAULT 0,
  sha256 VARCHAR(64) NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (build_id, name)
);

INSERT INTO app_build_artifacts (
  id, tenant_id, build_id, name, artifact_type, artifact_path,
  file_name, artifact_size, sha256, metadata, created_at, updated_at
)
SELECT
  gen_random_uuid(),
  b.tenant_id,
  b.id,
  coalesce(nullif(b.metadata->>'artifact_name', ''), nullif(b.artifact_type, ''), 'primary'),
  coalesce(nullif(b.artifact_type, ''), 'artifact'),
  b.artifact_path,
  coalesce(nullif(b.metadata->>'file_name', ''), b.version_name || '-' || b.build_number::text || '.apk'),
  b.artifact_size,
  b.sha256,
  jsonb_strip_nulls(jsonb_build_object(
    'storage_key', nullif(b.metadata->>'storage_key', ''),
    'source', 'app_builds_backfill'
  )),
  b.created_at,
  b.created_at
FROM app_builds b
WHERE b.tenant_id = 'default'
  AND (
    b.artifact_path <> ''
    OR b.artifact_size > 0
    OR b.sha256 <> ''
    OR coalesce(b.metadata->>'storage_key', '') <> ''
  )
ON CONFLICT (build_id, name) DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_app_build_artifacts_build ON app_build_artifacts (tenant_id, build_id, created_at);
CREATE INDEX IF NOT EXISTS idx_app_build_artifacts_type ON app_build_artifacts (tenant_id, artifact_type, created_at DESC);
