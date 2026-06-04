import { useMemo, useState, type ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  BookOpen,
  Database,
  Hammer,
  Home,
  KeyRound,
  LayoutDashboard,
  ListTree,
  LogOut,
  Menu,
  Rocket,
  Search,
  Shield,
  SlidersHorizontal,
  Users,
} from "lucide-react";
import { getAppTokens, saveAppTokens } from "@/api/client";
import {
  createSystemDictionary,
  createSystemMenu,
  createSystemPermission,
  createSystemRole,
  createSystemUser,
  getSystemManagement,
  setSystemDictionaryEnabled,
  setSystemMenuVisible,
  setSystemPermissionEnabled,
  setSystemRoleEnabled,
  setSystemUserEnabled,
  type SystemDictionary,
  type SystemManagementOverview,
  type SystemMenu,
  type SystemPermission,
  type SystemRole,
  type SystemUser,
} from "@/api/systemManagement";
import { AppReleasesPage } from "@/pages/AppReleasesPage";
import { BuildCenterPage } from "@/pages/BuildCenterPage";
import { ApiErrorState } from "@/components/stable/StableAdminComponents";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Switch } from "@/components/ui/Switch";
import { Table, Td, Th } from "@/components/ui/Table";
import { cn } from "@/lib/cn";

type AdminPageKey = "dashboard" | "release-center" | "build-center" | "users" | "roles" | "permissions" | "dictionaries" | "menus";

const navigation: Array<{
  group: string;
  items: Array<{ key: AdminPageKey; label: string; icon: typeof Home }>;
}> = [
  {
    group: "工作台",
    items: [
      { key: "dashboard", label: "首页", icon: LayoutDashboard },
      { key: "release-center", label: "发布中心", icon: Rocket },
      { key: "build-center", label: "构建中心", icon: Hammer },
    ],
  },
  {
    group: "系统管理",
    items: [
      { key: "users", label: "用户管理", icon: Users },
      { key: "roles", label: "角色管理", icon: Shield },
      { key: "permissions", label: "权限管理", icon: KeyRound },
      { key: "dictionaries", label: "数据字典", icon: Database },
      { key: "menus", label: "菜单编辑", icon: ListTree },
    ],
  },
];

const SYSTEM_MANAGEMENT_QUERY_KEY = ["system-management"] as const;

const emptySystemManagement: SystemManagementOverview = {
  users: [],
  roles: [],
  permissions: [],
  dictionaries: [],
  menus: [],
};

type SystemManagementActions = {
  pending: boolean;
  error: unknown;
  createUser: () => void;
  setUserEnabled: (id: string, enabled: boolean) => void;
  createRole: () => void;
  setRoleEnabled: (id: string, enabled: boolean) => void;
  createPermission: () => void;
  setPermissionEnabled: (id: string, enabled: boolean) => void;
  createDictionary: (group: string, sort: number) => void;
  setDictionaryEnabled: (id: string, enabled: boolean) => void;
  createMenu: (sort: number) => void;
  setMenuVisible: (id: string, visible: boolean) => void;
};

type SystemPageRenderContext = {
  system: SystemManagementOverview;
  loading: boolean;
  error: unknown;
  actions: SystemManagementActions;
};

export function App() {
  const storedToken = getAppTokens().configToken;
  const [session, setSession] = useState(() => ({
    signedIn: import.meta.env.VITE_USE_MOCK === "true" || Boolean(storedToken),
    token: storedToken,
    name: storedToken ? "Release Admin" : "演示管理员",
  }));
  const [activePage, setActivePage] = useState<AdminPageKey>("dashboard");
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const systemQuery = useQuery({
    queryKey: SYSTEM_MANAGEMENT_QUERY_KEY,
    queryFn: getSystemManagement,
    enabled: session.signedIn,
  });
  const systemActions = useSystemManagementActions();

  if (!session.signedIn) {
    return <LoginPage onLogin={(token, name) => setSession({ signedIn: true, token, name })} />;
  }

  const currentLabel = navigation.flatMap((group) => group.items).find((item) => item.key === activePage)?.label ?? "首页";
  const system = systemQuery.data ?? emptySystemManagement;

  return (
    <main className="admin-shell min-h-screen bg-muted/30">
      <div className="flex min-h-screen">
        <aside
          className={cn(
            "fixed inset-y-0 left-0 z-40 w-64 border-r bg-white transition-transform lg:static lg:translate-x-0",
            sidebarOpen ? "translate-x-0" : "-translate-x-full",
          )}
        >
          <div className="flex h-14 items-center gap-2 border-b px-4">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-black text-white">
              <SlidersHorizontal className="h-4 w-4" />
            </div>
            <div className="min-w-0">
              <div className="truncate text-sm font-semibold">管理控制台</div>
              <div className="text-[11px] text-muted-foreground">Release Center</div>
            </div>
          </div>
          <nav className="space-y-5 px-3 py-4">
            {navigation.map((group) => (
              <div key={group.group}>
                <div className="px-2 pb-2 text-[11px] font-medium text-muted-foreground">{group.group}</div>
                <div className="space-y-1">
                  {group.items.map((item) => (
                    <NavButton
                      key={item.key}
                      active={activePage === item.key}
                      icon={item.icon}
                      label={item.label}
                      onClick={() => {
                        setActivePage(item.key);
                        setSidebarOpen(false);
                      }}
                    />
                  ))}
                </div>
              </div>
            ))}
          </nav>
        </aside>
        {sidebarOpen ? <button className="fixed inset-0 z-30 bg-black/20 lg:hidden" aria-label="关闭菜单" onClick={() => setSidebarOpen(false)} /> : null}
        <section className="min-w-0 flex-1">
          <header className="sticky top-0 z-20 flex h-14 items-center justify-between border-b bg-white/95 px-4 backdrop-blur">
            <div className="flex min-w-0 items-center gap-3">
              <Button variant="ghost" size="icon" className="lg:hidden" onClick={() => setSidebarOpen(true)}>
                <Menu className="h-4 w-4" />
              </Button>
              <div className="min-w-0">
                <div className="truncate text-sm font-semibold">{currentLabel}</div>
                <div className="text-[11px] text-muted-foreground">生产后台基础脚手架</div>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Badge tone="success">{session.token ? "已认证" : "演示"}</Badge>
              <div className="hidden text-right text-xs sm:block">
                <div className="font-medium">{session.name}</div>
                <div className="text-muted-foreground">系统管理员</div>
              </div>
              <Button
                variant="secondary"
                size="icon"
                onClick={() => {
                  saveAppTokens({ configToken: "" });
                  setSession({ signedIn: false, token: "", name: "" });
                }}
              >
                <LogOut className="h-4 w-4" />
              </Button>
            </div>
          </header>
          <div className="admin-content mx-auto max-w-[1600px] px-4 py-4 sm:px-6 lg:px-8">
            {renderPage(activePage, setActivePage, {
              system,
              loading: systemQuery.isLoading,
              error: systemQuery.error,
              actions: systemActions,
            })}
          </div>
        </section>
      </div>
    </main>
  );
}

function useSystemManagementActions(): SystemManagementActions {
  const queryClient = useQueryClient();
  const invalidateSystemManagement = () => queryClient.invalidateQueries({ queryKey: SYSTEM_MANAGEMENT_QUERY_KEY });
  const mutationOptions = { onSuccess: invalidateSystemManagement };
  const createUserMutation = useMutation({ mutationFn: createSystemUser, ...mutationOptions });
  const userStatusMutation = useMutation({ mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => setSystemUserEnabled(id, enabled), ...mutationOptions });
  const createRoleMutation = useMutation({ mutationFn: createSystemRole, ...mutationOptions });
  const roleStatusMutation = useMutation({ mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => setSystemRoleEnabled(id, enabled), ...mutationOptions });
  const createPermissionMutation = useMutation({ mutationFn: createSystemPermission, ...mutationOptions });
  const permissionStatusMutation = useMutation({ mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => setSystemPermissionEnabled(id, enabled), ...mutationOptions });
  const createDictionaryMutation = useMutation({ mutationFn: createSystemDictionary, ...mutationOptions });
  const dictionaryStatusMutation = useMutation({ mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => setSystemDictionaryEnabled(id, enabled), ...mutationOptions });
  const createMenuMutation = useMutation({ mutationFn: createSystemMenu, ...mutationOptions });
  const menuVisibleMutation = useMutation({ mutationFn: ({ id, visible }: { id: string; visible: boolean }) => setSystemMenuVisible(id, visible), ...mutationOptions });
  const mutations = [
    createUserMutation,
    userStatusMutation,
    createRoleMutation,
    roleStatusMutation,
    createPermissionMutation,
    permissionStatusMutation,
    createDictionaryMutation,
    dictionaryStatusMutation,
    createMenuMutation,
    menuVisibleMutation,
  ];
  const suffix = () => Date.now().toString(36);

  return {
    pending: mutations.some((mutation) => mutation.isPending),
    error: mutations.find((mutation) => mutation.error)?.error,
    createUser: () =>
      createUserMutation.mutate({
        name: "新用户",
        account: `user.${suffix()}`,
        role_code: "release_viewer",
        department: "平台工程",
        status: "enabled",
      }),
    setUserEnabled: (id, enabled) => userStatusMutation.mutate({ id, enabled }),
    createRole: () =>
      createRoleMutation.mutate({
        name: "新角色",
        code: `custom_role_${suffix()}`,
        scope: "自定义数据",
        permissions: [],
        enabled: true,
      }),
    setRoleEnabled: (id, enabled) => roleStatusMutation.mutate({ id, enabled }),
    createPermission: () =>
      createPermissionMutation.mutate({
        name: "新权限",
        code: `custom:permission:${suffix()}`,
        module: "系统管理",
        type: "button",
        enabled: true,
      }),
    setPermissionEnabled: (id, enabled) => permissionStatusMutation.mutate({ id, enabled }),
    createDictionary: (group, sort) =>
      createDictionaryMutation.mutate({
        group: group || "custom_group",
        key: `custom_key_${suffix()}`,
        label: "新字典项",
        value: "custom",
        sort,
        enabled: true,
      }),
    setDictionaryEnabled: (id, enabled) => dictionaryStatusMutation.mutate({ id, enabled }),
    createMenu: (sort) =>
      createMenuMutation.mutate({
        title: "新菜单",
        path: `/custom/${suffix()}`,
        icon: "Settings",
        parent: "系统管理",
        sort,
        visible: true,
      }),
    setMenuVisible: (id, visible) => menuVisibleMutation.mutate({ id, visible }),
  };
}

function LoginPage({ onLogin }: { onLogin: (token: string, name: string) => void }) {
  const [account, setAccount] = useState("admin");
  const [token, setToken] = useState(getAppTokens().configToken);

  return (
    <main className="admin-shell flex min-h-screen items-center justify-center bg-muted/30 px-4">
      <div className="w-full max-w-[380px]">
        <div className="mb-4 flex items-center gap-2">
          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-black text-white">
            <KeyRound className="h-4 w-4" />
          </div>
          <div>
            <h1 className="text-lg font-semibold">后台登录</h1>
            <div className="text-xs text-muted-foreground">Release Center Admin</div>
          </div>
        </div>
        <Card className="p-4">
          <div className="grid gap-3">
            <label className="grid gap-1.5 text-xs">
              <span className="font-medium">账号</span>
              <Input value={account} onChange={(event) => setAccount(event.target.value)} />
            </label>
            <label className="grid gap-1.5 text-xs">
              <span className="font-medium">后台 Token</span>
              <Input value={token} type="password" onChange={(event) => setToken(event.target.value)} />
            </label>
            <Button
              className="mt-1 w-full"
              onClick={() => {
                saveAppTokens({ configToken: token.trim() });
                onLogin(token.trim(), account.trim() || "Admin");
              }}
            >
              登录
            </Button>
          </div>
        </Card>
      </div>
    </main>
  );
}

function NavButton({ active, icon: Icon, label, onClick }: { active: boolean; icon: typeof Home; label: string; onClick: () => void }) {
  return (
    <button
      className={cn(
        "flex h-9 w-full items-center gap-2 rounded-lg px-2.5 text-left text-xs font-medium transition",
        active ? "bg-black text-white" : "text-muted-foreground hover:bg-muted hover:text-foreground",
      )}
      onClick={onClick}
    >
      <Icon className="h-4 w-4 shrink-0" />
      <span className="truncate">{label}</span>
    </button>
  );
}

function renderPage(activePage: AdminPageKey, setActivePage: (page: AdminPageKey) => void, context: SystemPageRenderContext) {
  if (activePage === "release-center") return <AppReleasesPage />;
  if (activePage === "build-center") return <BuildCenterPage />;
  if (activePage === "users") return <UsersPage users={context.system.users} loading={context.loading} error={context.error} actions={context.actions} />;
  if (activePage === "roles") return <RolesPage roles={context.system.roles} loading={context.loading} error={context.error} actions={context.actions} />;
  if (activePage === "permissions") {
    return <PermissionsPage permissions={context.system.permissions} loading={context.loading} error={context.error} actions={context.actions} />;
  }
  if (activePage === "dictionaries") {
    return <DictionariesPage dictionaries={context.system.dictionaries} loading={context.loading} error={context.error} actions={context.actions} />;
  }
  if (activePage === "menus") return <MenusPage menus={context.system.menus} loading={context.loading} error={context.error} actions={context.actions} />;
  return <DashboardPage system={context.system} loading={context.loading} error={context.error} onOpen={setActivePage} />;
}

function DashboardPage({ system, loading, error, onOpen }: { system: SystemManagementOverview; loading: boolean; error: unknown; onOpen: (page: AdminPageKey) => void }) {
  const quickEntries: Array<{ label: string; value: string; page: AdminPageKey; icon: typeof Home }> = [
    { label: "用户", value: loading ? "-" : String(system.users.length), page: "users", icon: Users },
    { label: "角色", value: loading ? "-" : String(system.roles.length), page: "roles", icon: Shield },
    { label: "权限点", value: loading ? "-" : String(system.permissions.length), page: "permissions", icon: KeyRound },
    { label: "字典项", value: loading ? "-" : String(system.dictionaries.length), page: "dictionaries", icon: Database },
  ];
  const recentUsers = system.users.slice(0, 3);
  const menuCount = system.menus.length || navigation.flatMap((group) => group.items).length;

  return (
    <div className="grid gap-4">
      {error ? <ApiErrorState error={error} title="系统管理数据读取失败" /> : null}
      <div className="grid gap-3 md:grid-cols-4">
        {quickEntries.map((item) => (
          <button key={item.label} className="rounded-lg border bg-white p-3 text-left transition hover:border-black" onClick={() => onOpen(item.page)}>
            <div className="flex items-center justify-between gap-3">
              <span className="text-xs text-muted-foreground">{item.label}</span>
              <item.icon className="h-4 w-4 text-muted-foreground" />
            </div>
            <div className="mt-3 text-2xl font-semibold">{item.value}</div>
          </button>
        ))}
      </div>
      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
        <Card>
          <div className="mb-3 flex items-center justify-between">
            <div className="font-medium">系统模块</div>
            <Badge>{menuCount} 个菜单</Badge>
          </div>
          <div className="grid gap-2 md:grid-cols-2">
            {navigation.flatMap((group) => group.items).map((item) => (
              <button key={item.key} className="flex items-center gap-3 rounded-lg border p-3 text-left hover:border-black" onClick={() => onOpen(item.key)}>
                <item.icon className="h-4 w-4" />
                <div className="min-w-0">
                  <div className="truncate text-sm font-medium">{item.label}</div>
                  <div className="text-xs text-muted-foreground">{item.key}</div>
                </div>
              </button>
            ))}
          </div>
        </Card>
        <Card>
          <div className="mb-3 font-medium">最近登录</div>
          <div className="grid gap-2">
            {loading ? <SystemEmptyState label="正在读取登录记录" /> : null}
            {!loading && recentUsers.length === 0 ? <SystemEmptyState label="暂无登录记录" /> : null}
            {recentUsers.map((user) => (
              <div key={user.id} className="flex items-center justify-between gap-3 rounded-lg border p-2.5 text-xs">
                <div className="min-w-0">
                  <div className="truncate font-medium">{user.name}</div>
                  <div className="truncate text-muted-foreground">{user.account}</div>
                </div>
                <span className="shrink-0 text-muted-foreground">{formatLastLogin(user).slice(5)}</span>
              </div>
            ))}
          </div>
        </Card>
      </div>
    </div>
  );
}

function UsersPage({ users, loading, error, actions }: { users: SystemUser[]; loading: boolean; error: unknown; actions: SystemManagementActions }) {
  const [query, setQuery] = useState("");
  const rows = users.filter((user) => [user.name, user.account, user.role, user.department ?? ""].join(" ").toLowerCase().includes(query.toLowerCase()));

  return (
    <SystemPageShell
      title="用户管理"
      loading={loading}
      error={error}
      actionError={actions.error}
      actions={
        <Button onClick={actions.createUser} disabled={actions.pending}>
          <Users className="mr-1.5 h-4 w-4" />
          新增用户
        </Button>
      }
    >
      <ToolbarSearch value={query} onChange={setQuery} />
      <Table>
        <thead>
          <tr>
            <Th>姓名</Th>
            <Th>账号</Th>
            <Th>角色</Th>
            <Th>部门</Th>
            <Th>状态</Th>
            <Th>最后登录</Th>
            <Th>操作</Th>
          </tr>
        </thead>
        <tbody>
          {rows.map((user) => (
            <tr key={user.id}>
              <Td>{user.name}</Td>
              <Td>{user.account}</Td>
              <Td>{user.role}</Td>
              <Td>{user.department || "-"}</Td>
              <Td>
                <Badge tone={user.status === "enabled" ? "success" : "warning"}>{user.status === "enabled" ? "启用" : "停用"}</Badge>
              </Td>
              <Td>{formatLastLogin(user)}</Td>
              <Td>
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={actions.pending}
                  onClick={() => actions.setUserEnabled(user.id, user.status !== "enabled")}
                >
                  {user.status === "enabled" ? "停用" : "启用"}
                </Button>
              </Td>
            </tr>
          ))}
          {rows.length === 0 ? <EmptyTableRow colSpan={7} label={query ? "没有匹配的用户" : "暂无用户"} /> : null}
        </tbody>
      </Table>
    </SystemPageShell>
  );
}

function RolesPage({ roles, loading, error, actions }: { roles: SystemRole[]; loading: boolean; error: unknown; actions: SystemManagementActions }) {
  return (
    <SystemPageShell
      title="角色管理"
      loading={loading}
      error={error}
      actionError={actions.error}
      actions={
        <Button onClick={actions.createRole} disabled={actions.pending}>
          <Shield className="mr-1.5 h-4 w-4" />
          新增角色
        </Button>
      }
    >
      <div className="grid gap-3 xl:grid-cols-3">
        {roles.length === 0 ? <SystemEmptyState label="暂无角色" /> : null}
        {roles.map((role) => (
          <div key={role.id} className="rounded-lg border p-3">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <div className="truncate font-medium">{role.name}</div>
                <div className="mt-1 font-mono text-xs text-muted-foreground">{role.code}</div>
              </div>
              <Switch checked={role.enabled} disabled={actions.pending} onCheckedChange={(checked) => actions.setRoleEnabled(role.id, checked)} />
            </div>
            <div className="mt-3 grid gap-2 text-xs">
              <InfoRow label="用户数" value={`${role.users}`} />
              <InfoRow label="数据范围" value={role.scope} />
              <InfoRow label="权限" value={role.permissions.join(", ") || "-"} />
            </div>
          </div>
        ))}
      </div>
    </SystemPageShell>
  );
}

function PermissionsPage({ permissions, loading, error, actions }: { permissions: SystemPermission[]; loading: boolean; error: unknown; actions: SystemManagementActions }) {
  return (
    <SystemPageShell
      title="权限管理"
      loading={loading}
      error={error}
      actionError={actions.error}
      actions={
        <Button onClick={actions.createPermission} disabled={actions.pending}>
          <KeyRound className="mr-1.5 h-4 w-4" />
          新增权限
        </Button>
      }
    >
      <Table>
        <thead>
          <tr>
            <Th>权限名称</Th>
            <Th>权限标识</Th>
            <Th>模块</Th>
            <Th>类型</Th>
            <Th>状态</Th>
            <Th>操作</Th>
          </tr>
        </thead>
        <tbody>
          {permissions.map((permission) => (
            <tr key={permission.id}>
              <Td>{permission.name}</Td>
              <Td>{permission.code}</Td>
              <Td>{permission.module}</Td>
              <Td>{permission.type}</Td>
              <Td>
                <Badge tone={permission.enabled ? "success" : "warning"}>{permission.enabled ? "启用" : "停用"}</Badge>
              </Td>
              <Td>
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={actions.pending}
                  onClick={() => actions.setPermissionEnabled(permission.id, !permission.enabled)}
                >
                  {permission.enabled ? "停用" : "启用"}
                </Button>
              </Td>
            </tr>
          ))}
          {permissions.length === 0 ? <EmptyTableRow colSpan={6} label="暂无权限点" /> : null}
        </tbody>
      </Table>
    </SystemPageShell>
  );
}

function DictionariesPage({ dictionaries, loading, error, actions }: { dictionaries: SystemDictionary[]; loading: boolean; error: unknown; actions: SystemManagementActions }) {
  const groups = useMemo(() => Array.from(new Set(dictionaries.map((item) => item.group))), [dictionaries]);
  const [activeGroup, setActiveGroup] = useState("");
  const selectedGroup = groups.includes(activeGroup) ? activeGroup : groups[0] ?? "";
  const rows = dictionaries.filter((item) => item.group === selectedGroup);

  return (
    <SystemPageShell
      title="数据字典"
      loading={loading}
      error={error}
      actionError={actions.error}
      actions={
        <Button onClick={() => actions.createDictionary(selectedGroup, dictionaries.length + 1)} disabled={actions.pending}>
          <BookOpen className="mr-1.5 h-4 w-4" />
          新增字典
        </Button>
      }
    >
      <div className="grid gap-4 lg:grid-cols-[220px_minmax(0,1fr)]">
        <div className="rounded-lg border p-2">
          <div className="px-2 py-1.5 text-xs font-medium text-muted-foreground">字典分组</div>
          <div className="grid gap-1">
            {groups.length === 0 ? <SystemEmptyState label="暂无分组" /> : null}
            {groups.map((group) => (
              <button
                key={group}
                className={cn("rounded-lg px-2.5 py-2 text-left text-xs font-medium", selectedGroup === group ? "bg-black text-white" : "hover:bg-muted")}
                onClick={() => setActiveGroup(group)}
              >
                {group}
              </button>
            ))}
          </div>
        </div>
        <Table>
          <thead>
            <tr>
              <Th>标签</Th>
              <Th>键</Th>
              <Th>值</Th>
              <Th>排序</Th>
              <Th>状态</Th>
              <Th>操作</Th>
            </tr>
          </thead>
          <tbody>
            {rows.map((item) => (
              <tr key={item.id}>
                <Td>{item.label}</Td>
                <Td>{item.key}</Td>
                <Td>{item.value}</Td>
                <Td>{item.sort}</Td>
                <Td>
                  <Badge tone={item.enabled ? "success" : "warning"}>{item.enabled ? "启用" : "停用"}</Badge>
                </Td>
                <Td>
                  <Button variant="secondary" size="sm" disabled={actions.pending} onClick={() => actions.setDictionaryEnabled(item.id, !item.enabled)}>
                    {item.enabled ? "停用" : "启用"}
                  </Button>
                </Td>
              </tr>
            ))}
            {rows.length === 0 ? <EmptyTableRow colSpan={6} label="暂无字典项" /> : null}
          </tbody>
        </Table>
      </div>
    </SystemPageShell>
  );
}

function MenusPage({ menus, loading, error, actions }: { menus: SystemMenu[]; loading: boolean; error: unknown; actions: SystemManagementActions }) {
  const nextSort = (menus.at(-1)?.sort ?? 0) + 10;
  return (
    <SystemPageShell
      title="菜单编辑"
      loading={loading}
      error={error}
      actionError={actions.error}
      actions={
        <Button onClick={() => actions.createMenu(nextSort)} disabled={actions.pending}>
          <ListTree className="mr-1.5 h-4 w-4" />
          新增菜单
        </Button>
      }
    >
      <Table>
        <thead>
          <tr>
            <Th>菜单名称</Th>
            <Th>路径</Th>
            <Th>图标</Th>
            <Th>上级</Th>
            <Th>排序</Th>
            <Th>显示</Th>
            <Th>操作</Th>
          </tr>
        </thead>
        <tbody>
          {menus.map((menu) => (
            <tr key={menu.id}>
              <Td>{menu.title}</Td>
              <Td>{menu.path}</Td>
              <Td>{menu.icon}</Td>
              <Td>{menu.parent}</Td>
              <Td>{menu.sort}</Td>
              <Td>
                <Switch checked={menu.visible} disabled={actions.pending} onCheckedChange={(checked) => actions.setMenuVisible(menu.id, checked)} />
              </Td>
              <Td>
                <Button variant="secondary" size="sm" disabled={actions.pending} onClick={() => actions.setMenuVisible(menu.id, !menu.visible)}>
                  {menu.visible ? "隐藏" : "显示"}
                </Button>
              </Td>
            </tr>
          ))}
          {menus.length === 0 ? <EmptyTableRow colSpan={7} label="暂无菜单" /> : null}
        </tbody>
      </Table>
    </SystemPageShell>
  );
}

function SystemPageShell({
  title,
  actions,
  loading,
  error,
  actionError,
  children,
}: {
  title: string;
  actions?: ReactNode;
  loading: boolean;
  error: unknown;
  actionError: unknown;
  children: ReactNode;
}) {
  return (
    <div className="grid gap-4">
      <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
        <div>
          <h1 className="text-lg font-semibold">{title}</h1>
          <div className="text-xs text-muted-foreground">System Administration</div>
        </div>
        <div className="flex flex-wrap gap-2">{actions}</div>
      </div>
      {error ? <ApiErrorState error={error} title={`${title}数据读取失败`} /> : null}
      {actionError ? <ApiErrorState error={actionError} title={`${title}操作失败`} /> : null}
      <Card>{loading ? <SystemEmptyState label="正在读取数据" /> : children}</Card>
    </div>
  );
}

function ToolbarSearch({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  return (
    <div className="mb-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
      <label className="relative w-full sm:max-w-sm">
        <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input className="pl-8" value={value} onChange={(event) => onChange(event.target.value)} placeholder="搜索" />
      </label>
    </div>
  );
}

function EmptyTableRow({ colSpan, label }: { colSpan: number; label: string }) {
  return (
    <tr>
      <Td colSpan={colSpan} className="py-8 text-center text-muted-foreground">
        {label}
      </Td>
    </tr>
  );
}

function SystemEmptyState({ label }: { label: string }) {
  return <div className="rounded-lg border border-dashed px-3 py-6 text-center text-xs text-muted-foreground">{label}</div>;
}

function formatLastLogin(user: SystemUser) {
  return user.last_login || user.last_login_at || "-";
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-3">
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span className="min-w-0 truncate text-right font-medium">{value}</span>
    </div>
  );
}
