import type { ReactNode } from "react";
import { useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { GitBranch, Hammer, KeyRound, RefreshCw, Save, Webhook } from "lucide-react";
import {
  createBuildCenterProject,
  getBuildCenterOverview,
  upsertBuildProfile,
  upsertCodeRepository,
  upsertWebhookRoute,
  type BuildCenterProject,
  type BuildProfile,
  type CodeRepository,
  type WebhookRoute,
} from "@/api/buildCenter";
import { getIntegrationCredentials, type IntegrationCredentialStatus } from "@/api/integrationCredentials";
import { PageHeader } from "@/components/layout/PageHeader";
import { ApiErrorState } from "@/components/stable/StableAdminComponents";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Switch } from "@/components/ui/Switch";
import { formatDateTime } from "@/lib/format";

const providerOptions = ["github", "gitea", "gitlab", "generic"];
const lifecycleOptions = ["active", "paused", "archived"];
const channelOptions = ["dev", "internal", "beta", "stable", "emergency"];
const stackOptions = ["generic", "node", "go", "android", "docker", "cloudflare"];
const buildTypeOptions = ["release", "debug", "web", "server", "worker", "docker"];
const actionOptions = ["all", "status", "fetch", "prepare", "build", "image", "upload", "verify"];
const eventOptions = ["push", "tag", "workflow_run", "*"];

export function IntegrationConfigPage() {
  const [projectKey, setProjectKey] = useState("release-center");
  const [projectName, setProjectName] = useState("Release Center");
  const [projectDescription, setProjectDescription] = useState("轻量发布中心");
  const [ownerAccount, setOwnerAccount] = useState("platform");
  const [lifecycleStatus, setLifecycleStatus] = useState("active");
  const [defaultChannel, setDefaultChannel] = useState("dev");

  const [repoProvider, setRepoProvider] = useState("github");
  const [repoURL, setRepoURL] = useState("https://github.com/liut-coder/release-center");
  const [repoFullName, setRepoFullName] = useState("liut-coder/release-center");
  const [repoDefaultRef, setRepoDefaultRef] = useState("main");
  const [credentialRef, setCredentialRef] = useState("github_token_release_center");
  const [webhookSecretRef, setWebhookSecretRef] = useState("github_webhook_release_center");
  const [webhookEnabled, setWebhookEnabled] = useState(false);
  const [triggerOnPush, setTriggerOnPush] = useState(false);
  const [triggerOnTag, setTriggerOnTag] = useState(false);

  const [profileKey, setProfileKey] = useState("default");
  const [profileName, setProfileName] = useState("默认构建");
  const [buildCenterProject, setBuildCenterProject] = useState("release-center");
  const [stackType, setStackType] = useState("node");
  const [buildType, setBuildType] = useState("release");
  const [configPath, setConfigPath] = useState("config/release-center.yml");
  const [sourceWorkdir, setSourceWorkdir] = useState("");
  const [profileDefaultRef, setProfileDefaultRef] = useState("main");
  const [defaultVersionName, setDefaultVersionName] = useState("0.1.0");
  const [defaultVersionCode, setDefaultVersionCode] = useState("1");
  const [profileChannel, setProfileChannel] = useState("dev");
  const [buildAction, setBuildAction] = useState("all");
  const [profileEnabled, setProfileEnabled] = useState(true);
  const [commandsText, setCommandsText] = useState(`{"all":["buildctl all release-center"]}`);
  const [artifactRulesText, setArtifactRulesText] = useState(`[{"name":"web-dist","type":"web_dist","path":"dist"}]`);

  const [routeRepositoryId, setRouteRepositoryId] = useState("");
  const [routeProfileKey, setRouteProfileKey] = useState("default");
  const [routeEventType, setRouteEventType] = useState("push");
  const [routeRefPattern, setRouteRefPattern] = useState("refs/heads/main");
  const [routeAction, setRouteAction] = useState("status");
  const [routeEnabled, setRouteEnabled] = useState(false);
  const [message, setMessage] = useState("集成配置已就绪。");
  const [messageTone, setMessageTone] = useState<"default" | "success" | "warning" | "danger">("default");

  const overviewQuery = useQuery({
    queryKey: ["integration-config-overview"],
    queryFn: getBuildCenterOverview,
  });
  const credentialsQuery = useQuery({
    queryKey: ["integration-credentials"],
    queryFn: getIntegrationCredentials,
  });

  const projects = overviewQuery.data?.projects ?? [];
  const credentials = credentialsQuery.data?.credentials ?? [];
  const selectedProject = useMemo(() => projects.find((project) => project.project_key === projectKey), [projectKey, projects]);
  const repositories = selectedProject?.repositories ?? [];
  const profiles = selectedProject?.build_profiles ?? [];
  const routes = selectedProject?.webhook_routes ?? [];
  const metrics = useMemo(() => integrationMetrics(projects), [projects]);
  const effectiveRepositoryId = routeRepositoryId || repositories[0]?.id || "";

  const createProjectMutation = useMutation({
    mutationFn: () =>
      createBuildCenterProject({
        project_key: projectKey.trim(),
        name: projectName.trim(),
        description: projectDescription.trim(),
        owner_account: ownerAccount.trim(),
        lifecycle_status: lifecycleStatus,
        default_channel: defaultChannel,
        metadata: { source: "admin_web", ui: "integration_config" },
      }),
    onSuccess: async (result) => {
      setMessage(result.message_zh || `构建项目已保存：${result.project.project_key}`);
      setMessageTone("success");
      await overviewQuery.refetch();
    },
    onError: (error) => showConfigError(error, "保存构建项目失败", setMessage, setMessageTone),
  });
  const repositoryMutation = useMutation({
    mutationFn: () =>
      upsertCodeRepository(projectKey, {
        provider: repoProvider,
        repo_url: repoURL.trim(),
        repo_full_name: repoFullName.trim(),
        default_ref: repoDefaultRef.trim(),
        credential_ref: credentialRef.trim(),
        webhook_secret_ref: webhookSecretRef.trim(),
        webhook_enabled: webhookEnabled,
        trigger_on_push: triggerOnPush,
        trigger_on_tag: triggerOnTag,
        metadata: { source: "admin_web", ui: "integration_config" },
      }),
    onSuccess: async (result) => {
      setRouteRepositoryId(result.repository.id);
      setMessage(result.message_zh || `代码仓库已保存：${result.repository.repo_full_name || result.repository.repo_url}`);
      setMessageTone("success");
      await overviewQuery.refetch();
    },
    onError: (error) => showConfigError(error, "保存代码仓库失败", setMessage, setMessageTone),
  });
  const profileMutation = useMutation({
    mutationFn: () =>
      upsertBuildProfile(projectKey, {
        profile_key: profileKey.trim(),
        name: profileName.trim(),
        build_center_project: buildCenterProject.trim(),
        stack_type: stackType,
        build_type: buildType,
        config_path: configPath.trim(),
        source_workdir: sourceWorkdir.trim(),
        default_ref: profileDefaultRef.trim(),
        default_version_name: defaultVersionName.trim(),
        default_version_code: Number(defaultVersionCode),
        default_channel: profileChannel,
        build_action: buildAction,
        commands: parseJSON(commandsText, {}),
        artifact_rules: parseJSON(artifactRulesText, []),
        enabled: profileEnabled,
        metadata: { source: "admin_web", ui: "integration_config" },
      }),
    onSuccess: async (result) => {
      setRouteProfileKey(result.build_profile.profile_key);
      setMessage(result.message_zh || `构建配置已保存：${result.build_profile.profile_key}`);
      setMessageTone("success");
      await overviewQuery.refetch();
    },
    onError: (error) => showConfigError(error, "保存构建配置失败", setMessage, setMessageTone),
  });
  const routeMutation = useMutation({
    mutationFn: () =>
      upsertWebhookRoute(projectKey, {
        repository_id: effectiveRepositoryId,
        profile_key: routeProfileKey.trim(),
        event_type: routeEventType,
        ref_pattern: routeRefPattern.trim(),
        action: routeAction,
        enabled: routeEnabled,
        metadata: { source: "admin_web", ui: "integration_config" },
      }),
    onSuccess: async (result) => {
      setMessage(result.message_zh || `Webhook 路由已保存：${result.webhook_route.event_type}`);
      setMessageTone("success");
      await overviewQuery.refetch();
    },
    onError: (error) => showConfigError(error, "保存 Webhook 路由失败", setMessage, setMessageTone),
  });

  const busy = createProjectMutation.isPending || repositoryMutation.isPending || profileMutation.isPending || routeMutation.isPending;
  const mutationError = createProjectMutation.error || repositoryMutation.error || profileMutation.error || routeMutation.error;

  return (
    <>
      <PageHeader title="集成配置">
        <Button variant="secondary" onClick={() => overviewQuery.refetch()} disabled={overviewQuery.isFetching}>
          <RefreshCw className="mr-2 h-4 w-4" />
          刷新
        </Button>
      </PageHeader>

      {overviewQuery.isError ? <ApiErrorState error={overviewQuery.error} title="集成配置读取失败" /> : null}
      {credentialsQuery.isError ? <ApiErrorState error={credentialsQuery.error} title="凭证引用读取失败" /> : null}
      {mutationError ? <ApiErrorState error={mutationError} title="集成配置保存失败" /> : null}

      <StatusMessage text={message} tone={messageTone} />

      <div className="my-4 grid gap-3 md:grid-cols-4">
        <Metric label="项目" value={metrics.projects} note={`${metrics.activeProjects} 个 active`} icon={GitBranch} />
        <Metric label="仓库" value={metrics.repositories} note={`${metrics.githubRepositories} 个 GitHub`} icon={GitBranch} />
        <Metric label="构建配置" value={metrics.profiles} note={`${metrics.enabledProfiles} 个启用`} icon={Hammer} />
        <Metric label="Webhook" value={metrics.routes} note={`${metrics.enabledRoutes} 条启用`} icon={Webhook} />
      </div>

      <div className="grid gap-4 xl:grid-cols-[420px_minmax(0,1fr)]">
        <div className="grid gap-4">
          <Card>
            <SectionTitle title="项目" badge={selectedProject?.lifecycle_status || lifecycleStatus} />
            <div className="grid gap-2">
              <Select label="当前项目" value={projectKey} onChange={setProjectKey} options={unique(["release-center", ...projects.map((project) => project.project_key)])} />
              <div className="grid grid-cols-2 gap-2">
                <Input placeholder="project key" value={projectKey} onChange={(event) => setProjectKey(event.target.value)} />
                <Input placeholder="显示名称" value={projectName} onChange={(event) => setProjectName(event.target.value)} />
              </div>
              <Input placeholder="description" value={projectDescription} onChange={(event) => setProjectDescription(event.target.value)} />
              <div className="grid grid-cols-2 gap-2">
                <Input placeholder="owner" value={ownerAccount} onChange={(event) => setOwnerAccount(event.target.value)} />
                <Select label="状态" value={lifecycleStatus} onChange={setLifecycleStatus} options={lifecycleOptions} />
              </div>
              <Select label="默认渠道" value={defaultChannel} onChange={setDefaultChannel} options={channelOptions} />
              <Button onClick={() => createProjectMutation.mutate()} disabled={busy || !projectKey.trim() || !projectName.trim()}>
                <Save className="mr-2 h-4 w-4" />
                保存项目
              </Button>
            </div>
          </Card>

          <Card>
            <SectionTitle title="Git 仓库" badge={repoProvider} />
            <div className="grid gap-2">
              <Select label="Provider" value={repoProvider} onChange={setRepoProvider} options={providerOptions} />
              <Input placeholder="repo url" value={repoURL} onChange={(event) => setRepoURL(event.target.value)} />
              <Input placeholder="repo full name" value={repoFullName} onChange={(event) => setRepoFullName(event.target.value)} />
              <Input placeholder="default ref" value={repoDefaultRef} onChange={(event) => setRepoDefaultRef(event.target.value)} />
              <Input placeholder="credential ref" value={credentialRef} onChange={(event) => setCredentialRef(event.target.value)} />
              <Input placeholder="webhook secret ref" value={webhookSecretRef} onChange={(event) => setWebhookSecretRef(event.target.value)} />
              <SwitchRow label="Webhook" checked={webhookEnabled} onChange={setWebhookEnabled} />
              <div className="grid grid-cols-2 gap-2">
                <SwitchRow label="Push" checked={triggerOnPush} onChange={setTriggerOnPush} />
                <SwitchRow label="Tag" checked={triggerOnTag} onChange={setTriggerOnTag} />
              </div>
              <Button onClick={() => repositoryMutation.mutate()} disabled={busy || !projectKey.trim() || !repoURL.trim()}>
                <Save className="mr-2 h-4 w-4" />
                保存仓库
              </Button>
            </div>
          </Card>

          <CredentialReferencePanel
            credentials={credentials}
            onUse={(credential) => {
              if (credential.provider === "github" && credential.kind === "access_token") {
                setCredentialRef(credential.ref);
                setRepoProvider("github");
                setMessage(`已选择仓库凭证引用：${credential.ref}`);
                setMessageTone("success");
                return;
              }
              if (credential.provider === "github" && credential.kind === "webhook_secret") {
                setWebhookSecretRef(credential.ref);
                setRepoProvider("github");
                setMessage(`已选择 Webhook Secret 引用：${credential.ref}`);
                setMessageTone("success");
                return;
              }
              setMessage(`${credential.ref} 用于部署中心 credential_ref 或 Worker 环境。`);
              setMessageTone(credential.configured ? "success" : "warning");
            }}
          />

          <Card>
            <SectionTitle title="构建 Profile" badge={profileEnabled ? "enabled" : "disabled"} />
            <div className="grid gap-2">
              <div className="grid grid-cols-2 gap-2">
                <Input placeholder="profile key" value={profileKey} onChange={(event) => setProfileKey(event.target.value)} />
                <Input placeholder="显示名称" value={profileName} onChange={(event) => setProfileName(event.target.value)} />
              </div>
              <Input placeholder="build center project" value={buildCenterProject} onChange={(event) => setBuildCenterProject(event.target.value)} />
              <div className="grid grid-cols-2 gap-2">
                <Select label="stack" value={stackType} onChange={setStackType} options={stackOptions} />
                <Select label="build type" value={buildType} onChange={setBuildType} options={buildTypeOptions} />
              </div>
              <div className="grid grid-cols-2 gap-2">
                <Input placeholder="config path" value={configPath} onChange={(event) => setConfigPath(event.target.value)} />
                <Input placeholder="source workdir" value={sourceWorkdir} onChange={(event) => setSourceWorkdir(event.target.value)} />
              </div>
              <div className="grid grid-cols-2 gap-2">
                <Input placeholder="default ref" value={profileDefaultRef} onChange={(event) => setProfileDefaultRef(event.target.value)} />
                <Select label="action" value={buildAction} onChange={setBuildAction} options={actionOptions} />
              </div>
              <div className="grid grid-cols-[minmax(0,1fr)_96px] gap-2">
                <Input placeholder="version name" value={defaultVersionName} onChange={(event) => setDefaultVersionName(event.target.value)} />
                <Input placeholder="code" type="number" value={defaultVersionCode} onChange={(event) => setDefaultVersionCode(event.target.value)} />
              </div>
              <Select label="渠道" value={profileChannel} onChange={setProfileChannel} options={channelOptions} />
              <TextArea value={commandsText} onChange={setCommandsText} placeholder="commands json" />
              <TextArea value={artifactRulesText} onChange={setArtifactRulesText} placeholder="artifact rules json" />
              <SwitchRow label="启用 Profile" checked={profileEnabled} onChange={setProfileEnabled} />
              <Button onClick={() => profileMutation.mutate()} disabled={busy || !profileKey.trim() || !buildCenterProject.trim()}>
                <Save className="mr-2 h-4 w-4" />
                保存 Profile
              </Button>
            </div>
          </Card>

          <Card>
            <SectionTitle title="Webhook Route" badge={routeEnabled ? "enabled" : "disabled"} />
            <div className="grid gap-2">
              <Select
                label="仓库"
                value={effectiveRepositoryId}
                onChange={setRouteRepositoryId}
                options={repositories.map((repository) => ({
                  value: repository.id,
                  label: repository.repo_full_name || repository.repo_url,
                }))}
                placeholder="暂无仓库"
              />
              <Select
                label="Profile"
                value={routeProfileKey}
                onChange={setRouteProfileKey}
                options={unique(["default", ...profiles.map((profile) => profile.profile_key)])}
              />
              <div className="grid grid-cols-2 gap-2">
                <Select label="事件" value={routeEventType} onChange={setRouteEventType} options={eventOptions} />
                <Select label="动作" value={routeAction} onChange={setRouteAction} options={actionOptions} />
              </div>
              <Input placeholder="ref pattern" value={routeRefPattern} onChange={(event) => setRouteRefPattern(event.target.value)} />
              <SwitchRow label="启用 Route" checked={routeEnabled} onChange={setRouteEnabled} />
              <Button onClick={() => routeMutation.mutate()} disabled={busy || !effectiveRepositoryId}>
                <Save className="mr-2 h-4 w-4" />
                保存 Route
              </Button>
            </div>
          </Card>
        </div>

        <div className="grid gap-4">
          <ProjectSummary project={selectedProject} />
          <EntityList title="代码仓库" items={repositories} renderItem={(repository) => <RepositoryItem repository={repository} />} />
          <EntityList title="构建 Profile" items={profiles} renderItem={(profile) => <ProfileItem profile={profile} />} />
          <EntityList title="Webhook Route" items={routes} renderItem={(route) => <RouteItem route={route} repositories={repositories} />} />
        </div>
      </div>
    </>
  );
}

function ProjectSummary({ project }: { project?: BuildCenterProject }) {
  return (
    <Card>
      <SectionTitle title="项目概览" badge={project?.project_key || "未选择"} />
      {project ? (
        <div className="grid gap-2 text-xs md:grid-cols-2">
          <Info label="名称" value={project.name} />
          <Info label="Owner" value={project.owner_account || "-"} />
          <Info label="状态" value={project.lifecycle_status || "-"} />
          <Info label="渠道" value={project.default_channel || "-"} />
          <Info label="更新时间" value={formatDateTime(project.updated_at)} />
          <Info label="描述" value={project.description || "-"} />
        </div>
      ) : (
        <EmptyBox text="暂无项目" />
      )}
    </Card>
  );
}

function EntityList<T>({ title, items, renderItem }: { title: string; items: T[]; renderItem: (item: T) => ReactNode }) {
  return (
    <Card>
      <SectionTitle title={title} badge={`${items.length} 条`} />
      <div className="grid gap-2">{items.length ? items.map(renderItem) : <EmptyBox text={`暂无${title}`} />}</div>
    </Card>
  );
}

function CredentialReferencePanel({ credentials, onUse }: { credentials: IntegrationCredentialStatus[]; onUse: (credential: IntegrationCredentialStatus) => void }) {
  return (
    <Card>
      <SectionTitle title="凭证引用" badge={`${credentials.filter((item) => item.configured).length}/${credentials.length} 已配置`} />
      <div className="grid gap-2">
        {credentials.map((credential) => (
          <div key={credential.ref} className="rounded-lg border p-3 text-xs">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <KeyRound className="h-3.5 w-3.5 text-muted-foreground" />
                  <span className="truncate font-medium">{credential.name}</span>
                </div>
                <div className="mt-1 truncate font-mono text-[11px] text-muted-foreground">{credential.ref}</div>
              </div>
              <Badge tone={credential.configured ? "success" : "warning"}>{credential.configured ? "已配置" : "待配置"}</Badge>
            </div>
            <div className="mt-2 grid gap-1">
              <Info label="Provider" value={credential.provider} />
              <Info label="用途" value={credential.usage} />
              <Info label="已命中" value={credential.configured_env_vars?.join(" / ") || "-"} />
            </div>
            <div className="mt-2 grid gap-1 rounded-md bg-muted p-2 font-mono text-[11px] text-muted-foreground">
              {credential.env_vars.map((envVar) => (
                <span key={envVar} className="break-all">
                  {envVar}
                </span>
              ))}
            </div>
            <Button className="mt-2 w-full" variant="secondary" size="sm" onClick={() => onUse(credential)}>
              使用引用
            </Button>
          </div>
        ))}
        {!credentials.length ? <EmptyBox text="暂无凭证引用" /> : null}
      </div>
    </Card>
  );
}

function RepositoryItem({ repository }: { repository: CodeRepository }) {
  return (
    <div className="rounded-lg border p-3 text-xs">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="truncate font-medium">{repository.repo_full_name || repository.repo_url}</div>
          <div className="mt-1 truncate font-mono text-[11px] text-muted-foreground">{repository.repo_url}</div>
        </div>
        <Badge tone={repository.webhook_enabled ? "success" : "warning"}>{repository.provider}</Badge>
      </div>
      <div className="mt-2 grid gap-1">
        <Info label="默认 ref" value={repository.default_ref || "-"} />
        <Info label="凭证" value={repository.credential_ref || "-"} />
        <Info label="触发" value={[repository.trigger_on_push ? "push" : "", repository.trigger_on_tag ? "tag" : ""].filter(Boolean).join(" / ") || "-"} />
      </div>
    </div>
  );
}

function ProfileItem({ profile }: { profile: BuildProfile }) {
  return (
    <div className="rounded-lg border p-3 text-xs">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="truncate font-medium">{profile.name}</div>
          <div className="mt-1 truncate font-mono text-[11px] text-muted-foreground">{profile.profile_key}</div>
        </div>
        <Badge tone={profile.enabled === false ? "warning" : "success"}>{profile.stack_type || "generic"}</Badge>
      </div>
      <div className="mt-2 grid gap-1">
        <Info label="项目" value={profile.build_center_project} />
        <Info label="动作" value={profile.build_action || "all"} />
        <Info label="版本" value={`${profile.default_version_name || "-"} / ${profile.default_channel || "-"}`} />
      </div>
    </div>
  );
}

function RouteItem({ route, repositories }: { route: WebhookRoute; repositories: CodeRepository[] }) {
  const repository = repositories.find((item) => item.id === route.repository_id);
  return (
    <div className="rounded-lg border p-3 text-xs">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="truncate font-medium">{route.event_type} / {route.ref_pattern}</div>
          <div className="mt-1 truncate text-muted-foreground">{repository?.repo_full_name || repository?.repo_url || route.repository_id}</div>
        </div>
        <Badge tone={route.enabled ? "success" : "warning"}>{route.action}</Badge>
      </div>
      <div className="mt-2 grid gap-1">
        <Info label="Profile" value={route.profile_key || route.build_profile_id || "default"} />
        <Info label="更新时间" value={formatDateTime(route.updated_at)} />
      </div>
    </div>
  );
}

function Metric({ label, value, note, icon: Icon }: { label: string; value: string | number; note: string; icon: typeof GitBranch }) {
  return (
    <div className="rounded-lg border bg-white p-3">
      <div className="flex items-center justify-between gap-3">
        <span className="text-xs text-muted-foreground">{label}</span>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </div>
      <div className="mt-3 truncate text-2xl font-semibold">{value}</div>
      <div className="mt-1 truncate text-xs text-muted-foreground">{note}</div>
    </div>
  );
}

function SectionTitle({ title, badge }: { title: string; badge: string }) {
  return (
    <div className="mb-3 flex items-center justify-between gap-3">
      <div className="font-medium">{title}</div>
      <Badge>{badge}</Badge>
    </div>
  );
}

function Select({
  label,
  value,
  onChange,
  options,
  placeholder,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: Array<string | { value: string; label: string }>;
  placeholder?: string;
}) {
  return (
    <select
      aria-label={label}
      className="h-9 w-full rounded-lg border bg-white px-3 text-xs outline-none transition focus:border-black focus:ring-2 focus:ring-black/10"
      value={value}
      onChange={(event) => onChange(event.target.value)}
    >
      {!options.length && placeholder ? <option value="">{placeholder}</option> : null}
      {options.map((option) => {
        const item = typeof option === "string" ? { value: option, label: option } : option;
        return (
          <option key={item.value} value={item.value}>
            {item.label}
          </option>
        );
      })}
    </select>
  );
}

function TextArea({ value, onChange, placeholder }: { value: string; onChange: (value: string) => void; placeholder: string }) {
  return (
    <textarea
      className="min-h-20 w-full rounded-lg border bg-white px-3 py-2 font-mono text-[11px] outline-none transition focus:border-black focus:ring-2 focus:ring-black/10"
      placeholder={placeholder}
      value={value}
      onChange={(event) => onChange(event.target.value)}
    />
  );
}

function SwitchRow({ label, checked, onChange }: { label: string; checked: boolean; onChange: (value: boolean) => void }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-lg border p-3">
      <div className="text-xs font-medium">{label}</div>
      <Switch checked={checked} onCheckedChange={onChange} aria-label={label} />
    </div>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-3">
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span className="break-all text-right font-medium">{value || "-"}</span>
    </div>
  );
}

function StatusMessage({ text, tone }: { text: string; tone: "default" | "success" | "warning" | "danger" }) {
  const className =
    tone === "success"
      ? "border-emerald-200 bg-emerald-50 text-emerald-800"
      : tone === "danger"
        ? "border-red-200 bg-red-50 text-red-700"
        : tone === "warning"
          ? "border-amber-200 bg-amber-50 text-amber-800"
          : "border-border bg-muted text-muted-foreground";
  return <div className={`rounded-lg border p-3 text-xs ${className}`}>{text}</div>;
}

function EmptyBox({ text }: { text: string }) {
  return <div className="rounded-lg border bg-muted p-4 text-sm text-muted-foreground">{text}</div>;
}

function integrationMetrics(projects: BuildCenterProject[]) {
  const repositories = projects.flatMap((project) => project.repositories ?? []);
  const profiles = projects.flatMap((project) => project.build_profiles ?? []);
  const routes = projects.flatMap((project) => project.webhook_routes ?? []);
  return {
    projects: projects.length,
    activeProjects: projects.filter((project) => project.lifecycle_status === "active").length,
    repositories: repositories.length,
    githubRepositories: repositories.filter((repository) => repository.provider === "github").length,
    profiles: profiles.length,
    enabledProfiles: profiles.filter((profile) => profile.enabled !== false).length,
    routes: routes.length,
    enabledRoutes: routes.filter((route) => route.enabled).length,
  };
}

function parseJSON(value: string, fallback: unknown) {
  if (!value.trim()) return fallback;
  return JSON.parse(value);
}

function unique(values: string[]) {
  return Array.from(new Set(values.filter(Boolean)));
}

function showConfigError(
  error: unknown,
  fallback: string,
  setMessage: (value: string) => void,
  setTone: (value: "default" | "success" | "warning" | "danger") => void,
) {
  setMessage(error instanceof Error ? error.message : fallback);
  setTone("danger");
}
