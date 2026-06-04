DELETE FROM system_dictionaries
WHERE tenant_id = 'default'
  AND group_key = 'worker_label'
  AND item_key IN ('linux', 'windows', 'android', 'docker', 'cloudflare');

DELETE FROM system_permissions
WHERE tenant_id = 'default'
  AND code IN ('worker:api:write', 'worker:api:read');

DROP TABLE IF EXISTS worker_tasks;
DROP TABLE IF EXISTS worker_heartbeats;
DROP TABLE IF EXISTS build_workers;
