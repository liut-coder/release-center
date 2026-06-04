CREATE TABLE IF NOT EXISTS release_projects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL DEFAULT 'default',
  project_key VARCHAR(96) NOT NULL,
  name VARCHAR(128) NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  owner_account VARCHAR(128) NOT NULL DEFAULT '',
  lifecycle_status VARCHAR(32) NOT NULL DEFAULT 'active',
  default_channel VARCHAR(32) NOT NULL DEFAULT 'dev',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, project_key)
);

CREATE TABLE IF NOT EXISTS code_repositories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL DEFAULT 'default',
  project_id UUID NOT NULL REFERENCES release_projects(id) ON DELETE CASCADE,
  provider VARCHAR(32) NOT NULL,
  repo_url TEXT NOT NULL,
  repo_full_name TEXT NOT NULL DEFAULT '',
  default_ref VARCHAR(128) NOT NULL DEFAULT 'main',
  credential_ref TEXT NOT NULL DEFAULT '',
  webhook_secret_ref TEXT NOT NULL DEFAULT '',
  webhook_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  trigger_on_push BOOLEAN NOT NULL DEFAULT FALSE,
  trigger_on_tag BOOLEAN NOT NULL DEFAULT FALSE,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, provider, repo_url)
);

CREATE TABLE IF NOT EXISTS build_profiles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL DEFAULT 'default',
  project_id UUID NOT NULL REFERENCES release_projects(id) ON DELETE CASCADE,
  app_id UUID REFERENCES apps(id) ON DELETE SET NULL,
  profile_key VARCHAR(96) NOT NULL,
  name VARCHAR(128) NOT NULL,
  build_center_project VARCHAR(96) NOT NULL,
  stack_type VARCHAR(64) NOT NULL DEFAULT 'generic',
  build_type VARCHAR(32) NOT NULL DEFAULT 'release',
  config_path TEXT NOT NULL DEFAULT '',
  source_workdir TEXT NOT NULL DEFAULT '',
  default_ref VARCHAR(128) NOT NULL DEFAULT 'main',
  default_version_name VARCHAR(64) NOT NULL DEFAULT '',
  default_version_code BIGINT NOT NULL DEFAULT 0,
  default_channel VARCHAR(32) NOT NULL DEFAULT 'dev',
  build_action VARCHAR(32) NOT NULL DEFAULT 'all',
  commands JSONB NOT NULL DEFAULT '{}'::jsonb,
  artifact_rules JSONB NOT NULL DEFAULT '[]'::jsonb,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (project_id, profile_key)
);

CREATE TABLE IF NOT EXISTS build_center_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL DEFAULT 'default',
  project_id UUID NOT NULL REFERENCES release_projects(id) ON DELETE CASCADE,
  build_profile_id UUID REFERENCES build_profiles(id) ON DELETE SET NULL,
  app_build_id UUID REFERENCES app_builds(id) ON DELETE SET NULL,
  trigger_type VARCHAR(32) NOT NULL DEFAULT 'manual',
  trigger_source TEXT NOT NULL DEFAULT '',
  action VARCHAR(32) NOT NULL DEFAULT 'all',
  git_ref VARCHAR(128) NOT NULL DEFAULT '',
  git_commit VARCHAR(64) NOT NULL DEFAULT '',
  version_name VARCHAR(64) NOT NULL DEFAULT '',
  version_code BIGINT NOT NULL DEFAULT 0,
  build_number BIGINT NOT NULL DEFAULT 0,
  channel VARCHAR(32) NOT NULL DEFAULT 'dev',
  status VARCHAR(32) NOT NULL DEFAULT 'queued',
  exit_code INT,
  started_by VARCHAR(128) NOT NULL DEFAULT '',
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  duration_ms BIGINT NOT NULL DEFAULT 0,
  workspace_dir TEXT NOT NULL DEFAULT '',
  artifact_dir TEXT NOT NULL DEFAULT '',
  log_dir TEXT NOT NULL DEFAULT '',
  manifest_path TEXT NOT NULL DEFAULT '',
  upload_status VARCHAR(32) NOT NULL DEFAULT 'pending',
  error_message TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS build_center_run_artifacts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL DEFAULT 'default',
  run_id UUID NOT NULL REFERENCES build_center_runs(id) ON DELETE CASCADE,
  app_build_artifact_id UUID REFERENCES app_build_artifacts(id) ON DELETE SET NULL,
  name VARCHAR(128) NOT NULL,
  artifact_type VARCHAR(32) NOT NULL DEFAULT 'artifact',
  file_name TEXT NOT NULL DEFAULT '',
  local_path TEXT NOT NULL DEFAULT '',
  size_bytes BIGINT NOT NULL DEFAULT 0,
  sha256 VARCHAR(64) NOT NULL DEFAULT '',
  upload_status VARCHAR(32) NOT NULL DEFAULT 'pending',
  download_url TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (run_id, name)
);

CREATE TABLE IF NOT EXISTS deployment_targets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL DEFAULT 'default',
  project_id UUID NOT NULL REFERENCES release_projects(id) ON DELETE CASCADE,
  app_id UUID REFERENCES apps(id) ON DELETE SET NULL,
  target_key VARCHAR(96) NOT NULL,
  name VARCHAR(128) NOT NULL,
  provider VARCHAR(32) NOT NULL,
  environment VARCHAR(32) NOT NULL DEFAULT 'prod',
  endpoint_url TEXT NOT NULL DEFAULT '',
  cloudflare_account_id TEXT NOT NULL DEFAULT '',
  cloudflare_project_name TEXT NOT NULL DEFAULT '',
  cloudflare_script_name TEXT NOT NULL DEFAULT '',
  cloudflare_bucket_name TEXT NOT NULL DEFAULT '',
  credential_ref TEXT NOT NULL DEFAULT '',
  enabled BOOLEAN NOT NULL DEFAULT FALSE,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (project_id, target_key)
);

CREATE TABLE IF NOT EXISTS deployment_records (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL DEFAULT 'default',
  target_id UUID NOT NULL REFERENCES deployment_targets(id) ON DELETE CASCADE,
  run_id UUID REFERENCES build_center_runs(id) ON DELETE SET NULL,
  app_build_id UUID REFERENCES app_builds(id) ON DELETE SET NULL,
  app_build_artifact_id UUID REFERENCES app_build_artifacts(id) ON DELETE SET NULL,
  provider VARCHAR(32) NOT NULL DEFAULT '',
  external_deployment_id TEXT NOT NULL DEFAULT '',
  provider_status VARCHAR(32) NOT NULL DEFAULT 'queued',
  deployment_url TEXT NOT NULL DEFAULT '',
  version_name VARCHAR(64) NOT NULL DEFAULT '',
  build_number BIGINT NOT NULL DEFAULT 0,
  git_commit VARCHAR(64) NOT NULL DEFAULT '',
  triggered_by VARCHAR(128) NOT NULL DEFAULT '',
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  duration_ms BIGINT NOT NULL DEFAULT 0,
  log_tail TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
  error_message TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS webhook_routes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL DEFAULT 'default',
  project_id UUID NOT NULL REFERENCES release_projects(id) ON DELETE CASCADE,
  repository_id UUID NOT NULL REFERENCES code_repositories(id) ON DELETE CASCADE,
  build_profile_id UUID REFERENCES build_profiles(id) ON DELETE SET NULL,
  event_type VARCHAR(64) NOT NULL,
  ref_pattern TEXT NOT NULL DEFAULT '*',
  action VARCHAR(32) NOT NULL DEFAULT 'all',
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (repository_id, event_type, ref_pattern, action)
);

CREATE INDEX IF NOT EXISTS idx_release_projects_status ON release_projects (tenant_id, lifecycle_status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_code_repositories_project ON code_repositories (tenant_id, project_id, provider);
CREATE INDEX IF NOT EXISTS idx_build_profiles_project ON build_profiles (tenant_id, project_id, enabled);
CREATE INDEX IF NOT EXISTS idx_build_center_runs_project ON build_center_runs (tenant_id, project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_build_center_runs_status ON build_center_runs (tenant_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_build_center_artifacts_run ON build_center_run_artifacts (tenant_id, run_id, created_at);
CREATE INDEX IF NOT EXISTS idx_deployment_targets_project ON deployment_targets (tenant_id, project_id, provider, enabled);
CREATE INDEX IF NOT EXISTS idx_deployment_records_target ON deployment_records (tenant_id, target_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_deployment_records_external
  ON deployment_records (target_id, external_deployment_id)
  WHERE external_deployment_id <> '';
CREATE INDEX IF NOT EXISTS idx_webhook_routes_repository ON webhook_routes (tenant_id, repository_id, enabled);

INSERT INTO system_permissions (id, tenant_id, name, code, module, permission_type)
VALUES
  (gen_random_uuid(), 'default', '查看构建中心', 'build:center:read', '构建中心', 'api'),
  (gen_random_uuid(), 'default', '触发构建任务', 'build:center:write', '构建中心', 'button'),
  (gen_random_uuid(), 'default', '维护部署目标', 'deploy:target:write', '部署中心', 'button')
ON CONFLICT (tenant_id, code) DO NOTHING;

INSERT INTO system_menus (id, tenant_id, title, path, icon, parent_title, sort_order)
VALUES
  (gen_random_uuid(), 'default', '构建中心', '/build-center', 'Hammer', '工作台', 25),
  (gen_random_uuid(), 'default', '部署目标', '/deploy-targets', 'Cloud', '工作台', 30)
ON CONFLICT (tenant_id, path) DO NOTHING;

INSERT INTO system_dictionaries (id, tenant_id, group_key, item_key, label, item_value, sort_order)
VALUES
  (gen_random_uuid(), 'default', 'code_provider', 'github', 'GitHub', 'github', 10),
  (gen_random_uuid(), 'default', 'code_provider', 'gitea', 'Gitea', 'gitea', 20),
  (gen_random_uuid(), 'default', 'deploy_provider', 'cloudflare_pages', 'Cloudflare Pages', 'cloudflare_pages', 10),
  (gen_random_uuid(), 'default', 'deploy_provider', 'cloudflare_worker', 'Cloudflare Worker', 'cloudflare_worker', 20),
  (gen_random_uuid(), 'default', 'deploy_provider', 'cloudflare_r2', 'Cloudflare R2', 'cloudflare_r2', 30)
ON CONFLICT (tenant_id, group_key, item_key) DO NOTHING;

INSERT INTO release_projects (
  tenant_id, project_key, name, description, owner_account, lifecycle_status,
  default_channel, metadata
)
VALUES (
  'default',
  'release-center',
  'Release Center',
  'Lightweight release center and local build-center integration.',
  'release.admin',
  'active',
  'dev',
  jsonb_build_object('source', 'migration_000004')
)
ON CONFLICT (tenant_id, project_key) DO NOTHING;

WITH project_row AS (
  SELECT id FROM release_projects WHERE tenant_id = 'default' AND project_key = 'release-center'
)
INSERT INTO code_repositories (
  tenant_id, project_id, provider, repo_url, repo_full_name, default_ref,
  webhook_enabled, trigger_on_push, trigger_on_tag, metadata
)
SELECT
  'default',
  project_row.id,
  'github',
  'https://github.com/liut-coder/release-center.git',
  'liut-coder/release-center',
  'main',
  FALSE,
  FALSE,
  FALSE,
  jsonb_build_object('source', 'migration_000004')
FROM project_row
ON CONFLICT (tenant_id, provider, repo_url) DO NOTHING;

WITH project_row AS (
  SELECT id FROM release_projects WHERE tenant_id = 'default' AND project_key = 'release-center'
)
INSERT INTO build_profiles (
  tenant_id, project_id, profile_key, name, build_center_project, stack_type,
  build_type, config_path, source_workdir, default_ref, default_version_name,
  default_version_code, default_channel, build_action, commands, artifact_rules, metadata
)
SELECT
  'default',
  project_row.id,
  'default',
  'Default release-center build',
  'release-center',
  'go-vite',
  'release',
  '/root/build-center/config/release-center.yml',
  '/root/build-center/projects/release-center',
  'main',
  '0.1.0',
  100,
  'dev',
  'all',
  jsonb_build_object(
    'test', jsonb_build_array('go test ./...'),
    'web', jsonb_build_array('npm ci', 'npm run build'),
    'binaries', jsonb_build_array(
      'go build -buildvcs=false -o ${WORKSPACE_DIR}/bin/server ./cmd/server',
      'go build -buildvcs=false -o ${WORKSPACE_DIR}/bin/releasectl ./cmd/releasectl',
      'go build -buildvcs=false -o ${WORKSPACE_DIR}/bin/migrate ./cmd/migrate'
    )
  ),
  jsonb_build_array(
    jsonb_build_object('name', 'web-dist', 'type', 'web_dist', 'path', 'web/dist', 'package', 'web-dist.tar.gz'),
    jsonb_build_object('name', 'server', 'type', 'binary', 'path', 'server'),
    jsonb_build_object('name', 'releasectl', 'type', 'binary', 'path', 'releasectl'),
    jsonb_build_object('name', 'migrate', 'type', 'binary', 'path', 'migrate')
  ),
  jsonb_build_object('source', 'migration_000004')
FROM project_row
ON CONFLICT (project_id, profile_key) DO NOTHING;

WITH project_row AS (
  SELECT id FROM release_projects WHERE tenant_id = 'default' AND project_key = 'release-center'
)
INSERT INTO deployment_targets (
  tenant_id, project_id, target_key, name, provider, environment,
  cloudflare_project_name, enabled, metadata
)
SELECT
  'default',
  project_row.id,
  'admin-pages',
  'Release Center Admin Pages',
  'cloudflare_pages',
  'prod',
  'release-center-admin',
  FALSE,
  jsonb_build_object('stage', 'planned', 'purpose', 'admin_web', 'source', 'migration_000004')
FROM project_row
ON CONFLICT (project_id, target_key) DO NOTHING;

WITH project_row AS (
  SELECT id FROM release_projects WHERE tenant_id = 'default' AND project_key = 'release-center'
)
INSERT INTO deployment_targets (
  tenant_id, project_id, target_key, name, provider, environment,
  cloudflare_script_name, enabled, metadata
)
SELECT
  'default',
  project_row.id,
  'api-worker',
  'Release Center API Worker',
  'cloudflare_worker',
  'prod',
  'release-center-api',
  FALSE,
  jsonb_build_object('stage', 'planned', 'purpose', 'webhook_and_edge_api', 'source', 'migration_000004')
FROM project_row
ON CONFLICT (project_id, target_key) DO NOTHING;

