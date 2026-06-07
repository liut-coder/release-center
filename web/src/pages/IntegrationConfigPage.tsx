import type { ReactNode } from "react";
import { useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { CheckCircle2, Copy, ExternalLink, GitBranch, Hammer, KeyRound, RefreshCw, Save, Wand2, Webhook } from "lucide-react";
import { getAdminIdentity } from "@/api/client";
import {
  createBuildCenterProject,
  dryRunWebhookRoute,
  getBuildCenterOverview,
  upsertBuildProfile,
  upsertCodeRepository,
  upsertWebhookRoute,
  type BuildCenterProject,
  type BuildProfile,
  type CodeRepository,
  type WebhookRouteDryRunResponse,
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
import { hasPermission, missingPermissionText } from "@/lib/permissions";

const providerOptions = ["github", "gitea", "gitlab", "generic"];
const lifecycleOptions = ["active", "paused", "archived"];
const channelOptions = ["dev", "internal", "beta", "stable", "emergency"];
const stackOptions = ["generic", "node", "go", "android", "windows", "docker", "cloudflare"];
const buildTypeOptions = ["release", "debug", "web", "server", "worker", "windows", "docker"];
const actionOptions = ["all", "status", "fetch", "prepare", "build", "image", "upload", "verify"];
const eventOptions = ["push", "tag", "workflow_run", "*"];
const setupTemplates = [
  {
    key: "github_web",
    name: "GitHub Web 项目",
    provider: "github",
    stackType: "node",
    buildType: "web",
    buildAction: "all",
    profileKey: "web",
    profileName: "Web 构建",
    commands: { all: "buildctl all {{projectKey}}" },
    artifactRules: [{ name: "web-dist", type: "web_dist", path: "dist" }],
    routeAction: "all",
  },
  {
    key: "github_server",
    name: "GitHub Go 服务",
    provider: "github",
    stackType: "go",
    buildType: "server",
    buildAction: "all",
    profileKey: "server",
    profileName: "服务端构建",
    commands: { all: "buildctl all {{projectKey}}" },
    artifactRules: [{ name: "server-binary", type: "server_binary", path: "bin" }],
    routeAction: "all",
  },
  {
    key: "github_android",
    name: "GitHub Android",
    provider: "github",
    stackType: "android",
    buildType: "release",
    buildAction: "all",
    profileKey: "android",
    profileName: "Android 构建",
    commands: { all: "buildctl all {{projectKey}}" },
    artifactRules: [{ name: "apk", type: "apk", path: "app/build/outputs/apk" }],
    routeAction: "all",
  },
  {
    key: "github_windows",
    name: "GitHub Windows",
    provider: "github",
    stackType: "windows",
    buildType: "windows",
    buildAction: "all",
    profileKey: "windows",
    profileName: "Windows 构建",
    commands: { all: "powershell -ExecutionPolicy Bypass -File scripts\\package-windows-go.ps1" },
    artifactRules: [
      { name: "windows-exe", type: "windows_exe", path: "dist/windows/*.exe" },
      { name: "windows-archive", type: "windows_archive", path: "dist/windows/*.zip" },
    ],
    routeAction: "all",
  },
] as const;

export function IntegrationConfigPage() {
  const [setupTemplateKey, setSetupTemplateKey] = useState<(typeof setupTemplates)[number]["key"]>("github_web");
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

  const [profileKey, setProfileKey] = useState("web");
  const [profileName, setProfileName] = useState("Web 构建");
  const [buildCenterProject, setBuildCenterProject] = useState("release-center");
  const [stackType, setStackType] = useState("node");
  const [buildType, setBuildType] = useState("web");
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
  const [routeProfileKey, setRouteProfileKey] = useState("web");
  const [routeEventType, setRouteEventType] = useState("push");
  const [routeRefPattern, setRouteRefPattern] = useState("refs/heads/main");
  const [routeAction, setRouteAction] = useState("all");
  const [routeEnabled, setRouteEnabled] = useState(false);
  const [dryRunEventType, setDryRunEventType] = useState("push");
  const [dryRunRef, setDryRunRef] = useState("main");
  const [dryRunResult, setDryRunResult] = useState<WebhookRouteDryRunResponse>();
  const [message, setMessage] = useState("集成配置已就绪。");
  const [messageTone, setMessageTone] = useState<"default" | "success" | "warning" | "danger">("default");
  const role = getAdminIdentity().role;
  const canBuildWrite = hasPermission(role, "build:write");
  const canIntegrationWrite = hasPermission(role, "integration:write");
  const canQuickSetup = canBuildWrite && canIntegrationWrite;

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
  const selectedRepository = useMemo(
    () => repositories.find((repository) => repository.id === effectiveRepositoryId) ?? repositories[0],
    [effectiveRepositoryId, repositories],
  );
  const selectedProfile = useMemo(
    () => profiles.find((profile) => profile.profile_key === routeProfileKey || profile.profile_key === profileKey) ?? profiles[0],
    [profileKey, profiles, routeProfileKey],
  );
  const selectedRoute = useMemo(
    () =>
      routes.find((route) => route.repository_id === effectiveRepositoryId && (route.profile_key === routeProfileKey || route.profile_key === profileKey)) ??
      routes[0],
    [effectiveRepositoryId, profileKey, routeProfileKey, routes],
  );
  const selectedTemplate = setupTemplates.find((template) => template.key === setupTemplateKey) ?? setupTemplates[0];
  const webhookPayloadURL = webhookEndpointAbsolute(repoProvider);
  const webhookSettingsURL = repositoryWebhookSettingsURL(repoProvider, repoFullName || selectedRepository?.repo_full_name || repoURL);
  const closurePreview = useMemo(
    () =>
      buildIntegrationClosure({
        project: selectedProject,
        repository: selectedRepository,
        profile: selectedProfile,
        route: selectedRoute,
        credentialRef,
        webhookSecretRef,
        triggerOnPush,
        triggerOnTag,
        routeEnabled,
        webhookEnabled,
      }),
    [credentialRef, routeEnabled, selectedProfile, selectedProject, selectedRepository, selectedRoute, triggerOnPush, triggerOnTag, webhookEnabled, webhookSecretRef],
  );
  const derivedPreview = useMemo(
    () =>
      buildDerivedPreview({
        projectKey,
        repoProvider,
        repoURL,
        repoFullName,
        repoDefaultRef,
        credentialRef,
        webhookSecretRef,
        profileKey,
        stackType,
        buildAction,
        routeEventType,
        routeRefPattern,
        routeEnabled,
      }),
    [
      projectKey,
      repoProvider,
      repoURL,
      repoFullName,
      repoDefaultRef,
      credentialRef,
      webhookSecretRef,
      profileKey,
      stackType,
      buildAction,
      routeEventType,
      routeRefPattern,
      routeEnabled,
    ],
  );

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
  const quickSetupMutation = useMutation({
    mutationFn: async () => {
      const projectResp = await createBuildCenterProject({
        project_key: projectKey.trim(),
        name: projectName.trim(),
        description: projectDescription.trim(),
        owner_account: ownerAccount.trim(),
        lifecycle_status: lifecycleStatus,
        default_channel: defaultChannel,
        metadata: { source: "admin_web", ui: "integration_quick_setup", template: setupTemplateKey },
      });
      const repositoryResp = await upsertCodeRepository(projectKey, {
        provider: repoProvider,
        repo_url: repoURL.trim(),
        repo_full_name: repoFullName.trim(),
        default_ref: repoDefaultRef.trim(),
        credential_ref: credentialRef.trim(),
        webhook_secret_ref: webhookSecretRef.trim(),
        webhook_enabled: webhookEnabled,
        trigger_on_push: triggerOnPush,
        trigger_on_tag: triggerOnTag,
        metadata: { source: "admin_web", ui: "integration_quick_setup", template: setupTemplateKey },
      });
      const profileResp = await upsertBuildProfile(projectKey, {
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
        metadata: { source: "admin_web", ui: "integration_quick_setup", template: setupTemplateKey },
      });
      const routeResp = await upsertWebhookRoute(projectKey, {
        repository_id: repositoryResp.repository.id,
        profile_key: routeProfileKey.trim() || profileResp.build_profile.profile_key,
        event_type: routeEventType,
        ref_pattern: routeRefPattern.trim(),
        action: routeAction,
        enabled: routeEnabled,
        metadata: { source: "admin_web", ui: "integration_quick_setup", template: setupTemplateKey },
      });
      return { projectResp, repositoryResp, profileResp, routeResp };
    },
    onSuccess: async (result) => {
      setRouteRepositoryId(result.repositoryResp.repository.id);
      setRouteProfileKey(result.profileResp.build_profile.profile_key);
      setMessage(`已完成基础接入：${result.projectResp.project.project_key} / ${result.repositoryResp.repository.repo_full_name || result.repositoryResp.repository.repo_url}`);
      setMessageTone("success");
      await overviewQuery.refetch();
    },
    onError: (error) => showConfigError(error, "快速接入失败", setMessage, setMessageTone),
  });
  const dryRunMutation = useMutation({
    mutationFn: () =>
      dryRunWebhookRoute({
        provider: repoProvider,
        repository: repoFullName || selectedRepository?.repo_full_name || repoURL,
        event_type: dryRunEventType,
        ref: dryRunRef,
        sender: "admin-web",
      }),
    onSuccess: (result) => {
      setDryRunResult(result);
      setMessage(result.message_zh || "Webhook Route 试跑完成");
      setMessageTone(result.matches.some((match) => match.matched) ? "success" : "warning");
    },
    onError: (error) => showConfigError(error, "Webhook Route 试跑失败", setMessage, setMessageTone),
  });

  const busy = createProjectMutation.isPending || repositoryMutation.isPending || profileMutation.isPending || routeMutation.isPending || quickSetupMutation.isPending || dryRunMutation.isPending;
  const mutationError = createProjectMutation.error || repositoryMutation.error || profileMutation.error || routeMutation.error || quickSetupMutation.error || dryRunMutation.error;

  const applyTemplate = (templateKey: string) => {
    const template = setupTemplates.find((item) => item.key === templateKey) ?? setupTemplates[0];
    setSetupTemplateKey(template.key);
    setRepoProvider(template.provider);
    setStackType(template.stackType);
    setBuildType(template.buildType);
    setBuildAction(template.buildAction);
    setRouteAction(template.routeAction);
    setProfileKey(template.profileKey);
    setRouteProfileKey(template.profileKey);
    setProfileName(template.profileName);
    setBuildCenterProject(projectKey.trim() || inferProjectKey(repoFullName || repoURL) || "release-center");
    setConfigPath(`config/${projectKey.trim() || inferProjectKey(repoFullName || repoURL) || "release-center"}.yml`);
    setCommandsText(JSON.stringify(applyTemplateTokens(template.commands, projectKey.trim() || inferProjectKey(repoFullName || repoURL) || "release-center"), null, 2));
    setArtifactRulesText(JSON.stringify(template.artifactRules, null, 2));
    setMessage(`已套用模板：${template.name}`);
    setMessageTone("success");
  };

  const applyRepositoryURL = (value: string) => {
    setRepoURL(value);
    const parsed = parseRepositoryURL(value);
    if (!parsed) return;
    setRepoProvider(parsed.provider);
    setRepoFullName(parsed.fullName);
    const nextProjectKey = inferProjectKey(parsed.fullName);
    if (nextProjectKey) {
      setProjectKey(nextProjectKey);
      setBuildCenterProject(nextProjectKey);
      setConfigPath(`config/${nextProjectKey}.yml`);
      setProjectName(toTitle(nextProjectKey));
      setCredentialRef(`${parsed.provider}_token_${normalizeRef(nextProjectKey)}`);
      setWebhookSecretRef(`${parsed.provider}_webhook_${normalizeRef(nextProjectKey)}`);
      setCommandsText(JSON.stringify(applyTemplateTokens(selectedTemplate.commands, nextProjectKey), null, 2));
    }
  };

  const useCredentialBestMatch = (kind: "access_token" | "webhook_secret") => {
    const match = bestCredentialMatch(credentials, repoProvider, kind, projectKey);
    if (!match) {
      setMessage(kind === "access_token" ? "没有找到匹配的仓库访问凭证引用。" : "没有找到匹配的 Webhook Secret 引用。");
      setMessageTone("warning");
      return;
    }
    if (kind === "access_token") setCredentialRef(match.ref);
    if (kind === "webhook_secret") setWebhookSecretRef(match.ref);
    setMessage(`已自动选择${kind === "access_token" ? "仓库访问" : "Webhook Secret"}引用：${match.ref}`);
    setMessageTone(match.configured ? "success" : "warning");
  };

  const applyProject = (project: BuildCenterProject) => {
    setProjectKey(project.project_key);
    setProjectName(project.name);
    setProjectDescription(project.description || "");
    setOwnerAccount(project.owner_account || "");
    setLifecycleStatus(project.lifecycle_status || "active");
    setDefaultChannel(project.default_channel || "dev");
    setBuildCenterProject(project.project_key);
    setConfigPath(`config/${project.project_key}.yml`);
    setMessage(`已载入项目：${project.project_key}`);
    setMessageTone("success");
  };

  const applyRepository = (repository: CodeRepository) => {
    setRouteRepositoryId(repository.id);
    setRepoProvider(repository.provider || "generic");
    setRepoURL(repository.repo_url || "");
    setRepoFullName(repository.repo_full_name || "");
    setRepoDefaultRef(repository.default_ref || "main");
    setProfileDefaultRef(repository.default_ref || "main");
    setCredentialRef(repository.credential_ref || "");
    setWebhookSecretRef(repository.webhook_secret_ref || "");
    setWebhookEnabled(Boolean(repository.webhook_enabled));
    setTriggerOnPush(Boolean(repository.trigger_on_push));
    setTriggerOnTag(Boolean(repository.trigger_on_tag));
    setMessage(`已载入仓库：${repository.repo_full_name || repository.repo_url}`);
    setMessageTone("success");
  };

  const applyProfile = (profile: BuildProfile) => {
    setProfileKey(profile.profile_key);
    setRouteProfileKey(profile.profile_key);
    setProfileName(profile.name);
    setBuildCenterProject(profile.build_center_project);
    setStackType(profile.stack_type || "generic");
    setBuildType(profile.build_type || "release");
    setConfigPath(profile.config_path || "");
    setSourceWorkdir(profile.source_workdir || "");
    setProfileDefaultRef(profile.default_ref || "main");
    setDefaultVersionName(profile.default_version_name || "");
    setDefaultVersionCode(String(profile.default_version_code || 1));
    setProfileChannel(profile.default_channel || "dev");
    setBuildAction(profile.build_action || "all");
    setProfileEnabled(profile.enabled !== false);
    setCommandsText(JSON.stringify(profile.commands ?? {}, null, 2));
    setArtifactRulesText(JSON.stringify(profile.artifact_rules ?? [], null, 2));
    setMessage(`已载入 Profile：${profile.profile_key}`);
    setMessageTone("success");
  };

  const applyRoute = (route: WebhookRoute) => {
    setRouteRepositoryId(route.repository_id);
    setRouteProfileKey(route.profile_key || "default");
    setRouteEventType(route.event_type || "push");
    setRouteRefPattern(route.ref_pattern || "*");
    setRouteAction(route.action || "all");
    setRouteEnabled(Boolean(route.enabled));
    setMessage(`已载入 Webhook Route：${route.event_type} / ${route.ref_pattern}`);
    setMessageTone("success");
  };

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

      <Card className="my-4">
        <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
          <div className="grid gap-4">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <div className="flex items-center gap-2 text-base font-semibold">
                  <Wand2 className="h-4 w-4" />
                  快速接入向导
                </div>
                <div className="mt-1 text-xs text-muted-foreground">选择模板、粘贴仓库地址，系统会推导项目、Profile、Webhook Route 和凭证引用。</div>
              </div>
              <Button
                onClick={() => quickSetupMutation.mutate()}
                disabled={busy || !derivedPreview.ready || !canQuickSetup}
                title={!canQuickSetup ? missingPermissionText(canBuildWrite ? "integration:write" : "build:write") : undefined}
                className="min-w-32"
              >
                <Save className="mr-2 h-4 w-4" />
                一键保存接入
              </Button>
            </div>

            <div className="grid gap-3 md:grid-cols-3">
              {setupTemplates.map((template) => (
                <button
                  key={template.key}
                  className={`rounded-lg border p-3 text-left transition hover:bg-muted ${
                    setupTemplateKey === template.key ? "border-black bg-muted" : "bg-white"
                  }`}
                  type="button"
                  onClick={() => applyTemplate(template.key)}
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-sm font-medium">{template.name}</span>
                    {setupTemplateKey === template.key ? <CheckCircle2 className="h-4 w-4" /> : null}
                  </div>
                  <div className="mt-2 text-xs text-muted-foreground">{template.stackType} / {template.buildType} / {template.buildAction}</div>
                </button>
              ))}
            </div>

            <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_220px]">
              <div className="grid gap-2">
                <label className="text-xs font-medium text-muted-foreground">仓库地址</label>
                <Input
                  placeholder="https://github.com/owner/repo 或 git@github.com:owner/repo.git"
                  value={repoURL}
                  onChange={(event) => applyRepositoryURL(event.target.value)}
                />
              </div>
              <div className="grid gap-2">
                <label className="text-xs font-medium text-muted-foreground">默认分支</label>
                <Input value={repoDefaultRef} onChange={(event) => {
                  setRepoDefaultRef(event.target.value);
                  setProfileDefaultRef(event.target.value);
                  setRouteRefPattern(refPatternFromBranch(event.target.value, routeEventType));
                }} />
              </div>
            </div>

            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
              <QuickField label="项目" value={`${projectKey} / ${projectName}`} />
              <QuickField label="仓库" value={`${repoProvider}:${repoFullName || repoURL}`} />
              <QuickField label="Profile" value={`${profileKey} / ${stackType} / ${buildAction}`} />
              <QuickField label="Route" value={`${routeEventType} / ${routeRefPattern} / ${routeEnabled ? "enabled" : "disabled"}`} />
            </div>

            <div className="grid gap-3 lg:grid-cols-2">
              <div className="rounded-lg border p-3">
                <div className="flex items-center justify-between gap-3">
                  <div className="text-xs font-medium">凭证引用</div>
                  <div className="flex gap-2">
                    <Button type="button" size="sm" variant="secondary" onClick={() => useCredentialBestMatch("access_token")}>匹配 Token</Button>
                    <Button type="button" size="sm" variant="secondary" onClick={() => useCredentialBestMatch("webhook_secret")}>匹配 Secret</Button>
                  </div>
                </div>
                <div className="mt-3 grid gap-2 text-xs">
                  <Info label="仓库 Token" value={credentialRef || "-"} />
                  <Info label="Webhook Secret" value={webhookSecretRef || "-"} />
                  <Info label="已配置引用" value={`${credentials.filter((item) => item.configured).length}/${credentials.length}`} />
                </div>
              </div>
              <div className="rounded-lg border p-3">
                <div className="flex items-center justify-between gap-3">
                  <div className="text-xs font-medium">Webhook 配置</div>
                  <Button
                    type="button"
                    size="sm"
                    variant="secondary"
                    onClick={() => copyText(webhookEndpointFor(repoProvider))}
                  >
                    <Copy className="mr-1 h-3.5 w-3.5" />
                    复制地址
                  </Button>
                </div>
                <div className="mt-3 grid gap-2 text-xs">
                  <Info label="Payload URL" value={webhookEndpointFor(repoProvider)} />
                  <Info label="Content type" value="application/json" />
                  <Info label="启用策略" value={routeEnabled ? "Route 已启用" : "先保存为 disabled，真实 secret 配好后再启用"} />
                </div>
              </div>
            </div>
          </div>

          <SetupChecklist items={derivedPreview.items} ready={derivedPreview.ready} />
        </div>
      </Card>

      <IntegrationClosurePanel
        items={closurePreview.items}
        payloadURL={webhookPayloadURL}
        secretRef={webhookSecretRef || selectedRepository?.webhook_secret_ref || "-"}
        events={[triggerOnPush || selectedRepository?.trigger_on_push ? "push" : "", triggerOnTag || selectedRepository?.trigger_on_tag ? "tag" : ""].filter(Boolean)}
        refPattern={selectedRoute?.ref_pattern || routeRefPattern}
        routeEnabled={Boolean(selectedRoute?.enabled || routeEnabled)}
        settingsURL={webhookSettingsURL}
        onCopyPayload={() => {
          copyText(webhookPayloadURL);
          setMessage("Webhook Payload URL 已复制");
          setMessageTone("success");
        }}
        onCopySecret={() => {
          copyText(webhookSecretRef || selectedRepository?.webhook_secret_ref || "");
          setMessage("Webhook Secret 引用已复制");
          setMessageTone("success");
        }}
        onEnableRecommended={() => {
          setWebhookEnabled(true);
          setTriggerOnPush(true);
          setRouteEnabled(true);
          setRouteEventType("push");
          setRouteRefPattern(refPatternFromBranch(repoDefaultRef, "push"));
          setDryRunEventType("push");
          setDryRunRef(repoDefaultRef || "main");
          setMessage("已切换为推荐的 push webhook 接入配置，保存仓库和 Route 后生效。");
          setMessageTone("success");
        }}
        dryRunEventType={dryRunEventType}
        dryRunRef={dryRunRef}
        dryRunResult={dryRunResult}
        dryRunPending={dryRunMutation.isPending}
        canDryRun={canIntegrationWrite}
        onDryRunEventTypeChange={setDryRunEventType}
        onDryRunRefChange={setDryRunRef}
        onDryRun={() => dryRunMutation.mutate()}
      />

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
              <Button onClick={() => createProjectMutation.mutate()} disabled={busy || !projectKey.trim() || !projectName.trim() || !canBuildWrite} title={!canBuildWrite ? missingPermissionText("build:write") : undefined}>
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
              <Button onClick={() => repositoryMutation.mutate()} disabled={busy || !projectKey.trim() || !repoURL.trim() || !canIntegrationWrite} title={!canIntegrationWrite ? missingPermissionText("integration:write") : undefined}>
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
              <Button onClick={() => profileMutation.mutate()} disabled={busy || !profileKey.trim() || !buildCenterProject.trim() || !canBuildWrite} title={!canBuildWrite ? missingPermissionText("build:write") : undefined}>
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
                <Select
                  label="事件"
                  value={routeEventType}
                  onChange={(value) => {
                    setRouteEventType(value);
                    setRouteRefPattern(refPatternFromBranch(repoDefaultRef, value));
                  }}
                  options={eventOptions}
                />
                <Select label="动作" value={routeAction} onChange={setRouteAction} options={actionOptions} />
              </div>
              <Input placeholder="ref pattern" value={routeRefPattern} onChange={(event) => setRouteRefPattern(event.target.value)} />
              <SwitchRow label="启用 Route" checked={routeEnabled} onChange={setRouteEnabled} />
              <Button onClick={() => routeMutation.mutate()} disabled={busy || !effectiveRepositoryId || !canIntegrationWrite} title={!canIntegrationWrite ? missingPermissionText("integration:write") : undefined}>
                <Save className="mr-2 h-4 w-4" />
                保存 Route
              </Button>
            </div>
          </Card>
        </div>

        <div className="grid gap-4">
          <ProjectSummary project={selectedProject} onUse={applyProject} />
          <EntityList title="代码仓库" items={repositories} renderItem={(repository) => <RepositoryItem repository={repository} onUse={applyRepository} />} />
          <EntityList title="构建 Profile" items={profiles} renderItem={(profile) => <ProfileItem profile={profile} onUse={applyProfile} />} />
          <EntityList title="Webhook Route" items={routes} renderItem={(route) => <RouteItem route={route} repositories={repositories} onUse={applyRoute} />} />
        </div>
      </div>
    </>
  );
}

function ProjectSummary({ project, onUse }: { project?: BuildCenterProject; onUse: (project: BuildCenterProject) => void }) {
  return (
    <Card>
      <SectionTitle title="项目概览" badge={project?.project_key || "未选择"} />
      {project ? (
        <div className="grid gap-3">
          <div className="grid gap-2 text-xs md:grid-cols-2">
            <Info label="名称" value={project.name} />
            <Info label="Owner" value={project.owner_account || "-"} />
            <Info label="状态" value={project.lifecycle_status || "-"} />
            <Info label="渠道" value={project.default_channel || "-"} />
            <Info label="更新时间" value={formatDateTime(project.updated_at)} />
            <Info label="描述" value={project.description || "-"} />
          </div>
          <Button type="button" size="sm" variant="secondary" onClick={() => onUse(project)}>
            载入项目
          </Button>
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

function RepositoryItem({ repository, onUse }: { repository: CodeRepository; onUse: (repository: CodeRepository) => void }) {
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
      <Button type="button" className="mt-2 w-full" size="sm" variant="secondary" onClick={() => onUse(repository)}>
        载入编辑
      </Button>
    </div>
  );
}

function ProfileItem({ profile, onUse }: { profile: BuildProfile; onUse: (profile: BuildProfile) => void }) {
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
      <Button type="button" className="mt-2 w-full" size="sm" variant="secondary" onClick={() => onUse(profile)}>
        载入编辑
      </Button>
    </div>
  );
}

function RouteItem({ route, repositories, onUse }: { route: WebhookRoute; repositories: CodeRepository[]; onUse: (route: WebhookRoute) => void }) {
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
      <Button type="button" className="mt-2 w-full" size="sm" variant="secondary" onClick={() => onUse(route)}>
        载入编辑
      </Button>
    </div>
  );
}

function QuickField({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border bg-muted p-3">
      <div className="text-[11px] text-muted-foreground">{label}</div>
      <div className="mt-1 break-all text-xs font-medium">{value || "-"}</div>
    </div>
  );
}

function SetupChecklist({ items, ready }: { items: Array<{ label: string; ok: boolean; note: string }>; ready: boolean }) {
  return (
    <div className="rounded-lg border p-3">
      <div className="mb-3 flex items-center justify-between gap-3">
        <div className="font-medium">接入预检</div>
        <Badge tone={ready ? "success" : "warning"}>{ready ? "可保存" : "需补齐"}</Badge>
      </div>
      <div className="grid gap-2">
        {items.map((item) => (
          <div key={item.label} className="flex items-start gap-2 rounded-md border p-2 text-xs">
            <CheckCircle2 className={`mt-0.5 h-3.5 w-3.5 shrink-0 ${item.ok ? "text-emerald-600" : "text-muted-foreground"}`} />
            <div className="min-w-0">
              <div className="font-medium">{item.label}</div>
              <div className="mt-0.5 text-muted-foreground">{item.note}</div>
            </div>
          </div>
        ))}
      </div>
      <div className="mt-3 rounded-md bg-muted p-2 text-[11px] text-muted-foreground">
        Secret 和 Token 只保存引用名；真实值仍放在服务器或 Worker 环境变量中。
      </div>
    </div>
  );
}

function IntegrationClosurePanel({
  items,
  payloadURL,
  secretRef,
  events,
  refPattern,
  routeEnabled,
  settingsURL,
  dryRunEventType,
  dryRunRef,
  dryRunResult,
  dryRunPending,
  canDryRun,
  onCopyPayload,
  onCopySecret,
  onEnableRecommended,
  onDryRunEventTypeChange,
  onDryRunRefChange,
  onDryRun,
}: {
  items: Array<{ label: string; ok: boolean; note: string }>;
  payloadURL: string;
  secretRef: string;
  events: string[];
  refPattern: string;
  routeEnabled: boolean;
  settingsURL: string;
  dryRunEventType: string;
  dryRunRef: string;
  dryRunResult?: WebhookRouteDryRunResponse;
  dryRunPending: boolean;
  canDryRun: boolean;
  onCopyPayload: () => void;
  onCopySecret: () => void;
  onEnableRecommended: () => void;
  onDryRunEventTypeChange: (value: string) => void;
  onDryRunRefChange: (value: string) => void;
  onDryRun: () => void;
}) {
  const readyCount = items.filter((item) => item.ok).length;
  const allReady = readyCount === items.length;
  return (
    <Card className="my-4">
      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="grid gap-3">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <div className="flex items-center gap-2 font-semibold">
                <Webhook className="h-4 w-4" />
                外部接入闭环
              </div>
              <div className="mt-1 text-xs text-muted-foreground">保存配置后，按这里把外部仓库 webhook 配完，再用 push/tag 触发构建。</div>
            </div>
            <Badge tone={allReady ? "success" : "warning"}>{readyCount}/{items.length} 就绪</Badge>
          </div>
          <div className="grid gap-3 md:grid-cols-2">
            <WebhookSetupField label="Payload URL" value={payloadURL} onCopy={onCopyPayload} />
            <WebhookSetupField label="Secret 引用" value={secretRef} onCopy={onCopySecret} />
            <QuickField label="Content type" value="application/json" />
            <QuickField label="触发事件" value={events.length ? events.join(" / ") : "未启用"} />
            <QuickField label="Ref Pattern" value={refPattern || "-"} />
            <QuickField label="Route 状态" value={routeEnabled ? "enabled" : "disabled"} />
          </div>
          <div className="flex flex-wrap gap-2">
            <Button type="button" variant="secondary" onClick={onEnableRecommended}>
              <Webhook className="mr-2 h-4 w-4" />
              使用 push 推荐配置
            </Button>
            {settingsURL ? (
              <Button type="button" variant="secondary" onClick={() => window.open(settingsURL, "_blank", "noopener,noreferrer")}>
                <ExternalLink className="mr-2 h-4 w-4" />
                打开仓库 Webhooks
              </Button>
            ) : null}
          </div>
          <div className="rounded-lg border bg-muted p-3">
            <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
              <div>
                <div className="text-xs font-medium">Route 试跑</div>
                <div className="mt-1 text-[11px] text-muted-foreground">不创建构建任务，只验证当前事件和 ref 是否会命中已启用 route。</div>
              </div>
              <Button type="button" size="sm" onClick={onDryRun} disabled={dryRunPending || !canDryRun} title={!canDryRun ? missingPermissionText("integration:write") : undefined}>
                {dryRunPending ? "试跑中" : "试跑 Route"}
              </Button>
            </div>
            <div className="grid gap-2 md:grid-cols-[140px_minmax(0,1fr)]">
              <Select label="试跑事件" value={dryRunEventType} onChange={onDryRunEventTypeChange} options={eventOptions} />
              <Input placeholder="main / refs/heads/main / v1.0.0" value={dryRunRef} onChange={(event) => onDryRunRefChange(event.target.value)} />
            </div>
            {dryRunResult ? <DryRunResult result={dryRunResult} /> : null}
          </div>
        </div>
        <div className="grid gap-2">
          {items.map((item) => (
            <div key={item.label} className="flex items-start gap-2 rounded-lg border p-2 text-xs">
              <CheckCircle2 className={`mt-0.5 h-3.5 w-3.5 shrink-0 ${item.ok ? "text-emerald-600" : "text-muted-foreground"}`} />
              <div className="min-w-0">
                <div className="font-medium">{item.label}</div>
                <div className="mt-0.5 text-muted-foreground">{item.note}</div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </Card>
  );
}

function DryRunResult({ result }: { result: WebhookRouteDryRunResponse }) {
  return (
    <div className="mt-3 grid gap-2">
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-md border bg-white p-2 text-xs">
        <span className="font-medium">{result.message_zh || "试跑完成"}</span>
        <span className="font-mono text-muted-foreground">{result.event.event_type} / {result.event.ref}</span>
      </div>
      {result.matches.map((match) => (
        <div key={match.route.id} className="rounded-md border bg-white p-2 text-xs">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="font-medium">{match.route.project_key} / {match.route.profile_key}</div>
            <Badge tone={match.matched ? "success" : "warning"}>{match.matched ? "命中" : "未命中"}</Badge>
          </div>
          <div className="mt-2 grid gap-1">
            <Info label="Route" value={`${match.route.event_type} / ${match.route.ref_pattern} / ${match.route.action}`} />
            <Info label="事件检查" value={match.event_ok ? "通过" : "未通过"} />
            <Info label="分支检查" value={match.ref_ok ? "通过" : "未通过"} />
            <Info label="Build ref" value={match.build_ref || "-"} />
            {match.block_reason ? <Info label="原因" value={match.block_reason} /> : null}
          </div>
        </div>
      ))}
      {!result.matches.length ? <EmptyBox text="没有找到已启用的仓库 Webhook route。请先保存仓库、启用 Webhook，并保存启用状态的 Route。" /> : null}
    </div>
  );
}

function WebhookSetupField({ label, value, onCopy }: { label: string; value: string; onCopy: () => void }) {
  return (
    <div className="rounded-lg border bg-muted p-3">
      <div className="mb-1 text-[11px] text-muted-foreground">{label}</div>
      <div className="flex items-center gap-2">
        <span className="min-w-0 flex-1 break-all font-mono text-xs font-medium">{value || "-"}</span>
        <Button type="button" size="icon" variant="secondary" onClick={onCopy} title={`复制${label}`} disabled={!value || value === "-"}>
          <Copy className="h-3.5 w-3.5" />
        </Button>
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

function parseRepositoryURL(value: string) {
  const trimmed = value.trim();
  if (!trimmed) return null;
  const sshMatch = trimmed.match(/^git@([^:]+):(.+?)(?:\.git)?$/);
  if (sshMatch) {
    return {
      provider: providerFromHost(sshMatch[1]),
      fullName: sshMatch[2].replace(/\.git$/, ""),
    };
  }
  try {
    const url = new URL(trimmed);
    const path = url.pathname.replace(/^\/+/, "").replace(/\.git$/, "");
    if (!path.includes("/")) return null;
    return {
      provider: providerFromHost(url.hostname),
      fullName: path,
    };
  } catch {
    const plain = trimmed.replace(/\.git$/, "");
    if (!plain.includes("/")) return null;
    return { provider: "generic", fullName: plain };
  }
}

function providerFromHost(host: string) {
  const normalized = host.toLowerCase();
  if (normalized.includes("github")) return "github";
  if (normalized.includes("gitea")) return "gitea";
  if (normalized.includes("gitlab")) return "gitlab";
  return "generic";
}

function inferProjectKey(value: string) {
  const source = value.trim().split("/").filter(Boolean).pop() || "";
  return normalizeRef(source);
}

function normalizeRef(value: string) {
  return value
    .trim()
    .replace(/\.git$/, "")
    .replace(/([a-z0-9])([A-Z])/g, "$1-$2")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function toTitle(value: string) {
  return value
    .split("-")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function refPatternFromBranch(branch: string, eventType: string) {
  const normalized = branch.trim() || "main";
  if (eventType === "tag") return "refs/tags/*";
  if (eventType === "*") return "*";
  return normalized.startsWith("refs/") ? normalized : `refs/heads/${normalized}`;
}

function applyTemplateTokens(value: unknown, projectKey: string) {
  return JSON.parse(JSON.stringify(value).replace(/\{\{projectKey\}\}/g, projectKey));
}

function bestCredentialMatch(credentials: IntegrationCredentialStatus[], provider: string, kind: string, projectKey: string) {
  const candidates = credentials.filter((credential) => credential.provider === provider && credential.kind === kind);
  return (
    candidates.find((credential) => credential.configured && credential.ref.includes(projectKey)) ||
    candidates.find((credential) => credential.configured) ||
    candidates.find((credential) => credential.ref.includes(projectKey)) ||
    candidates[0]
  );
}

function webhookEndpointFor(provider: string) {
  if (provider === "gitea") return "/api/v1/webhooks/gitea";
  return "/api/v1/webhooks/github";
}

function webhookEndpointAbsolute(provider: string) {
  const path = webhookEndpointFor(provider);
  if (typeof window === "undefined") return path;
  return `${window.location.origin}${path}`;
}

function repositoryWebhookSettingsURL(provider: string, fullNameOrURL: string) {
  const parsed = parseRepositoryURL(fullNameOrURL);
  const fullName = parsed?.fullName || fullNameOrURL.trim().replace(/^https?:\/\/[^/]+\//, "").replace(/\.git$/, "");
  if (!fullName.includes("/")) return "";
  if (provider === "github") return `https://github.com/${fullName}/settings/hooks`;
  if (provider === "gitlab") return `https://gitlab.com/${fullName}/-/hooks`;
  return "";
}

function buildIntegrationClosure(input: {
  project?: BuildCenterProject;
  repository?: CodeRepository;
  profile?: BuildProfile;
  route?: WebhookRoute;
  credentialRef: string;
  webhookSecretRef: string;
  triggerOnPush: boolean;
  triggerOnTag: boolean;
  routeEnabled: boolean;
  webhookEnabled: boolean;
}) {
  const repositoryCredential = input.repository?.credential_ref || input.credentialRef;
  const repositorySecret = input.repository?.webhook_secret_ref || input.webhookSecretRef;
  const webhookEnabled = Boolean(input.repository?.webhook_enabled || input.webhookEnabled);
  const routeEnabled = Boolean(input.route?.enabled || input.routeEnabled);
  const hasTrigger = Boolean(input.repository?.trigger_on_push || input.repository?.trigger_on_tag || input.triggerOnPush || input.triggerOnTag);
  const items = [
    {
      label: "项目已保存",
      ok: Boolean(input.project?.id),
      note: input.project?.project_key || "先保存项目",
    },
    {
      label: "仓库已绑定",
      ok: Boolean(input.repository?.id),
      note: input.repository?.repo_full_name || input.repository?.repo_url || "先保存 Git 仓库",
    },
    {
      label: "构建 Profile 已保存",
      ok: Boolean(input.profile?.id),
      note: input.profile ? `${input.profile.profile_key} / ${input.profile.build_action || "all"}` : "先保存构建 Profile",
    },
    {
      label: "凭证引用已填写",
      ok: Boolean(repositoryCredential),
      note: repositoryCredential || "填写 credential_ref，仅保存引用名",
    },
    {
      label: "Webhook Secret 引用已填写",
      ok: Boolean(repositorySecret),
      note: repositorySecret || "填写 webhook_secret_ref，仅保存引用名",
    },
    {
      label: "仓库 Webhook 已启用",
      ok: webhookEnabled && hasTrigger,
      note: webhookEnabled ? (hasTrigger ? "push/tag 触发已配置" : "请选择 push 或 tag 触发") : "开启仓库 Webhook 开关",
    },
    {
      label: "Route 已启用",
      ok: routeEnabled && Boolean(input.route?.id),
      note: input.route ? `${input.route.event_type} / ${input.route.ref_pattern}` : "保存并启用 Webhook Route",
    },
  ];
  return { items };
}

function buildDerivedPreview(input: {
  projectKey: string;
  repoProvider: string;
  repoURL: string;
  repoFullName: string;
  repoDefaultRef: string;
  credentialRef: string;
  webhookSecretRef: string;
  profileKey: string;
  stackType: string;
  buildAction: string;
  routeEventType: string;
  routeRefPattern: string;
  routeEnabled: boolean;
}) {
  const items = [
    {
      label: "项目标识",
      ok: Boolean(input.projectKey.trim()),
      note: input.projectKey.trim() ? input.projectKey : "需要 project key",
    },
    {
      label: "仓库地址",
      ok: Boolean(input.repoURL.trim() && (input.repoFullName.trim() || parseRepositoryURL(input.repoURL))),
      note: input.repoFullName || "粘贴 Git URL 后自动解析 owner/repo",
    },
    {
      label: "构建模板",
      ok: Boolean(input.profileKey.trim() && input.stackType.trim() && input.buildAction.trim()),
      note: `${input.profileKey || "-"} / ${input.stackType || "-"} / ${input.buildAction || "-"}`,
    },
    {
      label: "默认分支",
      ok: Boolean(input.repoDefaultRef.trim()),
      note: input.repoDefaultRef || "需要默认分支",
    },
    {
      label: "凭证引用",
      ok: Boolean(input.credentialRef.trim() || input.webhookSecretRef.trim()),
      note: input.credentialRef || input.webhookSecretRef || "可先保存引用名，真实值放环境变量",
    },
    {
      label: "Webhook Route",
      ok: Boolean(input.routeEventType.trim() && input.routeRefPattern.trim()),
      note: `${input.routeEventType || "-"} / ${input.routeRefPattern || "-"} / ${input.routeEnabled ? "enabled" : "disabled"}`,
    },
  ];
  return { items, ready: items.slice(0, 4).every((item) => item.ok) };
}

function copyText(value: string) {
  if (typeof navigator !== "undefined" && navigator.clipboard) {
    void navigator.clipboard.writeText(value);
  }
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
