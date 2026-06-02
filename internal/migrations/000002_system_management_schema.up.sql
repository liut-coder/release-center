CREATE TABLE IF NOT EXISTS system_users (
  id UUID PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  name VARCHAR(128) NOT NULL,
  account VARCHAR(128) NOT NULL,
  role_code VARCHAR(64) NOT NULL DEFAULT '',
  department VARCHAR(128) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL DEFAULT 'enabled',
  last_login_at TIMESTAMPTZ,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, account)
);

CREATE TABLE IF NOT EXISTS system_roles (
  id UUID PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  name VARCHAR(128) NOT NULL,
  code VARCHAR(64) NOT NULL,
  data_scope VARCHAR(128) NOT NULL DEFAULT '',
  permissions TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, code)
);

CREATE TABLE IF NOT EXISTS system_permissions (
  id UUID PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  name VARCHAR(128) NOT NULL,
  code VARCHAR(128) NOT NULL,
  module VARCHAR(128) NOT NULL DEFAULT '',
  permission_type VARCHAR(32) NOT NULL DEFAULT 'button',
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, code)
);

CREATE TABLE IF NOT EXISTS system_dictionaries (
  id UUID PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  group_key VARCHAR(128) NOT NULL,
  item_key VARCHAR(128) NOT NULL,
  label VARCHAR(128) NOT NULL,
  item_value TEXT NOT NULL DEFAULT '',
  sort_order INT NOT NULL DEFAULT 100,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, group_key, item_key)
);

CREATE TABLE IF NOT EXISTS system_menus (
  id UUID PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  title VARCHAR(128) NOT NULL,
  path TEXT NOT NULL,
  icon VARCHAR(64) NOT NULL DEFAULT '',
  parent_title VARCHAR(128) NOT NULL DEFAULT '',
  sort_order INT NOT NULL DEFAULT 100,
  visible BOOLEAN NOT NULL DEFAULT TRUE,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, path)
);

INSERT INTO system_roles (id, tenant_id, name, code, data_scope, permissions)
VALUES
  (gen_random_uuid(), 'default', '系统管理员', 'system_admin', '全部数据', ARRAY['system:*', 'release:*']),
  (gen_random_uuid(), 'default', '发版管理员', 'release_admin', '发版数据', ARRAY['release:write', 'release:audit']),
  (gen_random_uuid(), 'default', '只读观察员', 'release_viewer', '只读数据', ARRAY['release:read'])
ON CONFLICT (tenant_id, code) DO NOTHING;

INSERT INTO system_users (id, tenant_id, name, account, role_code, department, status, last_login_at)
VALUES
  (gen_random_uuid(), 'default', '系统管理员', 'system.admin', 'system_admin', '平台工程', 'enabled', now()),
  (gen_random_uuid(), 'default', '发布管理员', 'release.admin', 'release_admin', '平台工程', 'enabled', now()),
  (gen_random_uuid(), 'default', '测试负责人', 'qa.lead', 'release_admin', '质量保障', 'enabled', now() - interval '1 day'),
  (gen_random_uuid(), 'default', '观察员', 'release.viewer', 'release_viewer', '运营支持', 'disabled', now() - interval '5 days')
ON CONFLICT (tenant_id, account) DO NOTHING;

INSERT INTO system_permissions (id, tenant_id, name, code, module, permission_type)
VALUES
  (gen_random_uuid(), 'default', '查看首页', 'dashboard:view', '工作台', 'menu'),
  (gen_random_uuid(), 'default', '查看发布', 'release:read', '发布中心', 'api'),
  (gen_random_uuid(), 'default', '管理发布', 'release:write', '发布中心', 'button'),
  (gen_random_uuid(), 'default', '查看审计', 'release:audit', '发布中心', 'api'),
  (gen_random_uuid(), 'default', '查看系统管理', 'system:read', '系统管理', 'api'),
  (gen_random_uuid(), 'default', '维护系统管理', 'system:write', '系统管理', 'api'),
  (gen_random_uuid(), 'default', '维护用户', 'system:user:write', '系统管理', 'button'),
  (gen_random_uuid(), 'default', '维护菜单', 'system:menu:write', '系统管理', 'button')
ON CONFLICT (tenant_id, code) DO NOTHING;

INSERT INTO system_dictionaries (id, tenant_id, group_key, item_key, label, item_value, sort_order)
VALUES
  (gen_random_uuid(), 'default', 'release_channel', 'stable', '正式渠道', 'stable', 10),
  (gen_random_uuid(), 'default', 'release_channel', 'beta', 'Beta 渠道', 'beta', 20),
  (gen_random_uuid(), 'default', 'resource_package', 'templates-bear', '打熊识图模板', 'templates-bear', 30),
  (gen_random_uuid(), 'default', 'audit_level', 'critical', '关键操作', 'critical', 40)
ON CONFLICT (tenant_id, group_key, item_key) DO NOTHING;

INSERT INTO system_menus (id, tenant_id, title, path, icon, parent_title, sort_order)
VALUES
  (gen_random_uuid(), 'default', '首页', '/dashboard', 'LayoutDashboard', '', 10),
  (gen_random_uuid(), 'default', '发布中心', '/release-center', 'Rocket', '工作台', 20),
  (gen_random_uuid(), 'default', '用户管理', '/system/users', 'Users', '系统管理', 30),
  (gen_random_uuid(), 'default', '角色管理', '/system/roles', 'Shield', '系统管理', 40),
  (gen_random_uuid(), 'default', '权限管理', '/system/permissions', 'KeyRound', '系统管理', 45),
  (gen_random_uuid(), 'default', '数据字典', '/system/dictionaries', 'Database', '系统管理', 50),
  (gen_random_uuid(), 'default', '菜单编辑', '/system/menus', 'ListTree', '系统管理', 60)
ON CONFLICT (tenant_id, path) DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_system_users_status ON system_users (tenant_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_system_roles_enabled ON system_roles (tenant_id, enabled, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_system_permissions_enabled ON system_permissions (tenant_id, enabled, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_system_dictionaries_group ON system_dictionaries (tenant_id, group_key, sort_order);
CREATE INDEX IF NOT EXISTS idx_system_menus_sort ON system_menus (tenant_id, sort_order);
