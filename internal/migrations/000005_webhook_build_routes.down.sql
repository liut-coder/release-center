DELETE FROM webhook_routes
WHERE tenant_id = 'default'
  AND metadata->>'source' = 'migration_000005';
