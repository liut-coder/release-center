DELETE FROM system_dictionaries
WHERE tenant_id = 'default'
  AND (
    (group_key = 'code_provider' AND item_key IN ('github', 'gitea'))
    OR (group_key = 'deploy_provider' AND item_key IN ('cloudflare_pages', 'cloudflare_worker', 'cloudflare_r2'))
  );

DELETE FROM system_menus
WHERE tenant_id = 'default'
  AND path IN ('/build-center', '/deploy-targets');

DELETE FROM system_permissions
WHERE tenant_id = 'default'
  AND code IN ('build:center:read', 'build:center:write', 'deploy:target:write');

DROP TABLE IF EXISTS webhook_routes;
DROP TABLE IF EXISTS deployment_records;
DROP TABLE IF EXISTS deployment_targets;
DROP TABLE IF EXISTS build_center_run_artifacts;
DROP TABLE IF EXISTS build_center_runs;
DROP TABLE IF EXISTS build_profiles;
DROP TABLE IF EXISTS code_repositories;
DROP TABLE IF EXISTS release_projects;
