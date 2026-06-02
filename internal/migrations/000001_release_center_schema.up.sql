CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS apps (
  id UUID PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  app_key VARCHAR(64) NOT NULL,
  name VARCHAR(100) NOT NULL,
  platform VARCHAR(20) NOT NULL,
  package_name VARCHAR(255) NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, app_key, platform)
);

CREATE TABLE IF NOT EXISTS app_builds (
  id UUID PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  app_id UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
  version_name VARCHAR(64) NOT NULL,
  version_code BIGINT NOT NULL,
  build_number BIGINT NOT NULL,
  channel VARCHAR(32) NOT NULL,
  build_type VARCHAR(32) NOT NULL DEFAULT 'debug',
  git_commit VARCHAR(64) NOT NULL DEFAULT '',
  git_branch VARCHAR(128) NOT NULL DEFAULT '',
  build_environment VARCHAR(32) NOT NULL DEFAULT '',
  build_status VARCHAR(32) NOT NULL DEFAULT 'success',
  artifact_type VARCHAR(32) NOT NULL DEFAULT 'apk',
  artifact_path TEXT NOT NULL DEFAULT '',
  artifact_size BIGINT NOT NULL DEFAULT 0,
  sha256 VARCHAR(64) NOT NULL DEFAULT '',
  built_by VARCHAR(128) NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (app_id, build_number)
);

CREATE TABLE IF NOT EXISTS app_releases (
  id UUID PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  app_id UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
  build_id UUID NOT NULL REFERENCES app_builds(id) ON DELETE CASCADE,
  channel VARCHAR(32) NOT NULL,
  build_type VARCHAR(32) NOT NULL DEFAULT 'debug',
  status VARCHAR(32) NOT NULL DEFAULT 'draft',
  title VARCHAR(255) NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  release_notes_markdown TEXT NOT NULL DEFAULT '',
  upgrade_message TEXT NOT NULL DEFAULT '',
  update_level VARCHAR(32) NOT NULL DEFAULT 'normal',
  rollout_percentage INT NOT NULL DEFAULT 100,
  min_supported_code BIGINT NOT NULL DEFAULT 0,
  block_old_versions BOOLEAN NOT NULL DEFAULT FALSE,
  current_version_available BOOLEAN NOT NULL DEFAULT TRUE,
  published_at TIMESTAMPTZ,
  paused_at TIMESTAMPTZ,
  created_by VARCHAR(128) NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (app_id, build_id, channel, build_type)
);

CREATE TABLE IF NOT EXISTS app_release_notes (
  id UUID PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  release_id UUID NOT NULL REFERENCES app_releases(id) ON DELETE CASCADE,
  language VARCHAR(32) NOT NULL,
  title VARCHAR(255) NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  content_markdown TEXT NOT NULL DEFAULT '',
  edited_by VARCHAR(128) NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (release_id, language)
);

CREATE TABLE IF NOT EXISTS app_release_rules (
  id UUID PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  release_id UUID NOT NULL REFERENCES app_releases(id) ON DELETE CASCADE,
  rule_type VARCHAR(32) NOT NULL,
  rule_value TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS app_resource_versions (
  id UUID PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  app_id UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
  resource_version VARCHAR(64) NOT NULL,
  channel VARCHAR(32) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'draft',
  min_app_version_code BIGINT NOT NULL DEFAULT 0,
  max_app_version_code BIGINT NOT NULL DEFAULT 0,
  update_level VARCHAR(32) NOT NULL DEFAULT 'normal',
  title VARCHAR(255) NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  release_notes_markdown TEXT NOT NULL DEFAULT '',
  rollout_percentage INT NOT NULL DEFAULT 100,
  manifest_url TEXT NOT NULL DEFAULT '',
  total_size BIGINT NOT NULL DEFAULT 0,
  published_at TIMESTAMPTZ,
  paused_at TIMESTAMPTZ,
  created_by VARCHAR(128) NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (app_id, resource_version, channel)
);

CREATE TABLE IF NOT EXISTS app_resource_packages (
  id UUID PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  resource_version_id UUID NOT NULL REFERENCES app_resource_versions(id) ON DELETE CASCADE,
  package_key VARCHAR(128) NOT NULL,
  package_type VARCHAR(32) NOT NULL DEFAULT 'zip',
  file_url TEXT NOT NULL DEFAULT '',
  file_size BIGINT NOT NULL DEFAULT 0,
  sha256 VARCHAR(64) NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (resource_version_id, package_key)
);

CREATE TABLE IF NOT EXISTS app_installations (
  id UUID PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  app_id UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL DEFAULT '',
  device_id TEXT NOT NULL,
  device_key TEXT NOT NULL DEFAULT '',
  device_name TEXT NOT NULL DEFAULT '',
  platform VARCHAR(20) NOT NULL DEFAULT 'android',
  os_version VARCHAR(32) NOT NULL DEFAULT '',
  device_model VARCHAR(128) NOT NULL DEFAULT '',
  installed_version VARCHAR(64) NOT NULL DEFAULT '',
  installed_code BIGINT NOT NULL DEFAULT 0,
  build_number BIGINT NOT NULL DEFAULT 0,
  resource_version VARCHAR(64) NOT NULL DEFAULT '',
  last_seen_at TIMESTAMPTZ,
  last_upgrade_at TIMESTAMPTZ,
  last_upgrade_status VARCHAR(32) NOT NULL DEFAULT '',
  last_error TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (app_id, device_id)
);

CREATE TABLE IF NOT EXISTS app_upgrade_events (
  id UUID PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  app_id UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
  release_id UUID REFERENCES app_releases(id) ON DELETE SET NULL,
  user_id TEXT NOT NULL DEFAULT '',
  device_id TEXT NOT NULL,
  from_version VARCHAR(64) NOT NULL DEFAULT '',
  to_version VARCHAR(64) NOT NULL DEFAULT '',
  from_version_code BIGINT NOT NULL DEFAULT 0,
  to_version_code BIGINT NOT NULL DEFAULT 0,
  event_type VARCHAR(32) NOT NULL,
  error_message TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS app_resource_update_events (
  id UUID PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  app_id UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL DEFAULT '',
  device_id TEXT NOT NULL,
  from_version VARCHAR(64) NOT NULL DEFAULT '',
  to_version VARCHAR(64) NOT NULL DEFAULT '',
  event_type VARCHAR(32) NOT NULL,
  package_key VARCHAR(128) NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS webhook_events (
  id UUID PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  provider VARCHAR(32) NOT NULL,
  event_type VARCHAR(64) NOT NULL,
  delivery_id TEXT NOT NULL DEFAULT '',
  repository TEXT NOT NULL DEFAULT '',
  ref TEXT NOT NULL DEFAULT '',
  commit_sha VARCHAR(64) NOT NULL DEFAULT '',
  sender TEXT NOT NULL DEFAULT '',
  action TEXT NOT NULL DEFAULT '',
  workflow TEXT NOT NULL DEFAULT '',
  run_id TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_events (
  id BIGSERIAL PRIMARY KEY,
  event_id UUID NOT NULL DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL DEFAULT 'default',
  actor_type VARCHAR(32) NOT NULL DEFAULT '',
  actor_id TEXT NOT NULL DEFAULT '',
  action VARCHAR(128) NOT NULL,
  target_type VARCHAR(64) NOT NULL DEFAULT '',
  target_id TEXT NOT NULL DEFAULT '',
  message_zh TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  event_time TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_app_releases_lookup ON app_releases (tenant_id, channel, build_type, status, published_at);
CREATE INDEX IF NOT EXISTS idx_app_builds_version ON app_builds (app_id, version_code DESC, build_number DESC);
CREATE INDEX IF NOT EXISTS idx_app_release_rules_lookup ON app_release_rules (tenant_id, release_id, rule_type, rule_value);
CREATE INDEX IF NOT EXISTS idx_app_resource_versions_lookup ON app_resource_versions (tenant_id, channel, status, published_at);
CREATE INDEX IF NOT EXISTS idx_app_installations_seen ON app_installations (tenant_id, last_seen_at DESC);
CREATE INDEX IF NOT EXISTS idx_app_upgrade_events_created ON app_upgrade_events (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_app_resource_events_created ON app_resource_update_events (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_webhook_events_created ON webhook_events (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_time ON audit_events (tenant_id, event_time DESC, id DESC);
