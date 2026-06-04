DELETE FROM system_dictionaries
WHERE tenant_id = 'default'
  AND group_key = 'release_unit_type';

DELETE FROM system_permissions
WHERE tenant_id = 'default'
  AND code IN ('release:plan:read', 'release:plan:write');

DROP TABLE IF EXISTS release_plan_artifacts;
DROP TABLE IF EXISTS release_plans;
DROP TABLE IF EXISTS release_units;
DROP TABLE IF EXISTS release_environments;
