import { useMemo, useState, type ReactNode } from "react";
import {
  BookOpen,
  Database,
  Home,
  KeyRound,
  LayoutDashboard,
  ListTree,
  LogOut,
  Menu,
  Rocket,
  Save,
  Search,
  Shield,
  SlidersHorizontal,
  Users,
} from "lucide-react";
import { getAppTokens, saveAppTokens } from "@/api/client";
import { AppReleasesPage } from "@/pages/AppReleasesPage";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Switch } from "@/components/ui/Switch";
import { Table, Td, Th } from "@/components/ui/Table";
import { cn } from "@/lib/cn";

type AdminPageKey = "dashboard" | "release-center" | "users" | "roles" | "permissions" | "dictionaries" | "menus";

type UserRow = {
  id: string;
  name: string;
  account: string;
  role: string;
  department: string;
  status: "enabled" | "disabled";
  lastLogin: string;
};

type RoleRow = {
  id: string;
  name: string;
  code: string;
  users: number;
  scope: string;
  permissions: string[];
  enabled: boolean;
};

type PermissionRow = {
  id: string;
  name: string;
  code: string;
  module: string;
  type: "menu" | "button" | "api";
  enabled: boolean;
};

type DictionaryRow = {
  id: string;
  group: string;
  key: string;
  label: string;
  value: string;
  sort: number;
  enabled: boolean;
};

type MenuRow = {
  id: string;
  title: string;
  path: string;
  icon: string;
  parent: string;
  sort: number;
  visible: boolean;
};

const navigation: Array<{
  group: string;
  items: Array<{ key: AdminPageKey; label: string; icon: typeof Home }>;
}> = [
  {
    group: "工作台",
    items: [
      { key: "dashboard", label: "首页", icon: LayoutDashboard },
      { key: "release-center", label: "App 发版中心", icon: Rocket },
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

const seedUsers: UserRow[] = [
  { id: "u_001", name: "发布管理员", account: "release.admin", role: "发版管理员", department: "平台工程", status: "enabled", lastLogin: "2026-06-02 13:46" },
  { id: "u_002", name: "测试负责人", account: "qa.lead", role: "测试负责人", department: "质量保障", status: "enabled", lastLogin: "2026-06-01 18:21" },
  { id: "u_003", name: "观察员", account: "release.viewer", role: "只读观察员", department: "运营支持", status: "disabled", lastLogin: "2026-05-28 09:12" },
];

const seedRoles: RoleRow[] = [
  { id: "r_001", name: "系统管理员", code: "system_admin", users: 2, scope: "全部数据", permissions: ["system:*", "release:*"], enabled: true },
  { id: "r_002", name: "发版管理员", code: "release_admin", users: 4, scope: "发版数据", permissions: ["release:write", "release:audit"], enabled: true },
  { id: "r_003", name: "只读观察员", code: "release_viewer", users: 8, scope: "只读数据", permissions: ["release:read"], enabled: true },
];

const seedPermissions: PermissionRow[] = [
  { id: "p_001", name: "查看首页", code: "dashboard:view", module: "工作台", type: "menu", enabled: true },
  { id: "p_002", name: "管理发布", code: "release:write", module: "App 发版中心", type: "button", enabled: true },
  { id: "p_003", name: "查看审计", code: "release:audit", module: "App 发版中心", type: "api", enabled: true },
  { id: "p_004", name: "维护用户", code: "system:user:write", module: "系统管理", type: "button", enabled: true },
  { id: "p_005", name: "维护菜单", code: "system:menu:write", module: "系统管理", type: "button", enabled: true },
];

const seedDictionaries: DictionaryRow[] = [
  { id: "d_001", group: "release_channel", key: "stable", label: "正式渠道", value: "stable", sort: 10, enabled: true },
  { id: "d_002", group: "release_channel", key: "beta", label: "Beta 渠道", value: "beta", sort: 20, enabled: true },
  { id: "d_003", group: "resource_package", key: "templates-bear", label: "打熊识图模板", value: "templates-bear", sort: 30, enabled: true },
  { id: "d_004", group: "audit_level", key: "critical", label: "关键操作", value: "critical", sort: 40, enabled: true },
];

const seedMenus: MenuRow[] = [
  { id: "m_001", title: "首页", path: "/dashboard", icon: "LayoutDashboard", parent: "-", sort: 10, visible: true },
  { id: "m_002", title: "App 发版中心", path: "/release-center", icon: "Rocket", parent: "工作台", sort: 20, visible: true },
  { id: "m_003", title: "用户管理", path: "/system/users", icon: "Users", parent: "系统管理", sort: 30, visible: true },
  { id: "m_004", title: "角色管理", path: "/system/roles", icon: "Shield", parent: "系统管理", sort: 40, visible: true },
  { id: "m_005", title: "数据字典", path: "/system/dictionaries", icon: "Database", parent: "系统管理", sort: 50, visible: true },
];

export function App() {
  const storedToken = getAppTokens().configToken;
  const [session, setSession] = useState(() => ({
    signedIn: import.meta.env.VITE_USE_MOCK === "true" || Boolean(storedToken),
    token: storedToken,
    name: storedToken ? "Release Admin" : "演示管理员",
  }));
  const [activePage, setActivePage] = useState<AdminPageKey>("dashboard");
  const [sidebarOpen, setSidebarOpen] = useState(false);

  if (!session.signedIn) {
    return <LoginPage onLogin={(token, name) => setSession({ signedIn: true, token, name })} />;
  }

  const currentLabel = navigation.flatMap((group) => group.items).find((item) => item.key === activePage)?.label ?? "首页";

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
          <div className="admin-content mx-auto max-w-[1600px] px-4 py-4 sm:px-6 lg:px-8">{renderPage(activePage, setActivePage)}</div>
        </section>
      </div>
    </main>
  );
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

function renderPage(activePage: AdminPageKey, setActivePage: (page: AdminPageKey) => void) {
  if (activePage === "release-center") return <AppReleasesPage />;
  if (activePage === "users") return <UsersPage />;
  if (activePage === "roles") return <RolesPage />;
  if (activePage === "permissions") return <PermissionsPage />;
  if (activePage === "dictionaries") return <DictionariesPage />;
  if (activePage === "menus") return <MenusPage />;
  return <DashboardPage onOpen={setActivePage} />;
}

function DashboardPage({ onOpen }: { onOpen: (page: AdminPageKey) => void }) {
  const quickEntries: Array<{ label: string; value: string; page: AdminPageKey; icon: typeof Home }> = [
    { label: "用户", value: String(seedUsers.length), page: "users", icon: Users },
    { label: "角色", value: String(seedRoles.length), page: "roles", icon: Shield },
    { label: "权限点", value: String(seedPermissions.length), page: "permissions", icon: KeyRound },
    { label: "字典项", value: String(seedDictionaries.length), page: "dictionaries", icon: Database },
  ];

  return (
    <div className="grid gap-4">
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
            <Badge>7 个菜单</Badge>
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
            {seedUsers.slice(0, 3).map((user) => (
              <div key={user.id} className="flex items-center justify-between gap-3 rounded-lg border p-2.5 text-xs">
                <div className="min-w-0">
                  <div className="truncate font-medium">{user.name}</div>
                  <div className="truncate text-muted-foreground">{user.account}</div>
                </div>
                <span className="shrink-0 text-muted-foreground">{user.lastLogin.slice(5)}</span>
              </div>
            ))}
          </div>
        </Card>
      </div>
    </div>
  );
}

function UsersPage() {
  const [users, setUsers] = useState(seedUsers);
  const [query, setQuery] = useState("");
  const rows = users.filter((user) => [user.name, user.account, user.role, user.department].join(" ").toLowerCase().includes(query.toLowerCase()));

  return (
    <SystemPageShell
      title="用户管理"
      actions={
        <Button onClick={() => setUsers([{ id: `u_${Date.now()}`, name: "新用户", account: "new.user", role: "只读观察员", department: "平台工程", status: "enabled", lastLogin: "-" }, ...users])}>
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
              <Td>{user.department}</Td>
              <Td>
                <Badge tone={user.status === "enabled" ? "success" : "warning"}>{user.status === "enabled" ? "启用" : "停用"}</Badge>
              </Td>
              <Td>{user.lastLogin}</Td>
              <Td>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() =>
                    setUsers((items) =>
                      items.map((item) => (item.id === user.id ? { ...item, status: item.status === "enabled" ? "disabled" : "enabled" } : item)),
                    )
                  }
                >
                  {user.status === "enabled" ? "停用" : "启用"}
                </Button>
              </Td>
            </tr>
          ))}
        </tbody>
      </Table>
    </SystemPageShell>
  );
}

function RolesPage() {
  const [roles, setRoles] = useState(seedRoles);
  return (
    <SystemPageShell
      title="角色管理"
      actions={
        <Button onClick={() => setRoles([{ id: `r_${Date.now()}`, name: "新角色", code: "custom_role", users: 0, scope: "自定义数据", permissions: [], enabled: true }, ...roles])}>
          <Shield className="mr-1.5 h-4 w-4" />
          新增角色
        </Button>
      }
    >
      <div className="grid gap-3 xl:grid-cols-3">
        {roles.map((role) => (
          <Card key={role.id}>
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <div className="truncate font-medium">{role.name}</div>
                <div className="mt-1 font-mono text-xs text-muted-foreground">{role.code}</div>
              </div>
              <Switch checked={role.enabled} onCheckedChange={(checked) => setRoles((items) => items.map((item) => (item.id === role.id ? { ...item, enabled: checked } : item)))} />
            </div>
            <div className="mt-3 grid gap-2 text-xs">
              <InfoRow label="用户数" value={`${role.users}`} />
              <InfoRow label="数据范围" value={role.scope} />
              <InfoRow label="权限" value={role.permissions.join(", ") || "-"} />
            </div>
          </Card>
        ))}
      </div>
    </SystemPageShell>
  );
}

function PermissionsPage() {
  const [permissions, setPermissions] = useState(seedPermissions);
  return (
    <SystemPageShell
      title="权限管理"
      actions={
        <Button onClick={() => setPermissions([{ id: `p_${Date.now()}`, name: "新权限", code: "custom:permission", module: "系统管理", type: "button", enabled: true }, ...permissions])}>
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
                  onClick={() => setPermissions((items) => items.map((item) => (item.id === permission.id ? { ...item, enabled: !item.enabled } : item)))}
                >
                  切换
                </Button>
              </Td>
            </tr>
          ))}
        </tbody>
      </Table>
    </SystemPageShell>
  );
}

function DictionariesPage() {
  const [items, setItems] = useState(seedDictionaries);
  const groups = useMemo(() => Array.from(new Set(items.map((item) => item.group))), [items]);
  const [activeGroup, setActiveGroup] = useState(seedDictionaries[0]?.group ?? "");
  const rows = items.filter((item) => item.group === activeGroup);

  return (
    <SystemPageShell
      title="数据字典"
      actions={
        <Button onClick={() => setItems([{ id: `d_${Date.now()}`, group: activeGroup || "custom_group", key: "custom_key", label: "新字典项", value: "custom", sort: items.length + 1, enabled: true }, ...items])}>
          <BookOpen className="mr-1.5 h-4 w-4" />
          新增字典
        </Button>
      }
    >
      <div className="grid gap-4 lg:grid-cols-[220px_minmax(0,1fr)]">
        <Card className="p-2">
          <div className="px-2 py-1.5 text-xs font-medium text-muted-foreground">字典分组</div>
          <div className="grid gap-1">
            {groups.map((group) => (
              <button
                key={group}
                className={cn("rounded-lg px-2.5 py-2 text-left text-xs font-medium", activeGroup === group ? "bg-black text-white" : "hover:bg-muted")}
                onClick={() => setActiveGroup(group)}
              >
                {group}
              </button>
            ))}
          </div>
        </Card>
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
                  <Button variant="secondary" size="sm" onClick={() => setItems((list) => list.map((row) => (row.id === item.id ? { ...row, enabled: !row.enabled } : row)))}>
                    切换
                  </Button>
                </Td>
              </tr>
            ))}
          </tbody>
        </Table>
      </div>
    </SystemPageShell>
  );
}

function MenusPage() {
  const [menus, setMenus] = useState(seedMenus);
  return (
    <SystemPageShell
      title="菜单编辑"
      actions={
        <Button onClick={() => setMenus([{ id: `m_${Date.now()}`, title: "新菜单", path: "/custom", icon: "Settings", parent: "系统管理", sort: menus.length * 10 + 10, visible: true }, ...menus])}>
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
                <Switch checked={menu.visible} onCheckedChange={(checked) => setMenus((items) => items.map((item) => (item.id === menu.id ? { ...item, visible: checked } : item)))} />
              </Td>
              <Td>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => setMenus((items) => items.map((item) => (item.id === menu.id ? { ...item, sort: item.sort - 1 } : item)).sort((left, right) => left.sort - right.sort))}
                >
                  上移
                </Button>
              </Td>
            </tr>
          ))}
        </tbody>
      </Table>
    </SystemPageShell>
  );
}

function SystemPageShell({ title, actions, children }: { title: string; actions?: ReactNode; children: ReactNode }) {
  return (
    <div className="grid gap-4">
      <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
        <div>
          <h1 className="text-lg font-semibold">{title}</h1>
          <div className="text-xs text-muted-foreground">System Administration</div>
        </div>
        <div className="flex flex-wrap gap-2">{actions}</div>
      </div>
      <Card>{children}</Card>
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
      <Button variant="secondary">
        <Save className="mr-1.5 h-4 w-4" />
        保存
      </Button>
    </div>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-3">
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span className="min-w-0 truncate text-right font-medium">{value}</span>
    </div>
  );
}
