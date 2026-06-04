CREATE TABLE IF NOT EXISTS build_workers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL DEFAULT 'default',
  worker_key VARCHAR(128) NOT NULL,
  name VARCHAR(128) NOT NULL DEFAULT '',
  endpoint_url TEXT NOT NULL DEFAULT '',
  labels TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
  status VARCHAR(32) NOT NULL DEFAULT 'registered',
  capacity INT NOT NULL DEFAULT 1,
  running_tasks INT NOT NULL DEFAULT 0,
  last_seen_at TIMESTAMPTZ,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, worker_key)
);

CREATE TABLE IF NOT EXISTS worker_heartbeats (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL DEFAULT 'default',
  worker_id UUID NOT NULL REFERENCES build_workers(id) ON DELETE CASCADE,
  status VARCHAR(32) NOT NULL DEFAULT 'online',
  labels TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
  capacity INT NOT NULL DEFAULT 1,
  running_tasks INT NOT NULL DEFAULT 0,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS worker_tasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL DEFAULT 'default',
  worker_id UUID REFERENCES build_workers(id) ON DELETE SET NULL,
  build_run_id UUID REFERENCES build_center_runs(id) ON DELETE SET NULL,
  project_id UUID REFERENCES release_projects(id) ON DELETE SET NULL,
  build_profile_id UUID REFERENCES build_profiles(id) ON DELETE SET NULL,
  task_type VARCHAR(32) NOT NULL DEFAULT 'build',
  action VARCHAR(32) NOT NULL DEFAULT 'all',
  status VARCHAR(32) NOT NULL DEFAULT 'queued',
  required_labels TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
  priority INT NOT NULL DEFAULT 0,
  lease_token TEXT NOT NULL DEFAULT '',
  leased_until TIMESTAMPTZ,
  attempts INT NOT NULL DEFAULT 0,
  log_tail TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
  artifact_manifest JSONB NOT NULL DEFAULT '{}'::jsonb,
  error_message TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_build_workers_status ON build_workers (tenant_id, status, last_seen_at DESC);
CREATE INDEX IF NOT EXISTS idx_worker_heartbeats_worker ON worker_heartbeats (tenant_id, worker_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_worker_tasks_queue ON worker_tasks (tenant_id, status, priority DESC, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_worker_tasks_worker ON worker_tasks (tenant_id, worker_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_worker_tasks_labels ON worker_tasks USING GIN (required_labels);

INSERT INTO system_permissions (id, tenant_id, name, code, module, permission_type)
VALUES
  (gen_random_uuid(), 'default', 'Worker 接入', 'worker:api:write', '构建中心', 'api'),
  (gen_random_uuid(), 'default', '查看 Worker', 'worker:api:read', '构建中心', 'api')
ON CONFLICT (tenant_id, code) DO NOTHING;

INSERT INTO system_dictionaries (id, tenant_id, group_key, item_key, label, item_value, sort_order)
VALUES
  (gen_random_uuid(), 'default', 'worker_label', 'linux', 'Linux', 'linux', 10),
  (gen_random_uuid(), 'default', 'worker_label', 'windows', 'Windows', 'windows', 20),
  (gen_random_uuid(), 'default', 'worker_label', 'android', 'Android', 'android', 30),
  (gen_random_uuid(), 'default', 'worker_label', 'docker', 'Docker', 'docker', 40),
  (gen_random_uuid(), 'default', 'worker_label', 'cloudflare', 'Cloudflare', 'cloudflare', 50)
ON CONFLICT (tenant_id, group_key, item_key) DO NOTHING;
