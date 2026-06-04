CREATE TABLE IF NOT EXISTS release_environments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL DEFAULT 'default',
  environment_key VARCHAR(32) NOT NULL,
  name VARCHAR(64) NOT NULL,
  sort_order INT NOT NULL DEFAULT 100,
  requires_approval BOOLEAN NOT NULL DEFAULT FALSE,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, environment_key)
);

CREATE TABLE IF NOT EXISTS release_units (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL DEFAULT 'default',
  project_id UUID NOT NULL REFERENCES release_projects(id) ON DELETE CASCADE,
  app_id UUID REFERENCES apps(id) ON DELETE SET NULL,
  unit_key VARCHAR(96) NOT NULL,
  name VARCHAR(128) NOT NULL,
  unit_type VARCHAR(32) NOT NULL,
  default_channel VARCHAR(32) NOT NULL DEFAULT 'dev',
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (project_id, unit_key)
);

CREATE TABLE IF NOT EXISTS release_plans (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL DEFAULT 'default',
  project_id UUID NOT NULL REFERENCES release_projects(id) ON DELETE CASCADE,
  release_unit_id UUID NOT NULL REFERENCES release_units(id) ON DELETE CASCADE,
  environment_id UUID NOT NULL REFERENCES release_environments(id) ON DELETE RESTRICT,
  plan_key VARCHAR(128) NOT NULL,
  title VARCHAR(255) NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  version_name VARCHAR(64) NOT NULL DEFAULT '',
  build_number BIGINT NOT NULL DEFAULT 0,
  git_commit VARCHAR(64) NOT NULL DEFAULT '',
  channel VARCHAR(32) NOT NULL DEFAULT 'dev',
  status VARCHAR(32) NOT NULL DEFAULT 'draft',
  rollout_percentage INT NOT NULL DEFAULT 100,
  target_type VARCHAR(32) NOT NULL DEFAULT 'all',
  target_value TEXT NOT NULL DEFAULT '',
  scheduled_at TIMESTAMPTZ,
  published_at TIMESTAMPTZ,
  paused_at TIMESTAMPTZ,
  approved_by VARCHAR(128) NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_by VARCHAR(128) NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (project_id, plan_key)
);

CREATE TABLE IF NOT EXISTS release_plan_artifacts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL DEFAULT 'default',
  release_plan_id UUID NOT NULL REFERENCES release_plans(id) ON DELETE CASCADE,
  build_run_id UUID REFERENCES build_center_runs(id) ON DELETE SET NULL,
  app_build_id UUID REFERENCES app_builds(id) ON DELETE SET NULL,
  app_build_artifact_id UUID REFERENCES app_build_artifacts(id) ON DELETE SET NULL,
  artifact_name VARCHAR(128) NOT NULL,
  artifact_type VARCHAR(32) NOT NULL DEFAULT 'artifact',
  file_name TEXT NOT NULL DEFAULT '',
  immutable_ref TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (release_plan_id, artifact_name)
);

CREATE INDEX IF NOT EXISTS idx_release_units_project ON release_units (tenant_id, project_id, unit_type, enabled);
CREATE INDEX IF NOT EXISTS idx_release_plans_status ON release_plans (tenant_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_release_plans_unit_env ON release_plans (tenant_id, release_unit_id, environment_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_release_plan_artifacts_plan ON release_plan_artifacts (tenant_id, release_plan_id, artifact_type);

INSERT INTO release_environments (tenant_id, environment_key, name, sort_order, requires_approval, metadata)
VALUES
  ('default', 'dev', '开发环境', 10, FALSE, jsonb_build_object('source', 'migration_000007')),
  ('default', 'test', '测试环境', 20, FALSE, jsonb_build_object('source', 'migration_000007')),
  ('default', 'staging', '预发环境', 30, TRUE, jsonb_build_object('source', 'migration_000007')),
  ('default', 'prod', '生产环境', 40, TRUE, jsonb_build_object('source', 'migration_000007'))
ON CONFLICT (tenant_id, environment_key) DO NOTHING;

INSERT INTO system_permissions (id, tenant_id, name, code, module, permission_type)
VALUES
  (gen_random_uuid(), 'default', '查看发布计划', 'release:plan:read', '发布中心', 'api'),
  (gen_random_uuid(), 'default', '维护发布计划', 'release:plan:write', '发布中心', 'button')
ON CONFLICT (tenant_id, code) DO NOTHING;

INSERT INTO system_dictionaries (id, tenant_id, group_key, item_key, label, item_value, sort_order)
VALUES
  (gen_random_uuid(), 'default', 'release_unit_type', 'android', 'Android', 'android', 10),
  (gen_random_uuid(), 'default', 'release_unit_type', 'web', 'Web', 'web', 20),
  (gen_random_uuid(), 'default', 'release_unit_type', 'docs', 'Docs', 'docs', 30),
  (gen_random_uuid(), 'default', 'release_unit_type', 'worker', 'Worker', 'worker', 40),
  (gen_random_uuid(), 'default', 'release_unit_type', 'server', 'Server', 'server', 50),
  (gen_random_uuid(), 'default', 'release_unit_type', 'docker', 'Docker', 'docker', 60),
  (gen_random_uuid(), 'default', 'release_unit_type', 'config', 'Config', 'config', 70)
ON CONFLICT (tenant_id, group_key, item_key) DO NOTHING;

WITH project_row AS (
  SELECT id FROM release_projects WHERE tenant_id = 'default' AND project_key = 'release-center'
)
INSERT INTO release_units (
  tenant_id, project_id, unit_key, name, unit_type, default_channel, enabled, metadata
)
SELECT 'default', project_row.id, unit_key, name, unit_type, 'dev', TRUE,
       jsonb_build_object('source', 'migration_000007')
FROM project_row,
     (VALUES
       ('admin-web', 'Release Center Admin Web', 'web'),
       ('api-server', 'Release Center API Server', 'server'),
       ('api-worker', 'Release Center Edge Worker', 'worker')
     ) AS seed(unit_key, name, unit_type)
ON CONFLICT (project_id, unit_key) DO NOTHING;
