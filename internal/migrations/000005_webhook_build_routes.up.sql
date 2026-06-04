WITH project_row AS (
  SELECT id FROM release_projects WHERE tenant_id = 'default' AND project_key = 'release-center'
),
repo_row AS (
  SELECT id FROM code_repositories WHERE tenant_id = 'default' AND provider = 'github' AND repo_full_name = 'liut-coder/release-center'
),
profile_row AS (
  SELECT id FROM build_profiles WHERE tenant_id = 'default' AND project_id = (SELECT id FROM project_row) AND profile_key = 'default'
)
INSERT INTO webhook_routes (
  tenant_id, project_id, repository_id, build_profile_id, event_type,
  ref_pattern, action, enabled, metadata
)
SELECT
  'default',
  project_row.id,
  repo_row.id,
  profile_row.id,
  'push',
  'refs/heads/main',
  'status',
  FALSE,
  jsonb_build_object(
    'source', 'migration_000005',
    'ready_for', 'github_push_to_build_center',
    'note', 'Enable code_repositories.webhook_enabled, trigger_on_push and this route after GitHub secret/auth is configured.'
  )
FROM project_row, repo_row, profile_row
WHERE repo_row.id IS NOT NULL AND profile_row.id IS NOT NULL
ON CONFLICT (repository_id, event_type, ref_pattern, action) DO NOTHING;

WITH project_row AS (
  SELECT id FROM release_projects WHERE tenant_id = 'default' AND project_key = 'release-center'
),
repo_row AS (
  SELECT id FROM code_repositories WHERE tenant_id = 'default' AND provider = 'github' AND repo_full_name = 'liut-coder/release-center'
),
profile_row AS (
  SELECT id FROM build_profiles WHERE tenant_id = 'default' AND project_id = (SELECT id FROM project_row) AND profile_key = 'default'
)
INSERT INTO webhook_routes (
  tenant_id, project_id, repository_id, build_profile_id, event_type,
  ref_pattern, action, enabled, metadata
)
SELECT
  'default',
  project_row.id,
  repo_row.id,
  profile_row.id,
  'push',
  'refs/tags/*',
  'all',
  FALSE,
  jsonb_build_object(
    'source', 'migration_000005',
    'ready_for', 'github_tag_to_full_build',
    'note', 'Keep disabled until GitHub webhook signature verification is configured.'
  )
FROM project_row, repo_row, profile_row
WHERE repo_row.id IS NOT NULL AND profile_row.id IS NOT NULL
ON CONFLICT (repository_id, event_type, ref_pattern, action) DO NOTHING;
