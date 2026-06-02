import { apiRequest, mockResponse } from "@/api/client";

const USE_MOCK = import.meta.env.VITE_USE_MOCK === "true";

export type SystemStatus = "enabled" | "disabled";

export interface SystemUser {
  id: string;
  name: string;
  account: string;
  role: string;
  role_code?: string;
  department?: string;
  status: SystemStatus;
  last_login?: string;
  last_login_at?: string;
  created_at?: string;
  updated_at?: string;
}

export interface SystemRole {
  id: string;
  name: string;
  code: string;
  users: number;
  scope: string;
  permissions: string[];
  enabled: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface SystemPermission {
  id: string;
  name: string;
  code: string;
  module: string;
  type: "menu" | "button" | "api" | string;
  enabled: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface SystemDictionary {
  id: string;
  group: string;
  key: string;
  label: string;
  value: string;
  sort: number;
  enabled: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface SystemMenu {
  id: string;
  title: string;
  path: string;
  icon: string;
  parent: string;
  sort: number;
  visible: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface SystemManagementOverview {
  _mock?: boolean;
  users: SystemUser[];
  roles: SystemRole[];
  permissions: SystemPermission[];
  dictionaries: SystemDictionary[];
  menus: SystemMenu[];
  message_zh?: string;
}

export interface CreateSystemUserPayload {
  name: string;
  account: string;
  role_code?: string;
  role?: string;
  department?: string;
  status?: SystemStatus;
}

export interface CreateSystemRolePayload {
  name: string;
  code: string;
  scope?: string;
  permissions?: string[];
  enabled?: boolean;
}

export interface CreateSystemPermissionPayload {
  name: string;
  code: string;
  module?: string;
  type?: "menu" | "button" | "api" | string;
  enabled?: boolean;
}

export interface CreateSystemDictionaryPayload {
  group: string;
  key: string;
  label: string;
  value?: string;
  sort?: number;
  enabled?: boolean;
}

export interface CreateSystemMenuPayload {
  title: string;
  path: string;
  icon?: string;
  parent?: string;
  sort?: number;
  visible?: boolean;
}

export function getSystemManagement() {
  if (USE_MOCK) return mockResponse(() => ({ ...systemManagementMock, _mock: true }));
  return apiRequest<unknown>("/admin/api/system/overview").then(normalizeSystemManagementOverview);
}

export function createSystemUser(payload: CreateSystemUserPayload) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, user: mockUser(payload) }));
  return apiRequest<unknown>("/admin/api/system/users", { method: "POST", body: JSON.stringify(payload) }).then(normalizeUserAction);
}

export function setSystemUserEnabled(id: string, enabled: boolean) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, user: { ...mockUser({ name: "Mock 用户", account: id }), id, status: enabled ? "enabled" : "disabled" } }));
  return apiRequest<unknown>(`/admin/api/system/users/${encodeURIComponent(id)}/${enabled ? "enable" : "disable"}`, { method: "POST" }).then(normalizeUserAction);
}

export function createSystemRole(payload: CreateSystemRolePayload) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, role: mockRole(payload) }));
  return apiRequest<unknown>("/admin/api/system/roles", { method: "POST", body: JSON.stringify(payload) }).then(normalizeRoleAction);
}

export function setSystemRoleEnabled(id: string, enabled: boolean) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, role: { ...mockRole({ name: "Mock 角色", code: id }), id, enabled } }));
  return apiRequest<unknown>(`/admin/api/system/roles/${encodeURIComponent(id)}/${enabled ? "enable" : "disable"}`, { method: "POST" }).then(normalizeRoleAction);
}

export function createSystemPermission(payload: CreateSystemPermissionPayload) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, permission: mockPermission(payload) }));
  return apiRequest<unknown>("/admin/api/system/permissions", { method: "POST", body: JSON.stringify(payload) }).then(normalizePermissionAction);
}

export function setSystemPermissionEnabled(id: string, enabled: boolean) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, permission: { ...mockPermission({ name: "Mock 权限", code: id }), id, enabled } }));
  return apiRequest<unknown>(`/admin/api/system/permissions/${encodeURIComponent(id)}/${enabled ? "enable" : "disable"}`, { method: "POST" }).then(normalizePermissionAction);
}

export function createSystemDictionary(payload: CreateSystemDictionaryPayload) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, dictionary: mockDictionary(payload) }));
  return apiRequest<unknown>("/admin/api/system/dictionaries", { method: "POST", body: JSON.stringify(payload) }).then(normalizeDictionaryAction);
}

export function setSystemDictionaryEnabled(id: string, enabled: boolean) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, dictionary: { ...mockDictionary({ group: "mock", key: id, label: "Mock 字典" }), id, enabled } }));
  return apiRequest<unknown>(`/admin/api/system/dictionaries/${encodeURIComponent(id)}/${enabled ? "enable" : "disable"}`, { method: "POST" }).then(normalizeDictionaryAction);
}

export function createSystemMenu(payload: CreateSystemMenuPayload) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, menu: mockMenu(payload) }));
  return apiRequest<unknown>("/admin/api/system/menus", { method: "POST", body: JSON.stringify(payload) }).then(normalizeMenuAction);
}

export function setSystemMenuVisible(id: string, visible: boolean) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, menu: { ...mockMenu({ title: "Mock 菜单", path: `/${id}` }), id, visible } }));
  return apiRequest<unknown>(`/admin/api/system/menus/${encodeURIComponent(id)}/${visible ? "show" : "hide"}`, { method: "POST" }).then(normalizeMenuAction);
}

export function normalizeSystemManagementOverview(payload: unknown): SystemManagementOverview {
  const record = unwrapDataRecord(payload);
  return {
    _mock: Boolean(record?._mock) || undefined,
    users: readArray<SystemUser>(record, ["users"]),
    roles: readArray<SystemRole>(record, ["roles"]),
    permissions: readArray<SystemPermission>(record, ["permissions"]),
    dictionaries: readArray<SystemDictionary>(record, ["dictionaries"]),
    menus: readArray<SystemMenu>(record, ["menus"]),
    message_zh: typeof record?.message_zh === "string" ? record.message_zh : undefined,
  };
}

function normalizeUserAction(payload: unknown): { ok: boolean; user: SystemUser } {
  const record = unwrapDataRecord(payload);
  const user = readRecord(record, ["user"]) ?? record;
  if (!user) throw new Error("用户响应格式不正确");
  return { ok: record?.ok !== false, user: user as unknown as SystemUser };
}

function normalizeRoleAction(payload: unknown): { ok: boolean; role: SystemRole } {
  const record = unwrapDataRecord(payload);
  const role = readRecord(record, ["role"]) ?? record;
  if (!role) throw new Error("角色响应格式不正确");
  return { ok: record?.ok !== false, role: role as unknown as SystemRole };
}

function normalizePermissionAction(payload: unknown): { ok: boolean; permission: SystemPermission } {
  const record = unwrapDataRecord(payload);
  const permission = readRecord(record, ["permission"]) ?? record;
  if (!permission) throw new Error("权限响应格式不正确");
  return { ok: record?.ok !== false, permission: permission as unknown as SystemPermission };
}

function normalizeDictionaryAction(payload: unknown): { ok: boolean; dictionary: SystemDictionary } {
  const record = unwrapDataRecord(payload);
  const dictionary = readRecord(record, ["dictionary"]) ?? record;
  if (!dictionary) throw new Error("字典响应格式不正确");
  return { ok: record?.ok !== false, dictionary: dictionary as unknown as SystemDictionary };
}

function normalizeMenuAction(payload: unknown): { ok: boolean; menu: SystemMenu } {
  const record = unwrapDataRecord(payload);
  const menu = readRecord(record, ["menu"]) ?? record;
  if (!menu) throw new Error("菜单响应格式不正确");
  return { ok: record?.ok !== false, menu: menu as unknown as SystemMenu };
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === "object" && !Array.isArray(value) ? (value as Record<string, unknown>) : undefined;
}

function unwrapDataRecord(value: unknown) {
  const record = asRecord(value);
  return asRecord(record?.data) ?? record;
}

function readRecord(record: Record<string, unknown> | undefined, keys: string[]) {
  if (!record) return undefined;
  for (const key of keys) {
    const nested = asRecord(record[key]);
    if (nested) return nested;
  }
  return undefined;
}

function readArray<T>(record: Record<string, unknown> | undefined, keys: string[]): T[] {
  if (!record) return [];
  for (const key of keys) {
    const value = record[key];
    if (Array.isArray(value)) return value as T[];
  }
  return [];
}

function mockUser(payload: CreateSystemUserPayload): SystemUser {
  return {
    id: `u_mock_${Date.now()}`,
    name: payload.name,
    account: payload.account,
    role: payload.role || payload.role_code || "只读观察员",
    role_code: payload.role_code,
    department: payload.department,
    status: payload.status ?? "enabled",
    last_login: "-",
  };
}

function mockRole(payload: CreateSystemRolePayload): SystemRole {
  return {
    id: `r_mock_${Date.now()}`,
    name: payload.name,
    code: payload.code,
    users: 0,
    scope: payload.scope || "自定义数据",
    permissions: payload.permissions ?? [],
    enabled: payload.enabled ?? true,
  };
}

function mockPermission(payload: CreateSystemPermissionPayload): SystemPermission {
  return {
    id: `p_mock_${Date.now()}`,
    name: payload.name,
    code: payload.code,
    module: payload.module || "系统管理",
    type: payload.type || "button",
    enabled: payload.enabled ?? true,
  };
}

function mockDictionary(payload: CreateSystemDictionaryPayload): SystemDictionary {
  return {
    id: `d_mock_${Date.now()}`,
    group: payload.group,
    key: payload.key,
    label: payload.label,
    value: payload.value ?? payload.key,
    sort: payload.sort ?? 100,
    enabled: payload.enabled ?? true,
  };
}

function mockMenu(payload: CreateSystemMenuPayload): SystemMenu {
  return {
    id: `m_mock_${Date.now()}`,
    title: payload.title,
    path: payload.path,
    icon: payload.icon || "Settings",
    parent: payload.parent || "系统管理",
    sort: payload.sort ?? 100,
    visible: payload.visible ?? true,
  };
}

const systemManagementMock: SystemManagementOverview = {
  users: [
    { id: "u_001", name: "发布管理员", account: "release.admin", role: "发版管理员", role_code: "release_admin", department: "平台工程", status: "enabled", last_login: "2026-06-02 13:46" },
    { id: "u_002", name: "测试负责人", account: "qa.lead", role: "发版管理员", role_code: "release_admin", department: "质量保障", status: "enabled", last_login: "2026-06-01 18:21" },
    { id: "u_003", name: "观察员", account: "release.viewer", role: "只读观察员", role_code: "release_viewer", department: "运营支持", status: "disabled", last_login: "2026-05-28 09:12" },
  ],
  roles: [
    { id: "r_001", name: "系统管理员", code: "system_admin", users: 1, scope: "全部数据", permissions: ["system:*", "release:*"], enabled: true },
    { id: "r_002", name: "发版管理员", code: "release_admin", users: 2, scope: "发版数据", permissions: ["release:write", "release:audit"], enabled: true },
    { id: "r_003", name: "只读观察员", code: "release_viewer", users: 1, scope: "只读数据", permissions: ["release:read"], enabled: true },
  ],
  permissions: [
    { id: "p_001", name: "查看首页", code: "dashboard:view", module: "工作台", type: "menu", enabled: true },
    { id: "p_002", name: "管理发布", code: "release:write", module: "发布中心", type: "button", enabled: true },
    { id: "p_003", name: "查看审计", code: "release:audit", module: "发布中心", type: "api", enabled: true },
    { id: "p_004", name: "维护用户", code: "system:user:write", module: "系统管理", type: "button", enabled: true },
    { id: "p_005", name: "维护菜单", code: "system:menu:write", module: "系统管理", type: "button", enabled: true },
  ],
  dictionaries: [
    { id: "d_001", group: "release_channel", key: "stable", label: "正式渠道", value: "stable", sort: 10, enabled: true },
    { id: "d_002", group: "release_channel", key: "beta", label: "Beta 渠道", value: "beta", sort: 20, enabled: true },
    { id: "d_003", group: "resource_package", key: "templates-bear", label: "打熊识图模板", value: "templates-bear", sort: 30, enabled: true },
    { id: "d_004", group: "audit_level", key: "critical", label: "关键操作", value: "critical", sort: 40, enabled: true },
  ],
  menus: [
    { id: "m_001", title: "首页", path: "/dashboard", icon: "LayoutDashboard", parent: "", sort: 10, visible: true },
    { id: "m_002", title: "发布中心", path: "/release-center", icon: "Rocket", parent: "工作台", sort: 20, visible: true },
    { id: "m_003", title: "用户管理", path: "/system/users", icon: "Users", parent: "系统管理", sort: 30, visible: true },
    { id: "m_004", title: "角色管理", path: "/system/roles", icon: "Shield", parent: "系统管理", sort: 40, visible: true },
    { id: "m_005", title: "权限管理", path: "/system/permissions", icon: "KeyRound", parent: "系统管理", sort: 45, visible: true },
    { id: "m_006", title: "数据字典", path: "/system/dictionaries", icon: "Database", parent: "系统管理", sort: 50, visible: true },
    { id: "m_007", title: "菜单编辑", path: "/system/menus", icon: "ListTree", parent: "系统管理", sort: 60, visible: true },
  ],
};
