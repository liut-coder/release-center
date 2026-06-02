import { useEffect, useMemo, useState, type ChangeEvent } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  Clipboard,
  Download,
  FileArchive,
  FileText,
  History,
  PauseCircle,
  QrCode,
  RefreshCw,
  Rocket,
  Search,
  ShieldCheck,
  SlidersHorizontal,
  Smartphone,
  Upload,
  XCircle,
} from "lucide-react";
import {
  createApp,
  createAppRelease,
  createAppReleaseBuild,
  createAppResourceVersion,
  disableApp,
  enableApp,
  getAppReleases,
  pauseAppRelease,
  pauseAppResourceVersion,
  publishAppRelease,
  publishAppResourceVersion,
  recallAppRelease,
  rollbackAppRelease,
  rollbackAppResourceVersion,
  unpublishAppRelease,
  updateAppReleaseRollout,
  updateAppReleaseNotes,
  updateAppResourceRollout,
  updateAppResourceNotes,
  type AppBuildStatus,
  type AppInfo,
  type AppInstallation,
  type AppRelease,
  type AppReleaseAuditLog,
  type AppReleaseBuildJob,
  type AppReleaseStatus,
  type AppResourcePackage,
  type AppResourceVersion,
  type AppUpdateLevel,
  type AppUpgradeEvent,
  type QualityAlert,
  type ReleaseQualityMetric,
  type UpdateReleaseNotesPayload,
} from "@/api/appReleases";
import { MockBadge } from "@/components/common/MockBadge";
import { QrCodeSvg } from "@/components/common/QrCodeSvg";
import { PageHeader } from "@/components/layout/PageHeader";
import { ApiErrorState } from "@/components/stable/StableAdminComponents";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/Dialog";
import { Input } from "@/components/ui/Input";
import { Switch } from "@/components/ui/Switch";
import { Table, Td, Th } from "@/components/ui/Table";
import { cn } from "@/lib/cn";
import { formatDateTime } from "@/lib/format";

const channels = ["dev", "internal", "beta", "stable", "emergency"];
const resourceChannels = ["dev", "internal", "beta", "stable"];
const buildTypes = ["debug", "release"];
const updateLevels: AppUpdateLevel[] = ["normal", "recommended", "forced"];
const updateLevelOptions = updateLevels.map((level) => ({ value: level, label: updateLevelLabel(level) }));
const rolloutOptions = [5, 10, 20, 50, 100];
const defaultBuildVersionName = "0.1.0-dev.1";
const prereleaseChannels = ["dev", "internal", "beta"];
const releaseTargetTypes = [
  { value: "all", label: "全部" },
  { value: "user_group", label: "用户组" },
  { value: "user_id", label: "指定用户" },
  { value: "device_id", label: "指定设备" },
];
const channelFilters = ["全部", ...channels];
const tabs = [
  { key: "overview", label: "概览" },
  { key: "builds", label: "构建记录" },
  { key: "app", label: "App 发布" },
  { key: "resources", label: "资源增量" },
  { key: "devices", label: "设备版本" },
  { key: "stats", label: "升级统计" },
  { key: "events", label: "升级事件" },
  { key: "audit", label: "操作审计" },
] as const;

type ReleaseTab = (typeof tabs)[number]["key"];
type ReleaseAction = "publish" | "pause" | "recall" | "rollback" | "unpublish" | "rollout";
type ResourceAction = "publish-resource" | "pause-resource" | "rollback-resource" | "resource-rollout";
type PendingReleaseAction = { type: ReleaseAction; release: AppRelease } | null;
type PendingResourceAction = { type: ResourceAction; resource: AppResourceVersion } | null;
type RolloutTarget = { kind: "release"; release: AppRelease; value: string } | { kind: "resource"; resource: AppResourceVersion; value: string } | null;
type NotesTarget =
  | { kind: "release"; release: AppRelease; title: string; summary: string; markdown: string }
  | { kind: "resource"; resource: AppResourceVersion; title: string; summary: string; markdown: string }
  | null;
type BuildVersionSuggestion = {
  semanticCode?: number;
  minNextCode: number;
  recommendedCode: number;
  recommendedName: string;
  versionNameError: string;
  channelMismatchError: string;
};

export function AppReleasesPage() {
  const [activeTab, setActiveTab] = useState<ReleaseTab>("overview");
  const [channelFilter, setChannelFilter] = useState("全部");
  const [query, setQuery] = useState("");
  const [gitRef, setGitRef] = useState("main");
  const [buildType, setBuildType] = useState("debug");
  const [channel, setChannel] = useState("dev");
  const [versionName, setVersionName] = useState(defaultBuildVersionName);
  const [versionCode, setVersionCode] = useState("1");
  const [buildNumber, setBuildNumber] = useState("1");
  const [apiBaseUrl, setApiBaseUrl] = useState(defaultApiBaseUrl);
  const [releaseNotes, setReleaseNotes] = useState("");
  const [buildApkFile, setBuildApkFile] = useState<File | null>(null);
  const [releaseBuildId, setReleaseBuildId] = useState("");
  const [releaseChannel, setReleaseChannel] = useState("internal");
  const [releaseTitle, setReleaseTitle] = useState("测试版本更新");
  const [releaseSummary, setReleaseSummary] = useState("优化任务执行稳定性");
  const [releaseMarkdown, setReleaseMarkdown] = useState(defaultReleaseMarkdown);
  const [releaseUpdateLevel, setReleaseUpdateLevel] = useState<AppUpdateLevel>("normal");
  const [releaseRollout, setReleaseRollout] = useState("100");
  const [releaseTargetType, setReleaseTargetType] = useState("all");
  const [releaseTargetValue, setReleaseTargetValue] = useState("");
  const [releaseScheduleMode, setReleaseScheduleMode] = useState("now");
  const [releaseScheduledAt, setReleaseScheduledAt] = useState("");
  const [releaseMinCode, setReleaseMinCode] = useState("");
  const [blockOldVersions, setBlockOldVersions] = useState(false);
  const [resourceVersion, setResourceVersion] = useState(defaultResourceVersion());
  const [resourceChannel, setResourceChannel] = useState("dev");
  const [resourceTitle, setResourceTitle] = useState("运行资源更新");
  const [resourceSummary, setResourceSummary] = useState("优化识别模板和 OCR 关键词库");
  const [resourceMarkdown, setResourceMarkdown] = useState(defaultResourceMarkdown);
  const [resourceMinAppCode, setResourceMinAppCode] = useState("");
  const [resourceMaxAppCode, setResourceMaxAppCode] = useState("");
  const [resourceUpdateLevel, setResourceUpdateLevel] = useState<AppUpdateLevel>("normal");
  const [resourceRollout, setResourceRollout] = useState("100");
  const [resourceFiles, setResourceFiles] = useState<File[]>([]);
  const [appKey, setAppKey] = useState("game-helper-android");
  const [appName, setAppName] = useState("游戏助手");
  const [appPlatform, setAppPlatform] = useState("android");
  const [appPackageName, setAppPackageName] = useState("com.kingdomhelper.executor");
  const [appDescription, setAppDescription] = useState("");
  const [message, setMessage] = useState("");
  const [messageTone, setMessageTone] = useState<"default" | "success" | "warning" | "danger">("default");
  const [pendingAction, setPendingAction] = useState<PendingReleaseAction>(null);
  const [pendingResourceAction, setPendingResourceAction] = useState<PendingResourceAction>(null);
  const [rolloutTarget, setRolloutTarget] = useState<RolloutTarget>(null);
  const [notesTarget, setNotesTarget] = useState<NotesTarget>(null);

  const releasesQuery = useQuery({
    queryKey: ["app-releases"],
    queryFn: getAppReleases,
    refetchInterval: (query) => {
      const jobs = query.state.data?.build_jobs ?? [];
      return jobs.some((job) => job.status === "queued" || job.status === "running") ? 5_000 : false;
    },
  });
  const data = releasesQuery.data;
  const releases = useMemo(() => sortReleases(data?.releases ?? []), [data?.releases]);
  const latest = useMemo(
    () => data?.latest ?? releases.find((item) => item.is_latest && item.channel === "stable") ?? releases[0],
    [releases, data?.latest],
  );
  const stableRelease = useMemo(
    () => releases.find((item) => item.channel === "stable" && (item.is_latest || item.is_published || item.status === "released")),
    [releases],
  );
  const betaRelease = useMemo(
    () => releases.find((item) => item.channel === "beta" && (item.is_latest || item.is_published || item.status === "released" || item.status === "rolling_out")),
    [releases],
  );
  const buildJobs = useMemo(() => sortBuildJobs(data?.build_jobs ?? []), [data?.build_jobs]);
  const resourceVersions = useMemo(() => sortResources(data?.resource_versions ?? []), [data?.resource_versions]);
  const latestResource = useMemo(
    () => data?.latest_resource ?? resourceVersions.find((item) => item.status === "released" && item.channel === "stable") ?? resourceVersions[0],
    [data?.latest_resource, resourceVersions],
  );
  const installations = useMemo(() => sortInstallations(data?.installations ?? []), [data?.installations]);
  const upgradeEvents = useMemo(() => sortEvents([...(data?.upgrade_events ?? []), ...(data?.resource_update_events ?? [])]), [
    data?.resource_update_events,
    data?.upgrade_events,
  ]);
  const auditLogs = useMemo(() => sortAuditLogs(data?.audit_logs ?? []), [data?.audit_logs]);
  const qualityMetrics = useMemo(() => data?.quality_metrics ?? deriveQualityMetrics(data?.upgrade_events ?? [], data?.resource_update_events ?? []), [
    data?.quality_metrics,
    data?.resource_update_events,
    data?.upgrade_events,
  ]);
  const qualityAlerts = data?.quality_alerts ?? deriveQualityAlerts(qualityMetrics);
  const apps = data?.apps ?? [];
  const activeJob = buildJobs[0];
  const buildVersionSuggestion = useMemo(() => suggestBuildVersion(versionName, channel, releases, buildJobs), [
    buildJobs,
    channel,
    releases,
    versionName,
  ]);
  const successfulBuilds = useMemo(() => buildJobs.filter((job) => job.status === "success"), [buildJobs]);
  const selectedBuild = useMemo(
    () => successfulBuilds.find((job) => job.id === releaseBuildId) ?? successfulBuilds[0],
    [releaseBuildId, successfulBuilds],
  );
  const stats = useMemo(() => calculateStats(releases, resourceVersions, installations, upgradeEvents), [
    installations,
    releases,
    resourceVersions,
    upgradeEvents,
  ]);

  const filteredReleases = useMemo(() => {
    return releases.filter((release) => {
      const text =
        `${release.version_name} ${release.version_code} ${release.channel} ${release.git_commit} ${release.file_name} ${release.title}`.toLowerCase();
      return (channelFilter === "全部" || release.channel === channelFilter) && text.includes(query.toLowerCase());
    });
  }, [channelFilter, query, releases]);
  const buildValidation = validateBuildForm({
    gitRef,
    versionName,
    versionCode,
    buildNumber,
    apiBaseUrl,
    releases,
    buildJobs,
    buildVersionSuggestion,
  });
  const releaseValidation = validateReleaseForm({
    build: selectedBuild,
    title: releaseTitle,
    rollout: releaseRollout,
    minCode: releaseMinCode,
    targetType: releaseTargetType,
    targetValue: releaseTargetValue,
    scheduleMode: releaseScheduleMode,
    scheduledAt: releaseScheduledAt,
  });
  const resourceValidation = validateResourceForm({
    resourceVersion,
    title: resourceTitle,
    rollout: resourceRollout,
    minAppCode: resourceMinAppCode,
    maxAppCode: resourceMaxAppCode,
    files: resourceFiles,
    existingResources: resourceVersions,
  });

  useEffect(() => {
    if (!releases.length && !buildJobs.length) return;
    setVersionCode((current) => {
      const currentCode = Number(current);
      return !Number.isInteger(currentCode) || currentCode <= 0 || currentCode < buildVersionSuggestion.recommendedCode
        ? String(buildVersionSuggestion.recommendedCode)
        : current;
    });
  }, [buildJobs, releases, buildVersionSuggestion.recommendedCode]);

  useEffect(() => {
    if (!releases.length && !buildJobs.length) return;
    const nextBuildNumber = nextBuildNumberValue(buildJobs);
    setBuildNumber((current) => {
      const currentNumber = Number(current);
      return !Number.isInteger(currentNumber) || currentNumber <= 0 || currentNumber < nextBuildNumber ? String(nextBuildNumber) : current;
    });
  }, [buildJobs, releases.length]);

  useEffect(() => {
    if (!releases.length && !buildJobs.length) return;
    setVersionName((current) => (current === defaultBuildVersionName ? buildVersionSuggestion.recommendedName : current));
  }, [buildJobs.length, buildVersionSuggestion.recommendedName, releases.length]);

  useEffect(() => {
    if (!releaseBuildId && successfulBuilds[0]) {
      setReleaseBuildId(successfulBuilds[0].id);
    }
  }, [releaseBuildId, successfulBuilds]);

  useEffect(() => {
    if (!selectedBuild) return;
    setReleaseChannel((current) => current || selectedBuild.channel);
    setReleaseTitle((current) => current || `${selectedBuild.version_name} 更新`);
  }, [selectedBuild]);

  const buildMutation = useMutation({
    mutationFn: () => {
      if (buildValidation) throw new Error(buildValidation);
      return createAppReleaseBuild({
        git_ref: gitRef.trim(),
        build_type: buildType,
        channel,
        version_name: versionName.trim(),
        version_code: Number(versionCode),
        build_number: Number(buildNumber),
        api_base_url: normalizeUrl(apiBaseUrl),
        release_notes: releaseNotes.trim(),
        apk_file: buildApkFile,
      });
    },
    onSuccess: async (result) => {
      setMessage(`构建任务已创建：${result.job.id}`);
      setMessageTone("success");
      setActiveTab("builds");
      await releasesQuery.refetch();
    },
    onError: (error) => showError(error, "构建任务创建失败", setMessage, setMessageTone),
  });

  const appValidation = validateAppForm({ appKey, appName, appPlatform, appPackageName });
  const createAppMutation = useMutation({
    mutationFn: () => {
      if (appValidation) throw new Error(appValidation);
      return createApp({
        app_key: appKey.trim(),
        name: appName.trim(),
        platform: appPlatform.trim(),
        package_name: appPackageName.trim(),
        description: appDescription.trim(),
        enabled: true,
      });
    },
    onSuccess: async (result) => {
      setMessage(`应用已保存：${result.app.name}`);
      setMessageTone("success");
      await releasesQuery.refetch();
    },
    onError: (error) => showError(error, "保存应用失败", setMessage, setMessageTone),
  });
  const enableAppMutation = useMutation({
    mutationFn: (app: AppInfo) => enableApp(app.id),
    onSuccess: () => refetchWithMessage(releasesQuery.refetch, "应用已启用", setMessage, setMessageTone),
    onError: (error) => showError(error, "启用应用失败", setMessage, setMessageTone),
  });
  const disableAppMutation = useMutation({
    mutationFn: (app: AppInfo) => disableApp(app.id),
    onSuccess: () => refetchWithMessage(releasesQuery.refetch, "应用已停用", setMessage, setMessageTone),
    onError: (error) => showError(error, "停用应用失败", setMessage, setMessageTone),
  });

  const createReleaseMutation = useMutation({
    mutationFn: () => {
      if (releaseValidation) throw new Error(releaseValidation);
      return createAppRelease({
        build_id: selectedBuild?.id ?? releaseBuildId,
        channel: releaseChannel,
        title: releaseTitle.trim(),
        summary: releaseSummary.trim(),
        release_notes_markdown: releaseMarkdown.trim(),
        update_level: releaseUpdateLevel,
        rollout_percentage: Number(releaseRollout),
        target_type: releaseTargetType,
        target_value: releaseTargetType === "all" ? undefined : releaseTargetValue.trim(),
        min_supported_code: releaseMinCode ? Number(releaseMinCode) : undefined,
        block_old_versions: blockOldVersions,
        scheduled_at: releaseScheduleMode === "scheduled" ? new Date(releaseScheduledAt).toISOString() : undefined,
      });
    },
    onSuccess: async (result) => {
      setMessage(`发布草稿已创建：${result.release.title || result.release.version_name}`);
      setMessageTone("success");
      setActiveTab("app");
      await releasesQuery.refetch();
    },
    onError: (error) => showError(error, "创建发布失败", setMessage, setMessageTone),
  });

  const createResourceMutation = useMutation({
    mutationFn: () => {
      if (resourceValidation) throw new Error(resourceValidation);
      return createAppResourceVersion({
        resource_version: resourceVersion.trim(),
        channel: resourceChannel,
        title: resourceTitle.trim(),
        summary: resourceSummary.trim(),
        release_notes_markdown: resourceMarkdown.trim(),
        min_app_version_code: resourceMinAppCode ? Number(resourceMinAppCode) : undefined,
        max_app_version_code: resourceMaxAppCode ? Number(resourceMaxAppCode) : undefined,
        update_level: resourceUpdateLevel,
        rollout_percentage: Number(resourceRollout),
        files: resourceFiles,
      });
    },
    onSuccess: async (result) => {
      setMessage(`资源版本已创建：${result.resource_version.resource_version}`);
      setMessageTone("success");
      setActiveTab("resources");
      await releasesQuery.refetch();
    },
    onError: (error) => showError(error, "创建资源版本失败", setMessage, setMessageTone),
  });

  const publishMutation = useMutation({
    mutationFn: (release: AppRelease) => publishAppRelease(release.id, release.channel === "stable" ? "stable" : "latest"),
    onSuccess: () => refetchWithMessage(releasesQuery.refetch, "发布状态已更新", setMessage, setMessageTone),
    onError: (error) => showError(error, "发布失败", setMessage, setMessageTone),
  });

  const pauseMutation = useMutation({
    mutationFn: (release: AppRelease) => pauseAppRelease(release.id),
    onSuccess: () => refetchWithMessage(releasesQuery.refetch, "发布已暂停", setMessage, setMessageTone),
    onError: (error) => showError(error, "暂停失败", setMessage, setMessageTone),
  });

  const recallMutation = useMutation({
    mutationFn: (release: AppRelease) => recallAppRelease(release.id),
    onSuccess: () => refetchWithMessage(releasesQuery.refetch, "版本已撤回", setMessage, setMessageTone),
    onError: (error) => showError(error, "撤回失败", setMessage, setMessageTone),
  });

  const rollbackMutation = useMutation({
    mutationFn: (release: AppRelease) => rollbackAppRelease(release.id),
    onSuccess: () => refetchWithMessage(releasesQuery.refetch, "已回滚到选定版本", setMessage, setMessageTone),
    onError: (error) => showError(error, "回滚失败", setMessage, setMessageTone),
  });

  const unpublishMutation = useMutation({
    mutationFn: (release: AppRelease) => unpublishAppRelease(release.id),
    onSuccess: () => refetchWithMessage(releasesQuery.refetch, "版本已下架", setMessage, setMessageTone),
    onError: (error) => showError(error, "下架失败", setMessage, setMessageTone),
  });

  const releaseRolloutMutation = useMutation({
    mutationFn: ({ release, rolloutPercentage }: { release: AppRelease; rolloutPercentage: number }) =>
      updateAppReleaseRollout(release.id, rolloutPercentage),
    onSuccess: () => refetchWithMessage(releasesQuery.refetch, "灰度比例已同步", setMessage, setMessageTone),
    onError: (error) => showError(error, "调整灰度失败", setMessage, setMessageTone),
  });

  const releaseNotesMutation = useMutation({
    mutationFn: ({ release, payload }: { release: AppRelease; payload: UpdateReleaseNotesPayload }) =>
      updateAppReleaseNotes(release.id, payload),
    onSuccess: () => refetchWithMessage(releasesQuery.refetch, "更新说明已保存", setMessage, setMessageTone),
    onError: (error) => showError(error, "保存更新说明失败", setMessage, setMessageTone),
  });

  const publishResourceMutation = useMutation({
    mutationFn: (resource: AppResourceVersion) => publishAppResourceVersion(resource.id),
    onSuccess: () => refetchWithMessage(releasesQuery.refetch, "资源版本已发布", setMessage, setMessageTone),
    onError: (error) => showError(error, "资源发布失败", setMessage, setMessageTone),
  });

  const pauseResourceMutation = useMutation({
    mutationFn: (resource: AppResourceVersion) => pauseAppResourceVersion(resource.id),
    onSuccess: () => refetchWithMessage(releasesQuery.refetch, "资源版本已暂停", setMessage, setMessageTone),
    onError: (error) => showError(error, "资源暂停失败", setMessage, setMessageTone),
  });

  const rollbackResourceMutation = useMutation({
    mutationFn: (resource: AppResourceVersion) => rollbackAppResourceVersion(resource.id),
    onSuccess: () => refetchWithMessage(releasesQuery.refetch, "资源指针已回滚", setMessage, setMessageTone),
    onError: (error) => showError(error, "资源回滚失败", setMessage, setMessageTone),
  });

  const resourceRolloutMutation = useMutation({
    mutationFn: ({ resource, rolloutPercentage }: { resource: AppResourceVersion; rolloutPercentage: number }) =>
      updateAppResourceRollout(resource.id, rolloutPercentage),
    onSuccess: () => refetchWithMessage(releasesQuery.refetch, "资源灰度已同步", setMessage, setMessageTone),
    onError: (error) => showError(error, "调整资源灰度失败", setMessage, setMessageTone),
  });

  const resourceNotesMutation = useMutation({
    mutationFn: ({ resource, payload }: { resource: AppResourceVersion; payload: UpdateReleaseNotesPayload }) =>
      updateAppResourceNotes(resource.id, payload),
    onSuccess: () => refetchWithMessage(releasesQuery.refetch, "资源更新说明已保存", setMessage, setMessageTone),
    onError: (error) => showError(error, "保存资源更新说明失败", setMessage, setMessageTone),
  });

  const operationBusy =
    mutationIsPending(createAppMutation) ||
    mutationIsPending(enableAppMutation) ||
    mutationIsPending(disableAppMutation) ||
    mutationIsPending(publishMutation) ||
    mutationIsPending(pauseMutation) ||
    mutationIsPending(recallMutation) ||
    mutationIsPending(rollbackMutation) ||
    mutationIsPending(unpublishMutation) ||
    mutationIsPending(releaseRolloutMutation) ||
    mutationIsPending(releaseNotesMutation) ||
    mutationIsPending(publishResourceMutation) ||
    mutationIsPending(pauseResourceMutation) ||
    mutationIsPending(rollbackResourceMutation) ||
    mutationIsPending(resourceRolloutMutation) ||
    mutationIsPending(resourceNotesMutation);
  const mutationError =
    buildMutation.error ||
    createAppMutation.error ||
    enableAppMutation.error ||
    disableAppMutation.error ||
    createReleaseMutation.error ||
    createResourceMutation.error ||
    publishMutation.error ||
    pauseMutation.error ||
    recallMutation.error ||
    rollbackMutation.error ||
    unpublishMutation.error ||
    releaseRolloutMutation.error ||
    releaseNotesMutation.error ||
    publishResourceMutation.error ||
    pauseResourceMutation.error ||
    rollbackResourceMutation.error ||
    resourceRolloutMutation.error ||
    resourceNotesMutation.error;

  const confirmReleaseAction = () => {
    if (!pendingAction) return;
    const release = pendingAction.release;
    if (pendingAction.type === "publish") publishMutation.mutate(release);
    if (pendingAction.type === "pause") pauseMutation.mutate(release);
    if (pendingAction.type === "recall") recallMutation.mutate(release);
    if (pendingAction.type === "rollback") rollbackMutation.mutate(release);
    if (pendingAction.type === "unpublish") unpublishMutation.mutate(release);
    setPendingAction(null);
  };

  const confirmResourceAction = () => {
    if (!pendingResourceAction) return;
    const resource = pendingResourceAction.resource;
    if (pendingResourceAction.type === "publish-resource") publishResourceMutation.mutate(resource);
    if (pendingResourceAction.type === "pause-resource") pauseResourceMutation.mutate(resource);
    if (pendingResourceAction.type === "rollback-resource") rollbackResourceMutation.mutate(resource);
    setPendingResourceAction(null);
  };

  const confirmRolloutChange = () => {
    if (!rolloutTarget) return;
    const rolloutPercentage = Number(rolloutTarget.value);
    if (!Number.isInteger(rolloutPercentage) || rolloutPercentage < 1 || rolloutPercentage > 100) {
      setMessage("灰度比例必须在 1 到 100 之间");
      setMessageTone("danger");
      return;
    }
    if (rolloutTarget.kind === "release") {
      releaseRolloutMutation.mutate({ release: rolloutTarget.release, rolloutPercentage });
    } else {
      resourceRolloutMutation.mutate({ resource: rolloutTarget.resource, rolloutPercentage });
    }
    setRolloutTarget(null);
  };

  const confirmNotesChange = () => {
    if (!notesTarget) return;
    const payload = {
      title: notesTarget.title.trim(),
      summary: notesTarget.summary.trim(),
      release_notes_markdown: notesTarget.markdown.trim(),
    };
    const validation = validateNotesPayload(payload);
    if (validation) {
      setMessage(validation);
      setMessageTone("danger");
      return;
    }
    if (notesTarget.kind === "release") {
      releaseNotesMutation.mutate({ release: notesTarget.release, payload });
    } else {
      resourceNotesMutation.mutate({ resource: notesTarget.resource, payload });
    }
    setNotesTarget(null);
  };

  const onResourceFilesChange = (event: ChangeEvent<HTMLInputElement>) => {
    setResourceFiles(Array.from(event.target.files ?? []));
  };

  return (
    <>
      <PageHeader title="App 发布">
        <MockBadge show={data?._mock} />
        <Button variant="secondary" onClick={() => releasesQuery.refetch()} disabled={releasesQuery.isFetching}>
          <RefreshCw className="mr-2 h-4 w-4" />
          刷新
        </Button>
      </PageHeader>

      <TabBar activeTab={activeTab} onChange={setActiveTab} />

      {releasesQuery.isError ? <ApiErrorState error={releasesQuery.error} title="App 发布数据读取失败" /> : null}
      {mutationError ? <ApiErrorState error={mutationError} title="App 发布操作失败" /> : null}

      <div className="mb-4 grid gap-3 md:grid-cols-4">
        <Metric label="正式版本" value={stableRelease ? `${stableRelease.version_name} (${stableRelease.version_code})` : "未发布"} note={stableRelease?.channel || "stable"} />
        <Metric label="资源版本" value={latestResource?.resource_version ?? "-"} note={latestResource?.channel || "-"} />
        <Metric label="活跃设备" value={stats.activeDevices} note={`${stats.pendingDevices} 台待更新`} />
        <Metric label="升级成功率" value={`${stats.successRate}%`} note={`${stats.failedEvents} 个失败事件`} />
      </div>

      <div className="mb-4">
        <StatusMessage tone={messageTone} text={message || "发版中心已按 APK 整包、资源增量、设备回传和审计链路组织。"} />
      </div>

      {activeTab === "overview" ? (
        <OverviewPanel
          latest={latest}
          stableRelease={stableRelease}
          betaRelease={betaRelease}
          latestResource={latestResource}
          activeJob={activeJob}
          stats={stats}
          apps={apps}
          appKey={appKey}
          setAppKey={setAppKey}
          appName={appName}
          setAppName={setAppName}
          appPlatform={appPlatform}
          setAppPlatform={setAppPlatform}
          appPackageName={appPackageName}
          setAppPackageName={setAppPackageName}
          appDescription={appDescription}
          setAppDescription={setAppDescription}
          appValidation={appValidation}
          appPending={createAppMutation.isPending}
          appActionPending={enableAppMutation.isPending || disableAppMutation.isPending}
          onCreateApp={() => createAppMutation.mutate()}
          onEnableApp={(app) => enableAppMutation.mutate(app)}
          onDisableApp={(app) => disableAppMutation.mutate(app)}
          releases={releases}
          resourceVersions={resourceVersions}
          installations={installations}
          onCopy={(value) => {
            copyText(value)
              .then(() => {
                setMessage("链接已复制");
                setMessageTone("success");
              })
              .catch(() => {
                setMessage("复制失败，请手动复制链接");
                setMessageTone("warning");
              });
          }}
        />
      ) : null}

      {activeTab === "builds" ? (
        <BuildsPanel
          gitRef={gitRef}
          setGitRef={setGitRef}
          buildType={buildType}
          setBuildType={setBuildType}
          channel={channel}
          setChannel={setChannel}
          versionName={versionName}
          setVersionName={setVersionName}
          versionCode={versionCode}
          setVersionCode={setVersionCode}
          buildNumber={buildNumber}
          setBuildNumber={setBuildNumber}
          buildVersionSuggestion={buildVersionSuggestion}
          apiBaseUrl={apiBaseUrl}
          setApiBaseUrl={setApiBaseUrl}
          releaseNotes={releaseNotes}
          setReleaseNotes={setReleaseNotes}
          buildApkFile={buildApkFile}
          setBuildApkFile={setBuildApkFile}
          buildValidation={buildValidation}
          buildPending={buildMutation.isPending}
          onBuild={() => buildMutation.mutate()}
          activeJob={activeJob}
          buildJobs={buildJobs}
          onCreateRelease={(job) => {
            setReleaseBuildId(job.id);
            setReleaseChannel(job.channel);
            setReleaseTitle(`${job.version_name} 更新`);
            setActiveTab("app");
          }}
        />
      ) : null}

      {activeTab === "app" ? (
        <AppReleasePanel
          successfulBuilds={successfulBuilds}
          selectedBuild={selectedBuild}
          releaseBuildId={releaseBuildId}
          setReleaseBuildId={setReleaseBuildId}
          releaseChannel={releaseChannel}
          setReleaseChannel={setReleaseChannel}
          releaseTitle={releaseTitle}
          setReleaseTitle={setReleaseTitle}
          releaseSummary={releaseSummary}
          setReleaseSummary={setReleaseSummary}
          releaseMarkdown={releaseMarkdown}
          setReleaseMarkdown={setReleaseMarkdown}
          releaseUpdateLevel={releaseUpdateLevel}
          setReleaseUpdateLevel={setReleaseUpdateLevel}
          releaseRollout={releaseRollout}
          setReleaseRollout={setReleaseRollout}
          releaseTargetType={releaseTargetType}
          setReleaseTargetType={setReleaseTargetType}
          releaseTargetValue={releaseTargetValue}
          setReleaseTargetValue={setReleaseTargetValue}
          releaseScheduleMode={releaseScheduleMode}
          setReleaseScheduleMode={setReleaseScheduleMode}
          releaseScheduledAt={releaseScheduledAt}
          setReleaseScheduledAt={setReleaseScheduledAt}
          releaseMinCode={releaseMinCode}
          setReleaseMinCode={setReleaseMinCode}
          blockOldVersions={blockOldVersions}
          setBlockOldVersions={setBlockOldVersions}
          releaseValidation={releaseValidation}
          createPending={createReleaseMutation.isPending}
          onCreate={() => createReleaseMutation.mutate()}
          filteredReleases={filteredReleases}
          channelFilter={channelFilter}
          setChannelFilter={setChannelFilter}
          query={query}
          setQuery={setQuery}
          busy={operationBusy}
          onAction={(type, release) => setPendingAction({ type, release })}
          onAdjustRollout={(release) => setRolloutTarget({ kind: "release", release, value: String(release.rollout_percentage ?? 100) })}
          onEditNotes={(release) =>
            setNotesTarget({
              kind: "release",
              release,
              title: release.title || release.version_name,
              summary: release.summary || release.release_notes || "",
              markdown: release.release_notes_markdown || release.release_notes || "",
            })
          }
        />
      ) : null}

      {activeTab === "resources" ? (
        <ResourcePanel
          resourceVersion={resourceVersion}
          setResourceVersion={setResourceVersion}
          resourceChannel={resourceChannel}
          setResourceChannel={setResourceChannel}
          resourceTitle={resourceTitle}
          setResourceTitle={setResourceTitle}
          resourceSummary={resourceSummary}
          setResourceSummary={setResourceSummary}
          resourceMarkdown={resourceMarkdown}
          setResourceMarkdown={setResourceMarkdown}
          resourceMinAppCode={resourceMinAppCode}
          setResourceMinAppCode={setResourceMinAppCode}
          resourceMaxAppCode={resourceMaxAppCode}
          setResourceMaxAppCode={setResourceMaxAppCode}
          resourceUpdateLevel={resourceUpdateLevel}
          setResourceUpdateLevel={setResourceUpdateLevel}
          resourceRollout={resourceRollout}
          setResourceRollout={setResourceRollout}
          resourceFiles={resourceFiles}
          onResourceFilesChange={onResourceFilesChange}
          resourceValidation={resourceValidation}
          createPending={createResourceMutation.isPending}
          onCreate={() => createResourceMutation.mutate()}
          resources={resourceVersions}
          busy={operationBusy}
          onAction={(type, resource) => setPendingResourceAction({ type, resource })}
          onAdjustRollout={(resource) => setRolloutTarget({ kind: "resource", resource, value: String(resource.rollout_percentage ?? 100) })}
          onEditNotes={(resource) =>
            setNotesTarget({
              kind: "resource",
              resource,
              title: resource.title || resource.resource_version,
              summary: resource.summary || "",
              markdown: resource.release_notes_markdown || "",
            })
          }
        />
      ) : null}

      {activeTab === "devices" ? <DevicesPanel installations={installations} /> : null}
      {activeTab === "stats" ? (
        <UpgradeStatsPanel stats={stats} installations={installations} events={upgradeEvents} qualityMetrics={qualityMetrics} qualityAlerts={qualityAlerts} />
      ) : null}
      {activeTab === "events" ? <EventsPanel events={upgradeEvents} /> : null}
      {activeTab === "audit" ? <AuditPanel logs={auditLogs} /> : null}

      <ConfirmReleaseActionDialog
        action={pendingAction}
        busy={operationBusy}
        onOpenChange={(open) => {
          if (!open) setPendingAction(null);
        }}
        onConfirm={confirmReleaseAction}
      />
      <ConfirmResourceActionDialog
        action={pendingResourceAction}
        busy={operationBusy}
        onOpenChange={(open) => {
          if (!open) setPendingResourceAction(null);
        }}
        onConfirm={confirmResourceAction}
      />
      <RolloutDialog
        target={rolloutTarget}
        busy={operationBusy}
        onChange={setRolloutTarget}
        onConfirm={confirmRolloutChange}
      />
      <NotesDialog
        target={notesTarget}
        busy={operationBusy}
        onChange={setNotesTarget}
        onConfirm={confirmNotesChange}
      />
    </>
  );
}

function TabBar({ activeTab, onChange }: { activeTab: ReleaseTab; onChange: (tab: ReleaseTab) => void }) {
  return (
    <div className="mb-4 overflow-x-auto">
      <div className="inline-flex min-w-full gap-1 rounded-lg border bg-muted p-1">
        {tabs.map((tab) => (
          <button
            key={tab.key}
            type="button"
            className={cn(
              "h-8 shrink-0 rounded-md px-3 text-xs font-medium text-muted-foreground transition",
              activeTab === tab.key && "bg-white text-foreground shadow-sm",
            )}
            onClick={() => onChange(tab.key)}
          >
            {tab.label}
          </button>
        ))}
      </div>
    </div>
  );
}

function OverviewPanel({
  latest,
  stableRelease,
  betaRelease,
  latestResource,
  activeJob,
  stats,
  apps,
  appKey,
  setAppKey,
  appName,
  setAppName,
  appPlatform,
  setAppPlatform,
  appPackageName,
  setAppPackageName,
  appDescription,
  setAppDescription,
  appValidation,
  appPending,
  appActionPending,
  onCreateApp,
  onEnableApp,
  onDisableApp,
  releases,
  resourceVersions,
  installations,
  onCopy,
}: {
  latest?: AppRelease;
  stableRelease?: AppRelease;
  betaRelease?: AppRelease;
  latestResource?: AppResourceVersion;
  activeJob?: AppReleaseBuildJob;
  stats: ReturnType<typeof calculateStats>;
  apps: AppInfo[];
  appKey: string;
  setAppKey: (value: string) => void;
  appName: string;
  setAppName: (value: string) => void;
  appPlatform: string;
  setAppPlatform: (value: string) => void;
  appPackageName: string;
  setAppPackageName: (value: string) => void;
  appDescription: string;
  setAppDescription: (value: string) => void;
  appValidation: string;
  appPending: boolean;
  appActionPending: boolean;
  onCreateApp: () => void;
  onEnableApp: (app: AppInfo) => void;
  onDisableApp: (app: AppInfo) => void;
  releases: AppRelease[];
  resourceVersions: AppResourceVersion[];
  installations: AppInstallation[];
  onCopy: (value: string) => void;
}) {
  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
      <div className="grid gap-4">
        <AppsOverview
          apps={apps}
          latest={latest}
          stableRelease={stableRelease}
          betaRelease={betaRelease}
          latestResource={latestResource}
          installations={installations}
          appKey={appKey}
          setAppKey={setAppKey}
          appName={appName}
          setAppName={setAppName}
          appPlatform={appPlatform}
          setAppPlatform={setAppPlatform}
          appPackageName={appPackageName}
          setAppPackageName={setAppPackageName}
          appDescription={appDescription}
          setAppDescription={setAppDescription}
          appValidation={appValidation}
          appPending={appPending}
          appActionPending={appActionPending}
          onCreateApp={onCreateApp}
          onEnableApp={onEnableApp}
          onDisableApp={onDisableApp}
        />
        <Card>
          <div className="mb-4 flex items-center justify-between gap-3">
            <div className="font-medium">App 当前发布</div>
            {latest ? <Badge tone={releaseStatusTone(latest.status, latest.is_published)}>{releaseStatusLabel(latest.status, latest.is_published)}</Badge> : null}
          </div>
          {latest ? <ReleaseSummary release={latest} onCopy={onCopy} /> : <EmptyBox text="暂无 App 版本" />}
        </Card>
        <Card>
          <div className="mb-4 flex items-center justify-between gap-3">
            <div className="font-medium">启动页资源</div>
            {latestResource ? <Badge tone={statusTone(latestResource.status)}>{statusLabel(latestResource.status)}</Badge> : null}
          </div>
          {latestResource ? <ResourceSummary resource={latestResource} /> : <EmptyBox text="暂无资源版本" />}
        </Card>
      </div>
      <div className="grid gap-4">
        <Card>
          <div className="mb-4 flex items-center gap-2 font-medium">
            <SlidersHorizontal className="h-4 w-4" />
            发布健康
          </div>
          <div className="grid gap-2 text-xs">
            <Info label="已发布 App" value={String(releases.filter((item) => item.is_published || item.status === "released").length)} />
            <Info label="已发布资源" value={String(resourceVersions.filter((item) => item.status === "released").length)} />
            <Info label="强制更新" value={String(releases.filter((item) => item.update_level === "forced" || item.force_update).length)} />
            <Info label="待升级设备" value={String(stats.pendingDevices)} />
            <Info label="失败事件" value={String(stats.failedEvents)} />
          </div>
        </Card>
        <Card>
          <div className="mb-4 flex items-center justify-between gap-3">
            <div className="font-medium">最近构建</div>
            {activeJob ? <Badge tone={buildStatusTone(activeJob.status)}>{buildStatusLabel(activeJob.status)}</Badge> : null}
          </div>
          {activeJob ? <BuildJobPanel job={activeJob} compact /> : <EmptyBox text="暂无构建任务" />}
        </Card>
        <Card>
          <div className="mb-4 flex items-center gap-2 font-medium">
            <Smartphone className="h-4 w-4" />
            设备分布
          </div>
          <VersionDistribution installations={installations} />
        </Card>
      </div>
    </div>
  );
}

function AppsOverview({
  apps,
  latest,
  stableRelease,
  betaRelease,
  latestResource,
  installations,
  appKey,
  setAppKey,
  appName,
  setAppName,
  appPlatform,
  setAppPlatform,
  appPackageName,
  setAppPackageName,
  appDescription,
  setAppDescription,
  appValidation,
  appPending,
  appActionPending,
  onCreateApp,
  onEnableApp,
  onDisableApp,
}: {
  apps: AppInfo[];
  latest?: AppRelease;
  stableRelease?: AppRelease;
  betaRelease?: AppRelease;
  latestResource?: AppResourceVersion;
  installations: AppInstallation[];
  appKey: string;
  setAppKey: (value: string) => void;
  appName: string;
  setAppName: (value: string) => void;
  appPlatform: string;
  setAppPlatform: (value: string) => void;
  appPackageName: string;
  setAppPackageName: (value: string) => void;
  appDescription: string;
  setAppDescription: (value: string) => void;
  appValidation: string;
  appPending: boolean;
  appActionPending: boolean;
  onCreateApp: () => void;
  onEnableApp: (app: AppInfo) => void;
  onDisableApp: (app: AppInfo) => void;
}) {
  const rows = apps.length
    ? apps
    : [
        {
          id: "app-current",
          app_key: "game-helper-android",
          name: "游戏助手",
          platform: "android",
          package_name: latest?.package_name ?? "com.kingdomhelper.executor",
          active_devices: installations.length,
        },
      ];

  return (
    <Card>
      <div className="mb-4 flex items-center justify-between gap-3">
        <div className="font-medium">App 列表</div>
        <Badge>{rows.length} 个应用</Badge>
      </div>
      <div className="mb-4 grid gap-2 rounded-lg border bg-muted p-3 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_120px]">
        <Input placeholder="app_key" value={appKey} onChange={(event) => setAppKey(event.target.value)} />
        <Input placeholder="应用名称" value={appName} onChange={(event) => setAppName(event.target.value)} />
        <Input placeholder="平台" value={appPlatform} onChange={(event) => setAppPlatform(event.target.value)} />
        <Input className="md:col-span-2" placeholder="包名" value={appPackageName} onChange={(event) => setAppPackageName(event.target.value)} />
        <Button onClick={onCreateApp} disabled={Boolean(appValidation) || appPending}>
          {appPending ? "保存中" : "保存应用"}
        </Button>
        <Input className="md:col-span-3" placeholder="描述" value={appDescription} onChange={(event) => setAppDescription(event.target.value)} />
        {appValidation ? <div className="text-xs text-amber-700 md:col-span-3">{appValidation}</div> : null}
      </div>
      <Table>
        <thead>
          <tr>
            <Th>应用</Th>
            <Th>平台</Th>
            <Th>包名</Th>
            <Th>当前正式版</Th>
            <Th>当前测试版</Th>
            <Th>最新资源版</Th>
            <Th>活跃设备</Th>
            <Th>状态</Th>
            <Th>操作</Th>
          </tr>
        </thead>
        <tbody>
          {rows.map((app) => (
            <tr key={app.id || app.app_key}>
              <Td>{app.name}</Td>
              <Td>{app.platform}</Td>
              <Td>{app.package_name}</Td>
              <Td>{app.current_stable_release?.version_name ?? stableRelease?.version_name ?? "未发布"}</Td>
              <Td>{app.current_beta_release?.version_name ?? betaRelease?.version_name ?? latest?.version_name ?? "未发布"}</Td>
              <Td>{app.latest_resource_version?.resource_version ?? latestResource?.resource_version ?? "-"}</Td>
              <Td>{app.active_devices ?? installations.filter((item) => item.last_seen_at).length}</Td>
              <Td>
                <Badge tone={app.enabled === false ? "warning" : "success"}>{app.enabled === false ? "停用" : "启用"}</Badge>
              </Td>
              <Td>
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={appActionPending || app.id === "app-current"}
                  onClick={() => (app.enabled === false ? onEnableApp(app) : onDisableApp(app))}
                >
                  {app.enabled === false ? "启用" : "停用"}
                </Button>
              </Td>
            </tr>
          ))}
        </tbody>
      </Table>
    </Card>
  );
}

function BuildsPanel({
  gitRef,
  setGitRef,
  buildType,
  setBuildType,
  channel,
  setChannel,
  versionName,
  setVersionName,
  versionCode,
  setVersionCode,
  buildNumber,
  setBuildNumber,
  buildVersionSuggestion,
  apiBaseUrl,
  setApiBaseUrl,
  releaseNotes,
  setReleaseNotes,
  buildApkFile,
  setBuildApkFile,
  buildValidation,
  buildPending,
  onBuild,
  activeJob,
  buildJobs,
  onCreateRelease,
}: {
  gitRef: string;
  setGitRef: (value: string) => void;
  buildType: string;
  setBuildType: (value: string) => void;
  channel: string;
  setChannel: (value: string) => void;
  versionName: string;
  setVersionName: (value: string) => void;
  versionCode: string;
  setVersionCode: (value: string) => void;
  buildNumber: string;
  setBuildNumber: (value: string) => void;
  buildVersionSuggestion: BuildVersionSuggestion;
  apiBaseUrl: string;
  setApiBaseUrl: (value: string) => void;
  releaseNotes: string;
  setReleaseNotes: (value: string) => void;
  buildApkFile: File | null;
  setBuildApkFile: (value: File | null) => void;
  buildValidation: string;
  buildPending: boolean;
  onBuild: () => void;
  activeJob?: AppReleaseBuildJob;
  buildJobs: AppReleaseBuildJob[];
  onCreateRelease: (job: AppReleaseBuildJob) => void;
}) {
  return (
    <div className="grid gap-4 xl:grid-cols-[360px_minmax(0,1fr)]">
      <Card>
        <div className="mb-4 flex items-center gap-2 font-medium">
          <Rocket className="h-4 w-4" />
          构建新版本
        </div>
        <div className="grid gap-2">
          <Input placeholder="branch / tag / commit" value={gitRef} onChange={(event) => setGitRef(event.target.value)} />
          <div className="grid grid-cols-2 gap-2">
            <Select label="构建类型" value={buildType} onChange={setBuildType} options={buildTypes} />
            <Select label="渠道" value={channel} onChange={setChannel} options={channels} />
          </div>
          <div className="grid grid-cols-[minmax(0,1fr)_110px] gap-2">
            <Input placeholder="versionName" value={versionName} onChange={(event) => setVersionName(event.target.value)} />
            <Input placeholder="versionCode" type="number" value={versionCode} onChange={(event) => setVersionCode(event.target.value)} />
          </div>
          <Input placeholder="buildNumber" type="number" value={buildNumber} onChange={(event) => setBuildNumber(event.target.value)} />
          <VersionCodeHint
            suggestion={buildVersionSuggestion}
            currentCode={versionCode}
            currentName={versionName}
            onUseRecommended={() => {
              setVersionName(buildVersionSuggestion.recommendedName);
              setVersionCode(String(buildVersionSuggestion.recommendedCode));
            }}
          />
          <Input placeholder="apiBaseUrl" value={apiBaseUrl} onChange={(event) => setApiBaseUrl(event.target.value)} />
          <Input placeholder="构建备注" value={releaseNotes} onChange={(event) => setReleaseNotes(event.target.value)} />
          <Input
            type="file"
            accept=".apk,application/vnd.android.package-archive"
            onChange={(event) => setBuildApkFile(event.target.files?.[0] ?? null)}
          />
          {buildApkFile ? <div className="text-xs text-muted-foreground">{buildApkFile.name}</div> : null}
          {buildValidation ? <InlineWarning text={buildValidation} /> : null}
          <Button onClick={onBuild} disabled={Boolean(buildValidation) || buildPending}>
            <Rocket className="mr-2 h-4 w-4" />
            {buildPending ? "提交中" : "创建构建任务"}
          </Button>
        </div>
      </Card>
      <div className="grid gap-4">
        <Card>
          <div className="mb-4 flex items-center justify-between gap-3">
            <div className="font-medium">构建日志</div>
            {activeJob ? <Badge tone={buildStatusTone(activeJob.status)}>{buildStatusLabel(activeJob.status)}</Badge> : null}
          </div>
          {activeJob ? <BuildJobPanel job={activeJob} /> : <EmptyBox text="暂无构建任务" />}
        </Card>
        <Card>
          <div className="mb-4 flex items-center justify-between gap-3">
            <div className="font-medium">构建记录</div>
            <Badge>{buildJobs.length} 条</Badge>
          </div>
          <BuildJobsTable jobs={buildJobs} onCreateRelease={onCreateRelease} />
        </Card>
      </div>
    </div>
  );
}

function AppReleasePanel({
  successfulBuilds,
  selectedBuild,
  releaseBuildId,
  setReleaseBuildId,
  releaseChannel,
  setReleaseChannel,
  releaseTitle,
  setReleaseTitle,
  releaseSummary,
  setReleaseSummary,
  releaseMarkdown,
  setReleaseMarkdown,
  releaseUpdateLevel,
  setReleaseUpdateLevel,
  releaseRollout,
  setReleaseRollout,
  releaseTargetType,
  setReleaseTargetType,
  releaseTargetValue,
  setReleaseTargetValue,
  releaseScheduleMode,
  setReleaseScheduleMode,
  releaseScheduledAt,
  setReleaseScheduledAt,
  releaseMinCode,
  setReleaseMinCode,
  blockOldVersions,
  setBlockOldVersions,
  releaseValidation,
  createPending,
  onCreate,
  filteredReleases,
  channelFilter,
  setChannelFilter,
  query,
  setQuery,
  busy,
  onAction,
  onAdjustRollout,
  onEditNotes,
}: {
  successfulBuilds: AppReleaseBuildJob[];
  selectedBuild?: AppReleaseBuildJob;
  releaseBuildId: string;
  setReleaseBuildId: (value: string) => void;
  releaseChannel: string;
  setReleaseChannel: (value: string) => void;
  releaseTitle: string;
  setReleaseTitle: (value: string) => void;
  releaseSummary: string;
  setReleaseSummary: (value: string) => void;
  releaseMarkdown: string;
  setReleaseMarkdown: (value: string) => void;
  releaseUpdateLevel: AppUpdateLevel;
  setReleaseUpdateLevel: (value: AppUpdateLevel) => void;
  releaseRollout: string;
  setReleaseRollout: (value: string) => void;
  releaseTargetType: string;
  setReleaseTargetType: (value: string) => void;
  releaseTargetValue: string;
  setReleaseTargetValue: (value: string) => void;
  releaseScheduleMode: string;
  setReleaseScheduleMode: (value: string) => void;
  releaseScheduledAt: string;
  setReleaseScheduledAt: (value: string) => void;
  releaseMinCode: string;
  setReleaseMinCode: (value: string) => void;
  blockOldVersions: boolean;
  setBlockOldVersions: (value: boolean) => void;
  releaseValidation: string;
  createPending: boolean;
  onCreate: () => void;
  filteredReleases: AppRelease[];
  channelFilter: string;
  setChannelFilter: (value: string) => void;
  query: string;
  setQuery: (value: string) => void;
  busy: boolean;
  onAction: (type: ReleaseAction, release: AppRelease) => void;
  onAdjustRollout: (release: AppRelease) => void;
  onEditNotes: (release: AppRelease) => void;
}) {
  return (
    <div className="grid min-w-0 gap-4 xl:grid-cols-[420px_minmax(0,1fr)]">
      <Card>
        <div className="mb-4 flex items-center gap-2 font-medium">
          <FileText className="h-4 w-4" />
          创建 App 发布
        </div>
        <div className="grid gap-2">
          <Select
            label="选择构建"
            value={releaseBuildId}
            onChange={setReleaseBuildId}
            options={successfulBuilds.map((job) => ({
              value: job.id,
              label: `${job.version_name} build ${job.build_number ?? job.version_code} / ${job.channel}`,
            }))}
            placeholder="暂无成功构建"
          />
          {selectedBuild ? (
            <div className="rounded-lg border bg-muted p-3 text-xs">
              <Info
                label="构建"
                value={`${selectedBuild.version_name} / versionCode ${selectedBuild.version_code} / build ${selectedBuild.build_number ?? "-"}`}
              />
              <Info label="Git" value={`${selectedBuild.git_ref} @ ${selectedBuild.git_commit || "-"}`} />
              <Info label="文件" value={selectedBuild.file_name || selectedBuild.artifact_path || "-"} />
            </div>
          ) : null}
          <div className="grid grid-cols-2 gap-2">
            <Select label="发布渠道" value={releaseChannel} onChange={setReleaseChannel} options={channels} />
            <Select
              label="升级级别"
              value={releaseUpdateLevel}
              onChange={(value) => setReleaseUpdateLevel(value as AppUpdateLevel)}
              options={updateLevelOptions}
            />
          </div>
          <Input placeholder="标题" value={releaseTitle} onChange={(event) => setReleaseTitle(event.target.value)} />
          <Input placeholder="摘要" value={releaseSummary} onChange={(event) => setReleaseSummary(event.target.value)} />
          <MarkdownTextarea value={releaseMarkdown} onChange={setReleaseMarkdown} />
          <div className="grid grid-cols-2 gap-2">
            <Select label="灰度比例" value={releaseRollout} onChange={setReleaseRollout} options={rolloutOptions.map(String)} />
            <Input
              placeholder="minSupportedCode"
              type="number"
              value={releaseMinCode}
              onChange={(event) => setReleaseMinCode(event.target.value)}
            />
          </div>
          <div className="grid grid-cols-2 gap-2">
            <Select label="目标用户" value={releaseTargetType} onChange={setReleaseTargetType} options={releaseTargetTypes} />
            <Input
              placeholder={releaseTargetPlaceholder(releaseTargetType)}
              value={releaseTargetValue}
              onChange={(event) => setReleaseTargetValue(event.target.value)}
              disabled={releaseTargetType === "all"}
            />
          </div>
          <div className="grid grid-cols-2 gap-2">
            <Select
              label="发布时间"
              value={releaseScheduleMode}
              onChange={setReleaseScheduleMode}
              options={[
                { value: "now", label: "立即" },
                { value: "scheduled", label: "定时" },
              ]}
            />
            <Input
              aria-label="定时发布时间"
              type="datetime-local"
              value={releaseScheduledAt}
              onChange={(event) => setReleaseScheduledAt(event.target.value)}
              disabled={releaseScheduleMode !== "scheduled"}
            />
          </div>
          <ToggleRow label="阻止旧版本执行任务" checked={blockOldVersions} onCheckedChange={setBlockOldVersions} />
          {releaseValidation ? <InlineWarning text={releaseValidation} /> : null}
          <Button onClick={onCreate} disabled={Boolean(releaseValidation) || createPending}>
            <ShieldCheck className="mr-2 h-4 w-4" />
            {createPending ? "创建中" : "创建发布草稿"}
          </Button>
        </div>
      </Card>
      <Card>
        <div className="mb-4 grid min-w-0 gap-3 md:grid-cols-[160px_minmax(0,1fr)_auto]">
          <Select label="筛选渠道" value={channelFilter} onChange={setChannelFilter} options={channelFilters} />
          <div className="relative">
            <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              className="pl-9"
              placeholder="搜索版本 / commit / 文件名"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
          </div>
          <Badge>{filteredReleases.length} 个版本</Badge>
        </div>
        <div className="grid gap-3">
          {filteredReleases.map((release) => (
            <ReleaseRow
              key={release.id}
              release={release}
              busy={busy}
              onAction={(type) => onAction(type, release)}
              onAdjustRollout={() => onAdjustRollout(release)}
              onEditNotes={() => onEditNotes(release)}
            />
          ))}
          {!filteredReleases.length ? <EmptyBox text="当前筛选没有版本" /> : null}
        </div>
      </Card>
    </div>
  );
}

function ResourcePanel({
  resourceVersion,
  setResourceVersion,
  resourceChannel,
  setResourceChannel,
  resourceTitle,
  setResourceTitle,
  resourceSummary,
  setResourceSummary,
  resourceMarkdown,
  setResourceMarkdown,
  resourceMinAppCode,
  setResourceMinAppCode,
  resourceMaxAppCode,
  setResourceMaxAppCode,
  resourceUpdateLevel,
  setResourceUpdateLevel,
  resourceRollout,
  setResourceRollout,
  resourceFiles,
  onResourceFilesChange,
  resourceValidation,
  createPending,
  onCreate,
  resources,
  busy,
  onAction,
  onAdjustRollout,
  onEditNotes,
}: {
  resourceVersion: string;
  setResourceVersion: (value: string) => void;
  resourceChannel: string;
  setResourceChannel: (value: string) => void;
  resourceTitle: string;
  setResourceTitle: (value: string) => void;
  resourceSummary: string;
  setResourceSummary: (value: string) => void;
  resourceMarkdown: string;
  setResourceMarkdown: (value: string) => void;
  resourceMinAppCode: string;
  setResourceMinAppCode: (value: string) => void;
  resourceMaxAppCode: string;
  setResourceMaxAppCode: (value: string) => void;
  resourceUpdateLevel: AppUpdateLevel;
  setResourceUpdateLevel: (value: AppUpdateLevel) => void;
  resourceRollout: string;
  setResourceRollout: (value: string) => void;
  resourceFiles: File[];
  onResourceFilesChange: (event: ChangeEvent<HTMLInputElement>) => void;
  resourceValidation: string;
  createPending: boolean;
  onCreate: () => void;
  resources: AppResourceVersion[];
  busy: boolean;
  onAction: (type: ResourceAction, resource: AppResourceVersion) => void;
  onAdjustRollout: (resource: AppResourceVersion) => void;
  onEditNotes: (resource: AppResourceVersion) => void;
}) {
  return (
    <div className="grid min-w-0 gap-4 xl:grid-cols-[420px_minmax(0,1fr)]">
      <Card>
        <div className="mb-4 flex items-center gap-2 font-medium">
          <FileArchive className="h-4 w-4" />
          创建资源增量发布
        </div>
        <div className="grid gap-2">
          <div className="grid grid-cols-2 gap-2">
            <Input placeholder="resourceVersion" value={resourceVersion} onChange={(event) => setResourceVersion(event.target.value)} />
            <Select label="资源渠道" value={resourceChannel} onChange={setResourceChannel} options={resourceChannels} />
          </div>
          <div className="grid grid-cols-2 gap-2">
            <Select
              label="升级级别"
              value={resourceUpdateLevel}
              onChange={(value) => setResourceUpdateLevel(value as AppUpdateLevel)}
              options={updateLevelOptions}
            />
            <Select label="灰度比例" value={resourceRollout} onChange={setResourceRollout} options={rolloutOptions.map(String)} />
          </div>
          <Input placeholder="标题" value={resourceTitle} onChange={(event) => setResourceTitle(event.target.value)} />
          <Input placeholder="摘要" value={resourceSummary} onChange={(event) => setResourceSummary(event.target.value)} />
          <div className="grid grid-cols-2 gap-2">
            <Input
              placeholder="最低 App versionCode"
              type="number"
              value={resourceMinAppCode}
              onChange={(event) => setResourceMinAppCode(event.target.value)}
            />
            <Input
              placeholder="最高 App versionCode"
              type="number"
              value={resourceMaxAppCode}
              onChange={(event) => setResourceMaxAppCode(event.target.value)}
            />
          </div>
          <MarkdownTextarea value={resourceMarkdown} onChange={setResourceMarkdown} />
          <label className="flex min-h-24 cursor-pointer flex-col items-center justify-center rounded-lg border border-dashed bg-muted px-3 py-4 text-center text-xs text-muted-foreground transition hover:border-black hover:bg-white">
            <Upload className="mb-2 h-5 w-5" />
            上传资源 ZIP
            <input className="sr-only" type="file" multiple accept=".zip,application/zip" onChange={onResourceFilesChange} />
          </label>
          {resourceFiles.length ? (
            <div className="grid gap-1 rounded-lg border p-3 text-xs">
              {resourceFiles.map((file) => (
                <div key={`${file.name}-${file.size}`} className="flex justify-between gap-3">
                  <span className="min-w-0 truncate">{file.name}</span>
                  <span className="shrink-0 text-muted-foreground">{formatBytes(file.size)}</span>
                </div>
              ))}
            </div>
          ) : null}
          {resourceValidation ? <InlineWarning text={resourceValidation} /> : null}
          <Button onClick={onCreate} disabled={Boolean(resourceValidation) || createPending}>
            <Upload className="mr-2 h-4 w-4" />
            {createPending ? "创建中" : "创建资源版本"}
          </Button>
        </div>
      </Card>
      <Card>
        <div className="mb-4 flex items-center justify-between gap-3">
          <div className="font-medium">资源版本</div>
          <Badge>{resources.length} 个版本</Badge>
        </div>
        <div className="grid gap-3">
          {resources.map((resource) => (
            <ResourceRow
              key={resource.id}
              resource={resource}
              busy={busy}
              onAction={(type) => onAction(type, resource)}
              onAdjustRollout={() => onAdjustRollout(resource)}
              onEditNotes={() => onEditNotes(resource)}
            />
          ))}
          {!resources.length ? <EmptyBox text="暂无资源版本" /> : null}
        </div>
      </Card>
    </div>
  );
}

function DevicesPanel({ installations }: { installations: AppInstallation[] }) {
  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("全部");
  const filteredInstallations = installations.filter((item) => {
    const status = installationStatus(item);
    const text =
      `${item.device_name} ${item.device_id} ${item.user_id} ${item.installed_version} ${item.installed_code} ${item.resource_version} ${item.device_model}`.toLowerCase();
    return (statusFilter === "全部" || status === statusFilter) && text.includes(query.toLowerCase());
  });

  return (
    <Card>
      <div className="mb-4 grid gap-3 md:grid-cols-[minmax(0,1fr)_140px_auto]">
        <div className="font-medium">设备版本</div>
        <Select
          label="设备状态"
          value={statusFilter}
          onChange={setStatusFilter}
          options={["全部", "正常", "待更新", "资源激活失败", "未知"]}
        />
        <Badge>{filteredInstallations.length} / {installations.length} 台设备</Badge>
      </div>
      <div className="relative mb-4">
        <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
        <Input
          className="pl-9"
          placeholder="搜索设备 / 用户 / App 版本 / 资源版本"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
      </div>
      <Table>
        <thead>
          <tr>
            <Th>设备</Th>
            <Th>用户</Th>
            <Th>App 版本</Th>
            <Th>资源版本</Th>
            <Th>系统</Th>
            <Th>最近在线</Th>
            <Th>状态</Th>
          </tr>
        </thead>
        <tbody>
          {filteredInstallations.map((item) => (
            <tr key={item.id || item.device_id}>
              <Td>{item.device_name || item.device_id}</Td>
              <Td>{item.user_id ? `用户 ${item.user_id}` : "-"}</Td>
              <Td>{`${item.installed_version || "-"} / ${item.installed_code || "-"}`}</Td>
              <Td>{item.resource_version || "-"}</Td>
              <Td>{`${item.platform || "Android"} ${item.os_version || ""}`}</Td>
              <Td>{formatDateTime(item.last_seen_at)}</Td>
              <Td>
                <Badge tone={installationTone(item)}>{installationStatus(item)}</Badge>
              </Td>
            </tr>
          ))}
          {!filteredInstallations.length ? (
            <tr>
              <Td colSpan={7}>{installations.length ? "当前筛选没有设备" : "暂无设备安装记录"}</Td>
            </tr>
          ) : null}
        </tbody>
      </Table>
    </Card>
  );
}

function UpgradeStatsPanel({
  stats,
  installations,
  events,
  qualityMetrics,
  qualityAlerts,
}: {
  stats: ReturnType<typeof calculateStats>;
  installations: AppInstallation[];
  events: AppUpgradeEvent[];
  qualityMetrics: ReleaseQualityMetric[];
  qualityAlerts: QualityAlert[];
}) {
  const eventRows = countBy(events, (event) => eventLabel(event.event_type));
  const failureRows = countBy(
    events.filter((event) => eventTone(event.event_type) === "danger"),
    (event) => event.error_message || eventLabel(event.event_type),
  );
  const deviceRows = countBy(installations, installationStatus);

  return (
    <div className="grid gap-4 xl:grid-cols-3">
      {qualityAlerts.length ? <QualityAlertsPanel alerts={qualityAlerts} /> : null}
      <Card>
        <div className="mb-4 font-medium">升级概况</div>
        <div className="grid gap-2 text-xs">
          <Info label="升级成功率" value={`${stats.successRate}%`} />
          <Info label="活跃设备" value={String(stats.activeDevices)} />
          <Info label="待升级设备" value={String(stats.pendingDevices)} />
          <Info label="失败事件" value={String(stats.failedEvents)} />
        </div>
      </Card>
      {qualityMetrics.map((metric) => (
        <QualityMetricCard key={metric.category} metric={metric} />
      ))}
      <StatsList title="失败原因" rows={failureRows} emptyText="暂无失败事件" />
      <StatsList title="事件分布" rows={eventRows} emptyText="暂无升级事件" />
      <Card className="xl:col-span-3">
        <div className="mb-4 flex items-center justify-between gap-3">
          <div className="font-medium">设备状态分布</div>
          <Badge>{installations.length} 台设备</Badge>
        </div>
        <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-4">
          {deviceRows.map((row) => (
            <div key={row.label} className="rounded-lg border p-3">
              <div className="text-xs text-muted-foreground">{row.label}</div>
              <div className="mt-2 text-xl font-semibold">{row.count}</div>
              <div className="mt-2 h-2 rounded-full bg-muted">
                <div className="h-2 rounded-full bg-black" style={{ width: `${percentage(row.count, installations.length)}%` }} />
              </div>
            </div>
          ))}
          {!deviceRows.length ? <EmptyBox text="暂无设备数据" /> : null}
        </div>
      </Card>
    </div>
  );
}

function QualityAlertsPanel({ alerts }: { alerts: QualityAlert[] }) {
  return (
    <Card className="xl:col-span-3">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div className="font-medium">质量告警</div>
        <Badge tone="warning">{alerts.length} 条</Badge>
      </div>
      <div className="grid gap-2 md:grid-cols-2">
        {alerts.map((alert) => (
          <div key={alert.id} className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-900">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div className="font-medium">{qualityMetricLabel(alert.category)}</div>
              <Badge tone={alert.severity === "critical" ? "danger" : "warning"}>{qualityActionLabel(alert.recommended_action)}</Badge>
            </div>
            <div className="mt-2 text-amber-800">{alert.reason}</div>
            <div className="mt-2 grid gap-1 text-amber-800 md:grid-cols-3">
              <Info label="失败率" value={`${alert.failure_rate}%`} />
              <Info label="失败事件" value={String(alert.failure_events)} />
              <Info label="阈值" value={alert.threshold} />
            </div>
            {alert.latest_failure ? <div className="mt-2 truncate text-amber-800">{alert.latest_failure}</div> : null}
          </div>
        ))}
      </div>
    </Card>
  );
}

function QualityMetricCard({ metric }: { metric: ReleaseQualityMetric }) {
  const failureRows = (metric.failure_reasons ?? []).map((item) => ({ label: item.reason, count: item.count }));
  return (
    <Card>
      <div className="mb-4 flex items-center justify-between gap-3">
        <div className="font-medium">{qualityMetricLabel(metric.category)}</div>
        <Badge tone={metric.failure_events ? "danger" : "success"}>{metric.failure_rate}% 失败</Badge>
      </div>
      <div className="grid gap-2 text-xs">
        <Info label="成功率" value={`${metric.success_rate}%`} />
        <Info label="成功事件" value={String(metric.success_events)} />
        <Info label="失败事件" value={String(metric.failure_events)} />
        <Info label="总事件" value={String(metric.total_events)} />
      </div>
      <div className="mt-3 h-2 rounded-full bg-muted">
        <div
          className={cn("h-2 rounded-full", metric.failure_events ? "bg-red-600" : "bg-black")}
          style={{ width: `${Math.max(4, Math.min(100, metric.failure_rate || 0))}%` }}
        />
      </div>
      <div className="mt-3 rounded-lg border p-3 text-xs">
        <div className="flex items-center justify-between gap-3">
          <span className="text-muted-foreground">策略建议</span>
          <Badge tone={metric.recommended_action && metric.recommended_action !== "observe" ? "warning" : "success"}>
            {qualityActionLabel(metric.recommended_action)}
          </Badge>
        </div>
        <div className="mt-2 text-muted-foreground">{metric.action_reason || "指标未达到自动干预阈值"}</div>
        {metric.policy_threshold ? <div className="mt-1 text-muted-foreground">阈值：{metric.policy_threshold}</div> : null}
      </div>
      {metric.latest_failure_reason ? <div className="mt-3 truncate text-xs text-muted-foreground">{metric.latest_failure_reason}</div> : null}
      {failureRows.length ? <MiniStatsList rows={failureRows} /> : null}
    </Card>
  );
}

function MiniStatsList({ rows }: { rows: Array<{ label: string; count: number }> }) {
  const total = rows.reduce((sum, row) => sum + row.count, 0);
  return (
    <div className="mt-3 grid gap-2">
      {rows.slice(0, 3).map((row) => (
        <div key={row.label} className="grid grid-cols-[minmax(0,1fr)_32px] items-center gap-2 text-xs">
          <div className="min-w-0">
            <div className="truncate text-muted-foreground">{row.label}</div>
            <div className="mt-1 h-1.5 rounded-full bg-muted">
              <div className="h-1.5 rounded-full bg-black" style={{ width: `${percentage(row.count, total)}%` }} />
            </div>
          </div>
          <div className="text-right text-muted-foreground">{row.count}</div>
        </div>
      ))}
    </div>
  );
}

function StatsList({ title, rows, emptyText }: { title: string; rows: Array<{ label: string; count: number }>; emptyText: string }) {
  const total = rows.reduce((sum, row) => sum + row.count, 0);
  return (
    <Card>
      <div className="mb-4 flex items-center justify-between gap-3">
        <div className="font-medium">{title}</div>
        <Badge>{total} 条</Badge>
      </div>
      <div className="grid gap-2">
        {rows.map((row) => (
          <div key={row.label} className="grid grid-cols-[minmax(0,1fr)_40px] items-center gap-2 text-xs">
            <div className="min-w-0">
              <div className="truncate font-medium">{row.label}</div>
              <div className="mt-1 h-2 rounded-full bg-muted">
                <div className="h-2 rounded-full bg-black" style={{ width: `${percentage(row.count, total)}%` }} />
              </div>
            </div>
            <div className="text-right text-muted-foreground">{row.count}</div>
          </div>
        ))}
        {!rows.length ? <EmptyBox text={emptyText} /> : null}
      </div>
    </Card>
  );
}

function EventsPanel({ events }: { events: AppUpgradeEvent[] }) {
  const [query, setQuery] = useState("");
  const [categoryFilter, setCategoryFilter] = useState("全部");
  const filteredEvents = events.filter((event) => {
    const category = event.category || (event.package_key ? "resource" : "apk");
    const text =
      `${event.device_id} ${event.event_type} ${event.package_key} ${event.error_message} ${event.from_version} ${event.to_version} ${event.from_version_code} ${event.to_version_code}`.toLowerCase();
    return (categoryFilter === "全部" || category === categoryFilter) && text.includes(query.toLowerCase());
  });

  return (
    <Card>
      <div className="mb-4 grid gap-3 md:grid-cols-[minmax(0,1fr)_140px_auto]">
        <div className="font-medium">升级事件</div>
        <Select label="事件类型" value={categoryFilter} onChange={setCategoryFilter} options={["全部", "apk", "resource"]} />
        <Badge>{filteredEvents.length} / {events.length} 条</Badge>
      </div>
      <div className="relative mb-4">
        <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
        <Input
          className="pl-9"
          placeholder="搜索设备 / 事件 / 资源包 / 错误"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
      </div>
      <Table>
        <thead>
          <tr>
            <Th>时间</Th>
            <Th>设备</Th>
            <Th>类型</Th>
            <Th>版本变化</Th>
            <Th>资源包</Th>
            <Th>错误</Th>
          </tr>
        </thead>
        <tbody>
          {filteredEvents.map((event) => (
            <tr key={event.id}>
              <Td>{formatDateTime(event.created_at)}</Td>
              <Td>{event.device_id}</Td>
              <Td>
                <Badge tone={eventTone(event.event_type)}>{eventLabel(event.event_type)}</Badge>
              </Td>
              <Td>{eventVersionText(event)}</Td>
              <Td>{event.package_key || "-"}</Td>
              <Td>{event.error_message || "-"}</Td>
            </tr>
          ))}
          {!filteredEvents.length ? (
            <tr>
              <Td colSpan={6}>{events.length ? "当前筛选没有升级事件" : "暂无升级事件"}</Td>
            </tr>
          ) : null}
        </tbody>
      </Table>
    </Card>
  );
}

function AuditPanel({ logs }: { logs: AppReleaseAuditLog[] }) {
  const [query, setQuery] = useState("");
  const autoPauseLogs = logs.filter((log) => log.action === "resource.auto_pause");
  const filteredLogs = logs.filter((log) =>
    `${log.operator} ${log.action} ${log.target_type} ${log.target_id} ${auditSummary(log)}`.toLowerCase().includes(query.toLowerCase()),
  );

  return (
    <div className="grid gap-4">
      {autoPauseLogs.length ? <AutoPauseAuditSummary logs={autoPauseLogs} /> : null}
      <Card>
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <div className="font-medium">操作审计</div>
          <Badge>{filteredLogs.length} / {logs.length} 条</Badge>
        </div>
        <div className="relative mb-4">
          <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            className="pl-9"
            placeholder="搜索操作人 / 动作 / 对象"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
          />
        </div>
        <Table>
          <thead>
            <tr>
              <Th>时间</Th>
              <Th>操作人</Th>
              <Th>动作</Th>
              <Th>对象</Th>
              <Th>摘要</Th>
            </tr>
          </thead>
          <tbody>
            {filteredLogs.map((log) => (
              <tr key={log.id}>
                <Td>{formatDateTime(log.created_at)}</Td>
                <Td>{log.operator}</Td>
                <Td>
                  <Badge tone={log.action === "resource.auto_pause" ? "danger" : "default"}>{auditActionLabel(log.action)}</Badge>
                </Td>
                <Td>{`${log.target_type}${log.target_id ? ` #${log.target_id}` : ""}`}</Td>
                <Td>{auditSummary(log)}</Td>
              </tr>
            ))}
            {!filteredLogs.length ? (
              <tr>
                <Td colSpan={5}>{logs.length ? "当前筛选没有审计记录" : "暂无操作审计"}</Td>
              </tr>
            ) : null}
          </tbody>
        </Table>
      </Card>
    </div>
  );
}

function AutoPauseAuditSummary({ logs }: { logs: AppReleaseAuditLog[] }) {
  const latest = logs[0];
  const detail = autoPauseDetail(latest);
  return (
    <Card>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex min-w-0 items-start gap-3">
          <div className="mt-0.5 rounded-lg border border-red-200 bg-red-50 p-2 text-red-700">
            <ShieldCheck className="h-4 w-4" />
          </div>
          <div className="min-w-0">
            <div className="font-medium">资源自动暂停保护</div>
            <div className="mt-1 text-sm text-muted-foreground">
              最近一次因激活失败暂停 {detail.resourceVersion || "资源版本"}，后续设备不会继续命中该资源。
            </div>
          </div>
        </div>
        <Badge tone="danger">{logs.length} 次</Badge>
      </div>
      <div className="mt-3 grid gap-2 text-xs md:grid-cols-4">
        <Info label="资源版本" value={detail.resourceVersion} />
        <Info label="设备" value={detail.device} />
        <Info label="包" value={detail.packageKey} />
        <Info label="时间" value={formatDateTime(latest.created_at)} />
      </div>
      {detail.error ? <InlineWarning text={detail.error} /> : null}
    </Card>
  );
}

function Metric({ label, value, note }: { label: string; value: string | number; note: string }) {
  return (
    <Card>
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-2 truncate text-xl font-semibold">{value}</div>
      <div className="mt-1 truncate text-xs text-muted-foreground">{note}</div>
    </Card>
  );
}

function ReleaseSummary({ release, onCopy }: { release: AppRelease; onCopy: (value: string) => void }) {
  const command = `adb install -r ${release.file_name}`;
  const url = downloadUrl(release, true);
  return (
    <div className="grid gap-2 text-xs">
      <Info label="包名" value={release.package_name} />
      <Info label="版本" value={`${release.version_name} / ${release.version_code}`} />
      <Info label="构建" value={`${release.channel} / ${release.build_type} / build ${release.build_number ?? release.version_code}`} />
      <Info label="升级级别" value={updateLevelLabel(release.update_level, release.force_update)} />
      <Info label="灰度比例" value={`${release.rollout_percentage ?? 100}%`} />
      <Info label="最低支持" value={release.min_supported_code ? String(release.min_supported_code) : "-"} />
      <Info label="Commit" value={release.git_commit || "-"} />
      <Info label="大小" value={formatBytes(release.size_bytes)} />
      <Info label="SHA256" value={release.sha256} />
      <div className="mt-2 grid gap-3 rounded-lg border p-3 sm:grid-cols-[auto_minmax(0,1fr)]">
        <QrCodeSvg value={url} size={112} />
        <div className="min-w-0">
          <div className="mb-2 flex items-center gap-2 font-medium">
            <QrCode className="h-4 w-4" />
            快速安装
          </div>
          <div className="mb-2 break-all rounded-md bg-muted p-2 font-mono text-[11px]">{url}</div>
          <div className="break-all rounded-md bg-muted p-2 font-mono text-[11px]">{command}</div>
          <div className="mt-2 grid grid-cols-2 gap-2">
            <Button variant="secondary" size="sm" onClick={() => onCopy(url)}>
              <Clipboard className="mr-2 h-3.5 w-3.5" />
              复制链接
            </Button>
            <Button variant="secondary" size="sm" onClick={() => window.open(url, "_blank")}>
              <Download className="mr-2 h-3.5 w-3.5" />
              下载 APK
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

function ResourceSummary({ resource }: { resource: AppResourceVersion }) {
  return (
    <div className="grid gap-2 text-xs">
      <Info label="版本" value={resource.resource_version} />
      <Info label="渠道" value={resource.channel} />
      <Info label="升级级别" value={updateLevelLabel(resource.update_level)} />
      <Info label="灰度比例" value={`${resource.rollout_percentage ?? 100}%`} />
      <Info label="App 范围" value={resourceAppRange(resource)} />
      <Info label="总大小" value={formatBytes(resource.total_size)} />
      <Info label="Manifest" value={resource.manifest_url} />
      <PackageList packages={resource.packages ?? []} />
    </div>
  );
}

function BuildJobPanel({ job, compact = false }: { job: AppReleaseBuildJob; compact?: boolean }) {
  return (
    <div className="grid gap-3">
      <div className="grid gap-2 text-xs md:grid-cols-2">
        <Info label="任务" value={job.id} />
        <Info label="版本" value={`${job.version_name} / ${job.version_code}`} />
        <Info label="来源" value={job.git_commit ? `${job.git_ref} @ ${job.git_commit}` : job.git_ref} />
        <Info label="耗时" value={job.duration_ms ? `${Math.round(job.duration_ms / 1000)}s` : "-"} />
        <Info label="默认服务" value={job.api_base_url || "-"} />
        <Info label="构建备注" value={job.release_notes || "-"} />
      </div>
      {job.error_message ? <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-xs text-red-700">{job.error_message}</div> : null}
      <pre className={cn("overflow-auto rounded-lg border bg-zinc-950 p-3 text-xs leading-5 text-zinc-100", compact ? "max-h-[140px]" : "max-h-[260px]")}>
        {(job.log_tail?.length ? job.log_tail : ["等待构建日志"]).join("\n")}
      </pre>
    </div>
  );
}

function BuildJobsTable({ jobs, onCreateRelease }: { jobs: AppReleaseBuildJob[]; onCreateRelease: (job: AppReleaseBuildJob) => void }) {
  return (
    <Table>
      <thead>
        <tr>
          <Th>构建</Th>
          <Th>版本</Th>
          <Th>渠道</Th>
          <Th>Git</Th>
          <Th>环境</Th>
          <Th>状态</Th>
          <Th>大小</Th>
          <Th>时间</Th>
          <Th>操作</Th>
        </tr>
      </thead>
      <tbody>
        {jobs.map((job) => (
          <tr key={job.id}>
            <Td>{job.build_number ?? job.id}</Td>
            <Td>{`${job.version_name} / versionCode ${job.version_code}`}</Td>
            <Td>{job.channel}</Td>
            <Td>{job.git_commit || job.git_ref}</Td>
            <Td>{job.build_environment || job.build_type}</Td>
            <Td>
              <Badge tone={buildStatusTone(job.status)}>{buildStatusLabel(job.status)}</Badge>
            </Td>
            <Td>{formatBytes(job.artifact_size)}</Td>
            <Td>{formatDateTime(job.finished_at || job.started_at || job.created_at)}</Td>
            <Td>
              <Button variant="secondary" size="sm" disabled={job.status !== "success"} onClick={() => onCreateRelease(job)}>
                创建发布
              </Button>
            </Td>
          </tr>
        ))}
        {!jobs.length ? (
          <tr>
            <Td colSpan={9}>暂无构建记录</Td>
          </tr>
        ) : null}
      </tbody>
    </Table>
  );
}

function ReleaseRow({
  release,
  busy,
  onAction,
  onAdjustRollout,
  onEditNotes,
}: {
  release: AppRelease;
  busy: boolean;
  onAction: (type: ReleaseAction) => void;
  onAdjustRollout: () => void;
  onEditNotes: () => void;
}) {
  const published = release.is_published || release.status === "released" || release.status === "rolling_out";
  return (
    <div className="min-w-0 rounded-lg border p-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <div className="min-w-0 break-words font-medium">{release.title || release.version_name}</div>
            <Badge>{release.version_code}</Badge>
            <Badge>{release.channel}</Badge>
            {release.is_latest ? <Badge tone="success">latest</Badge> : null}
            <Badge tone={releaseStatusTone(release.status, release.is_published)}>{releaseStatusLabel(release.status, release.is_published)}</Badge>
            <Badge tone={release.update_level === "forced" || release.force_update ? "danger" : "default"}>{updateLevelLabel(release.update_level, release.force_update)}</Badge>
          </div>
          <div className="mt-1 truncate text-xs text-muted-foreground">
            {release.file_name} | {formatDateTime(release.created_at)}
          </div>
        </div>
        <div className="flex max-w-full flex-wrap gap-2">
          <Button variant="secondary" size="sm" onClick={() => window.open(downloadUrl(release, true), "_blank")}>
            <Download className="mr-2 h-3.5 w-3.5" />
            下载
          </Button>
          <Button variant="secondary" size="sm" onClick={() => onAction("publish")} disabled={busy || published}>
            <ShieldCheck className="mr-2 h-3.5 w-3.5" />
            发布
          </Button>
          <Button variant="secondary" size="sm" onClick={() => onAction("pause")} disabled={busy || !published || release.status === "paused"}>
            <PauseCircle className="mr-2 h-3.5 w-3.5" />
            暂停
          </Button>
          <Button variant="secondary" size="sm" onClick={() => onAction("rollback")} disabled={busy || release.is_latest}>
            <History className="mr-2 h-3.5 w-3.5" />
            回滚
          </Button>
          <Button variant="secondary" size="sm" onClick={onAdjustRollout} disabled={busy}>
            灰度
          </Button>
          <Button variant="secondary" size="sm" onClick={onEditNotes} disabled={busy}>
            编辑说明
          </Button>
          <Button variant="secondary" size="sm" onClick={() => onAction("recall")} disabled={busy || !published}>
            撤回
          </Button>
          <Button variant="secondary" size="sm" onClick={() => onAction("unpublish")} disabled={busy || !published}>
            下架
          </Button>
        </div>
      </div>
      <div className="mt-3 grid gap-2 text-xs md:grid-cols-4">
        <Info label="构建" value={`${release.build_type} / build ${release.build_number ?? release.version_code}`} />
        <Info label="灰度" value={`${release.rollout_percentage ?? 100}%`} />
        <Info label="目标" value={releaseTargetLabel(release.target_type, release.target_value)} />
        <Info label="Git" value={`${release.git_ref} @ ${release.git_commit}`} />
      </div>
      <ReleaseNotesPreview
        summary={release.summary}
        markdown={release.release_notes_markdown || release.release_notes}
        emptyText="暂无更新说明"
      />
    </div>
  );
}

function ResourceRow({
  resource,
  busy,
  onAction,
  onAdjustRollout,
  onEditNotes,
}: {
  resource: AppResourceVersion;
  busy: boolean;
  onAction: (type: ResourceAction) => void;
  onAdjustRollout: () => void;
  onEditNotes: () => void;
}) {
  const released = resource.status === "released" || resource.status === "rolling_out";
  const paused = resource.status === "paused";
  return (
    <div className="min-w-0 rounded-lg border p-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <div className="min-w-0 break-words font-medium">{resource.title || resource.resource_version}</div>
            <Badge>{resource.resource_version}</Badge>
            <Badge>{resource.channel}</Badge>
            <Badge tone={statusTone(resource.status)}>{statusLabel(resource.status)}</Badge>
            <Badge tone={resource.update_level === "forced" ? "danger" : "default"}>{updateLevelLabel(resource.update_level)}</Badge>
            {resource.manifest_signed ? <Badge tone="success">Manifest {resource.signature_algorithm || "已签名"}</Badge> : null}
          </div>
          <div className="mt-1 truncate text-xs text-muted-foreground">
            manifest-{resource.resource_version}.json | {formatDateTime(resource.created_at)}
          </div>
          {paused ? (
            <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-red-700">
              <Badge tone="danger">已暂停</Badge>
              <span>{formatDateTime(resource.paused_at)}</span>
            </div>
          ) : null}
        </div>
        <div className="flex max-w-full flex-wrap gap-2">
          <Button variant="secondary" size="sm" onClick={() => window.open(resource.manifest_url, "_blank")}>
            Manifest
          </Button>
          <Button variant="secondary" size="sm" onClick={() => onAction("publish-resource")} disabled={busy || released}>
            发布
          </Button>
          <Button variant="secondary" size="sm" onClick={() => onAction("pause-resource")} disabled={busy || !released || resource.status === "paused"}>
            暂停
          </Button>
          <Button variant="secondary" size="sm" onClick={() => onAction("rollback-resource")} disabled={busy}>
            回滚
          </Button>
          <Button variant="secondary" size="sm" onClick={onAdjustRollout} disabled={busy}>
            灰度
          </Button>
          <Button variant="secondary" size="sm" onClick={onEditNotes} disabled={busy}>
            编辑说明
          </Button>
        </div>
      </div>
      <div className="mt-3 grid gap-2 text-xs md:grid-cols-4">
        <Info label="灰度" value={`${resource.rollout_percentage ?? 100}%`} />
        <Info label="App 范围" value={resourceAppRange(resource)} />
        <Info label="总大小" value={formatBytes(resource.total_size)} />
        <Info label={paused ? "暂停时间" : "包数量"} value={paused ? formatDateTime(resource.paused_at) : String(resource.packages?.length ?? 0)} />
      </div>
      <ReleaseNotesPreview summary={resource.summary} markdown={resource.release_notes_markdown} emptyText="暂无资源说明" />
      <PackageList packages={resource.packages ?? []} />
    </div>
  );
}

function PackageList({ packages }: { packages: AppResourcePackage[] }) {
  if (!packages.length) return null;
  return (
    <div className="mt-3 grid gap-1 rounded-lg border p-3 text-xs">
      {packages.map((pkg) => (
        <div key={pkg.id || pkg.package_key} className="grid gap-2 md:grid-cols-[140px_80px_minmax(0,1fr)]">
          <span className="font-medium">{pkg.package_key}</span>
          <span className="text-muted-foreground">{formatBytes(pkg.file_size)}</span>
          <span className="truncate font-mono text-[11px] text-muted-foreground">{pkg.sha256}</span>
        </div>
      ))}
    </div>
  );
}

function ReleaseNotesPreview({ summary, markdown, emptyText }: { summary?: string; markdown?: string; emptyText: string }) {
  const lines = formatReleaseNotes(markdown);
  const hasSummary = Boolean(summary?.trim());
  if (!hasSummary && !lines.length) return null;
  return (
    <div className="mt-3 rounded-lg border bg-white p-3 text-xs">
      {hasSummary ? <div className="mb-2 font-medium leading-5 text-foreground">{summary}</div> : null}
      <div className="grid gap-1.5 leading-5 text-muted-foreground">
        {lines.length ? lines.map((line, index) => <ReleaseNoteLine key={`${line}-${index}`} line={line} />) : <span>{emptyText}</span>}
      </div>
    </div>
  );
}

function ReleaseNoteLine({ line }: { line: FormattedNoteLine }) {
  if (line.kind === "blank") return <span className="h-1" />;
  if (line.kind === "heading") return <span className="pt-1 font-medium text-foreground">{line.text}</span>;
  return <span className={line.kind === "bullet" ? "pl-3" : ""}>{line.text}</span>;
}

function formatReleaseNotes(markdown?: string): FormattedNoteLine[] {
  return (markdown || "")
    .split(/\r?\n/)
    .map<FormattedNoteLine>((raw) => {
      const line = raw.trim();
      if (!line) return { kind: "blank", text: "" };
      if (/^#{1,6}\s+/.test(line)) return { kind: "heading", text: line.replace(/^#{1,6}\s+/, "") };
      if (/^[-*]\s+/.test(line)) return { kind: "bullet", text: line.replace(/^[-*]\s+/, "• ") };
      return { kind: "text", text: line };
    })
    .filter((line, index, lines) => !(line.kind === "blank" && (index === 0 || index === lines.length - 1)));
}

type FormattedNoteLine = { kind: "heading" | "bullet" | "text" | "blank"; text: string };

function VersionDistribution({ installations }: { installations: AppInstallation[] }) {
  const rows = Object.entries(
    installations.reduce<Record<string, number>>((acc, item) => {
      const key = item.installed_version || "unknown";
      acc[key] = (acc[key] ?? 0) + 1;
      return acc;
    }, {}),
  ).sort((left, right) => right[1] - left[1]);

  if (!rows.length) return <EmptyBox text="暂无设备数据" />;
  return (
    <div className="grid gap-2">
      {rows.map(([version, count]) => (
        <div key={version} className="grid grid-cols-[100px_minmax(0,1fr)_36px] items-center gap-2 text-xs">
          <span className="truncate">{version}</span>
          <div className="h-2 rounded-full bg-muted">
            <div className="h-2 rounded-full bg-black" style={{ width: `${Math.max(8, (count / installations.length) * 100)}%` }} />
          </div>
          <span className="text-right text-muted-foreground">{count}</span>
        </div>
      ))}
    </div>
  );
}

function ConfirmReleaseActionDialog({
  action,
  busy,
  onOpenChange,
  onConfirm,
}: {
  action: PendingReleaseAction;
  busy: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  const copy = action ? releaseActionCopy(action.type, action.release) : undefined;
  return (
    <Dialog open={Boolean(action)} onOpenChange={onOpenChange}>
      <DialogContent className="rounded-lg">
        <DialogTitle className="text-base font-semibold">{copy?.title ?? "确认操作"}</DialogTitle>
        <DialogDescription className="mt-2 text-sm leading-6 text-muted-foreground">{copy?.description ?? "请确认是否继续。"}</DialogDescription>
        {action ? (
          <div className="mt-4 rounded-lg border bg-muted p-3 text-xs">
            <Info label="版本" value={`${action.release.version_name} / ${action.release.version_code}`} />
            <Info label="渠道" value={`${action.release.channel} / ${action.release.build_type}`} />
            <Info label="灰度" value={`${action.release.rollout_percentage ?? 100}%`} />
            <Info label="文件" value={action.release.file_name} />
          </div>
        ) : null}
        <div className="mt-5 flex justify-end gap-2">
          <DialogClose asChild>
            <Button variant="secondary" disabled={busy}>
              取消
            </Button>
          </DialogClose>
          <Button onClick={onConfirm} disabled={busy}>
            {copy?.confirm ?? "确认"}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function ConfirmResourceActionDialog({
  action,
  busy,
  onOpenChange,
  onConfirm,
}: {
  action: PendingResourceAction;
  busy: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  const copy = action ? resourceActionCopy(action.type, action.resource) : undefined;
  return (
    <Dialog open={Boolean(action)} onOpenChange={onOpenChange}>
      <DialogContent className="rounded-lg">
        <DialogTitle className="text-base font-semibold">{copy?.title ?? "确认操作"}</DialogTitle>
        <DialogDescription className="mt-2 text-sm leading-6 text-muted-foreground">{copy?.description ?? "请确认是否继续。"}</DialogDescription>
        {action ? (
          <div className="mt-4 rounded-lg border bg-muted p-3 text-xs">
            <Info label="资源版本" value={action.resource.resource_version} />
            <Info label="渠道" value={action.resource.channel} />
            <Info label="灰度" value={`${action.resource.rollout_percentage ?? 100}%`} />
            <Info label="Manifest" value={action.resource.manifest_url} />
          </div>
        ) : null}
        <div className="mt-5 flex justify-end gap-2">
          <DialogClose asChild>
            <Button variant="secondary" disabled={busy}>
              取消
            </Button>
          </DialogClose>
          <Button onClick={onConfirm} disabled={busy}>
            {copy?.confirm ?? "确认"}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function RolloutDialog({
  target,
  busy,
  onChange,
  onConfirm,
}: {
  target: RolloutTarget;
  busy: boolean;
  onChange: (target: RolloutTarget) => void;
  onConfirm: () => void;
}) {
  const title = target?.kind === "release" ? "调整 App 灰度" : "调整资源灰度";
  const name = target?.kind === "release" ? target.release.version_name : target?.resource.resource_version;
  return (
    <Dialog open={Boolean(target)} onOpenChange={(open) => !open && onChange(null)}>
      <DialogContent className="rounded-lg">
        <DialogTitle className="text-base font-semibold">{title}</DialogTitle>
        <DialogDescription className="mt-2 text-sm leading-6 text-muted-foreground">
          灰度比例会影响 update-check 或 resource-check 命中的设备范围。
        </DialogDescription>
        {target ? (
          <div className="mt-4 grid gap-3">
            <div className="rounded-lg border bg-muted p-3 text-xs">
              <Info label="对象" value={name || "-"} />
              <Info
                label="渠道"
                value={target.kind === "release" ? target.release.channel : target.resource.channel}
              />
            </div>
            <div className="grid grid-cols-[minmax(0,1fr)_90px] gap-2">
              <Select
                label="灰度比例"
                value={target.value}
                onChange={(value) => onChange({ ...target, value } as RolloutTarget)}
                options={rolloutOptions.map(String)}
              />
              <Input
                aria-label="自定义灰度比例"
                type="number"
                min={1}
                max={100}
                value={target.value}
                onChange={(event) => onChange({ ...target, value: event.target.value } as RolloutTarget)}
              />
            </div>
          </div>
        ) : null}
        <div className="mt-5 flex justify-end gap-2">
          <DialogClose asChild>
            <Button variant="secondary" disabled={busy}>
              取消
            </Button>
          </DialogClose>
          <Button onClick={onConfirm} disabled={busy}>
            确认调整
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function NotesDialog({
  target,
  busy,
  onChange,
  onConfirm,
}: {
  target: NotesTarget;
  busy: boolean;
  onChange: (target: NotesTarget) => void;
  onConfirm: () => void;
}) {
  const dialogTitle = target?.kind === "resource" ? "编辑资源说明" : "编辑版本说明";
  const targetName = target?.kind === "resource" ? target.resource.resource_version : target?.release.version_name;
  return (
    <Dialog open={Boolean(target)} onOpenChange={(open) => !open && onChange(null)}>
      <DialogContent className="max-w-4xl rounded-lg">
        <DialogTitle className="text-base font-semibold">{dialogTitle}</DialogTitle>
        {target ? (
          <div className="mt-4 grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
            <div className="grid gap-3">
              <div className="rounded-lg border bg-muted p-3 text-xs">
                <Info label="对象" value={targetName || "-"} />
                <Info label="渠道" value={target.kind === "resource" ? target.resource.channel : target.release.channel} />
              </div>
              <Input
                aria-label="说明标题"
                placeholder="标题"
                value={target.title}
                onChange={(event) => onChange({ ...target, title: event.target.value } as NotesTarget)}
              />
              <Input
                aria-label="说明摘要"
                placeholder="摘要"
                value={target.summary}
                onChange={(event) => onChange({ ...target, summary: event.target.value } as NotesTarget)}
              />
              <textarea
                aria-label="Markdown 更新内容"
                className="min-h-72 w-full rounded-lg border border-input bg-white px-3 py-2 text-xs leading-5 outline-none transition focus:border-black focus:ring-2 focus:ring-black/10"
                value={target.markdown}
                onChange={(event) => onChange({ ...target, markdown: event.target.value } as NotesTarget)}
              />
            </div>
            <div className="rounded-lg border bg-muted/50 p-3">
              <div className="mb-2 text-xs font-medium text-muted-foreground">客户端展示</div>
              <div className="rounded-lg border bg-white p-3">
                <div className="text-sm font-semibold">{target.title.trim() || targetName || "更新说明"}</div>
                <ReleaseNotesPreview
                  summary={target.summary}
                  markdown={target.markdown}
                  emptyText={target.kind === "resource" ? "暂无资源说明" : "暂无更新说明"}
                />
              </div>
            </div>
          </div>
        ) : null}
        <div className="mt-5 flex justify-end gap-2">
          <DialogClose asChild>
            <Button variant="secondary" disabled={busy}>
              取消
            </Button>
          </DialogClose>
          <Button onClick={onConfirm} disabled={busy}>
            保存说明
          </Button>
        </div>
      </DialogContent>
    </Dialog>
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
      className="h-9 min-w-0 w-full rounded-lg border bg-white px-3 text-xs outline-none transition focus:border-black focus:ring-2 focus:ring-black/10"
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

function MarkdownTextarea({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  return (
    <textarea
      className="min-h-36 w-full rounded-lg border border-input bg-white px-3 py-2 text-xs leading-5 outline-none transition focus:border-black focus:ring-2 focus:ring-black/10"
      placeholder="Markdown 更新内容"
      value={value}
      onChange={(event) => onChange(event.target.value)}
    />
  );
}

function ToggleRow({ label, checked, onCheckedChange }: { label: string; checked: boolean; onCheckedChange: (checked: boolean) => void }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-lg border p-3">
      <div className="text-xs font-medium">{label}</div>
      <Switch checked={checked} onCheckedChange={onCheckedChange} aria-label={label} />
    </div>
  );
}

function InlineWarning({ text }: { text: string }) {
  return (
    <div className="flex items-start gap-2 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs leading-5 text-amber-800">
      <XCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
      <span>{text}</span>
    </div>
  );
}

function VersionCodeHint({
  suggestion,
  currentCode,
  currentName,
  onUseRecommended,
}: {
  suggestion: BuildVersionSuggestion;
  currentCode: string;
  currentName: string;
  onUseRecommended: () => void;
}) {
  const current = Number(currentCode);
  const alignedWithRecommendation =
    Number.isInteger(current) && current === suggestion.recommendedCode && currentName.trim() === suggestion.recommendedName;
  const detail = suggestion.versionNameError || suggestion.channelMismatchError
    ? suggestion.versionNameError
      ? suggestion.versionNameError
      : suggestion.channelMismatchError
    : suggestion.semanticCode && suggestion.semanticCode < suggestion.minNextCode
      ? `语义规则对应 ${suggestion.semanticCode}，已有构建/发布最高到 ${suggestion.minNextCode - 1}，本次建议 ${suggestion.recommendedCode}。`
      : `语义规则建议 versionCode ${suggestion.recommendedCode}。`;

  return (
    <div className="flex flex-wrap items-center justify-between gap-2 rounded-lg border bg-muted px-3 py-2 text-xs text-muted-foreground">
      <div className="min-w-0">
        <div className="font-medium text-foreground">
          版本建议：{suggestion.recommendedName} / {suggestion.recommendedCode}
        </div>
        <div className="mt-1 leading-5">{detail}</div>
      </div>
      <Button variant="secondary" size="sm" onClick={onUseRecommended} disabled={alignedWithRecommendation}>
        使用建议值
      </Button>
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

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-3">
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span className="break-all text-right font-medium">{value || "-"}</span>
    </div>
  );
}

function EmptyBox({ text }: { text: string }) {
  return <div className="rounded-lg border bg-muted p-4 text-sm text-muted-foreground">{text}</div>;
}

function buildStatusTone(status: AppBuildStatus): "default" | "success" | "warning" | "danger" {
  if (status === "success") return "success";
  if (status === "failed" || status === "canceled") return "danger";
  if (status === "queued" || status === "running" || status === "building") return "warning";
  return "default";
}

function buildStatusLabel(status: AppBuildStatus) {
  return (
    {
      queued: "排队中",
      running: "构建中",
      building: "构建中",
      success: "成功",
      failed: "失败",
      canceled: "已取消",
      archived: "已归档",
    }[status] ?? status
  );
}

function statusTone(status?: AppReleaseStatus): "default" | "success" | "warning" | "danger" {
  if (status === "released") return "success";
  if (status === "rolling_out" || status === "testing" || status === "scheduled") return "warning";
  if (status === "recalled" || status === "archived") return "danger";
  return "default";
}

function statusLabel(status?: AppReleaseStatus) {
  if (!status) return "草稿";
  return (
    {
      draft: "草稿",
      testing: "测试中",
      scheduled: "已定时",
      rolling_out: "灰度中",
      released: "已发布",
      paused: "已暂停",
      recalled: "已撤回",
      archived: "已归档",
    }[status] ?? status
  );
}

function releaseStatusTone(status?: AppReleaseStatus, published?: boolean): "default" | "success" | "warning" | "danger" {
  if (published && !status) return "success";
  return statusTone(status);
}

function releaseStatusLabel(status?: AppReleaseStatus, published?: boolean) {
  if (status) return statusLabel(status);
  return published ? "已发布" : "草稿";
}

function eventTone(eventType: string): "default" | "success" | "warning" | "danger" {
  if (eventType.includes("success") || eventType.includes("completed")) return "success";
  if (eventType.includes("failed") || eventType.includes("checksum")) return "danger";
  if (eventType.includes("started") || eventType.includes("detected")) return "warning";
  return "default";
}

function eventLabel(eventType: string) {
  return (
    {
      update_detected: "发现更新",
      download_started: "开始下载",
      download_completed: "下载完成",
      checksum_failed: "校验失败",
      install_started: "开始安装",
      install_failed: "安装失败",
      install_success: "安装成功",
      app_started: "App 启动",
      rollback_detected: "检测回滚",
      manifest_checked: "检查清单",
      extract_failed: "解压失败",
      activation_success: "激活成功",
      activation_failed: "激活失败",
      rollback_success: "回滚成功",
    }[eventType] ?? eventType
  );
}

function auditActionLabel(action: string) {
  return (
    {
      "release.create": "创建发布",
      "release.publish": "发布版本",
      "release.pause": "暂停发布",
      "release.recall": "撤回版本",
      "release.rollback": "回滚版本",
      "release.rollout": "调整灰度",
      "resource.create": "创建资源",
      "resource.publish": "发布资源",
      "resource.pause": "暂停资源",
      "resource.rollback": "回滚资源",
      "resource.rollout": "调整资源灰度",
      "resource.notes": "更新资源说明",
      "resource.auto_pause": "自动暂停资源",
    }[action] ?? action
  );
}

function releaseTargetPlaceholder(targetType: string) {
  if (targetType === "user_group") return "例如 internal-testers";
  if (targetType === "user_id") return "用户 ID，多个用逗号分隔";
  if (targetType === "device_id") return "设备 ID，多个用逗号分隔";
  return "全部用户";
}

function releaseTargetLabel(targetType?: string, targetValue?: string) {
  if (!targetType || targetType === "all") return "全部";
  const label =
    releaseTargetTypes.find((item) => item.value === targetType)?.label ??
    targetType;
  return targetValue ? `${label}: ${targetValue}` : label;
}

function resourceAppRange(resource: AppResourceVersion) {
  const min = resource.min_app_version_code ? String(resource.min_app_version_code) : "-";
  const max = resource.max_app_version_code ? String(resource.max_app_version_code) : "-";
  if (min === "-" && max === "-") return "不限";
  return `${min} - ${max}`;
}

function installationTone(item: AppInstallation): "default" | "success" | "warning" | "danger" {
  const status = installationStatus(item);
  if (status === "正常") return "success";
  if (status.includes("失败")) return "danger";
  if (status === "待更新") return "warning";
  return "default";
}

function installationStatus(item: AppInstallation) {
  if (item.last_upgrade_status?.includes("failed") || item.last_error) return "资源激活失败";
  if (item.last_upgrade_status?.includes("pending")) return "待更新";
  if (!item.last_seen_at) return "未知";
  return "正常";
}

function releaseActionCopy(type: ReleaseAction, release: AppRelease) {
  if (type === "publish") {
    return {
      title: "发布版本",
      description: `发布后 ${release.channel} 渠道会把 ${release.version_name} 标记为可检查、可下载版本。`,
      confirm: "确认发布",
    };
  }
  if (type === "pause") {
    return {
      title: "暂停发布",
      description: `暂停后 ${release.channel} 渠道不会继续扩大 ${release.version_name} 的灰度范围。`,
      confirm: "确认暂停",
    };
  }
  if (type === "recall") {
    return {
      title: "撤回版本",
      description: `撤回会让 ${release.version_name} 从可更新版本中移除，适合发现严重问题时使用。`,
      confirm: "确认撤回",
    };
  }
  if (type === "rollback") {
    return {
      title: "回滚版本",
      description: `回滚后 ${release.channel} 渠道会把 ${release.version_name} 标记为 latest。APK 降级安装仍需要重新构建更高 versionCode。`,
      confirm: "确认回滚",
    };
  }
  if (type === "rollout") {
    return {
      title: "同步灰度比例",
      description: `将 ${release.version_name} 的灰度比例同步为 ${release.rollout_percentage ?? 100}%。`,
      confirm: "确认同步",
    };
  }
  return {
    title: "下架版本",
    description: `下架后 ${release.version_name} 将不再作为已发布版本提供下载。`,
    confirm: "确认下架",
  };
}

function resourceActionCopy(type: ResourceAction, resource: AppResourceVersion) {
  if (type === "publish-resource") {
    return {
      title: "发布资源版本",
      description: `发布后 ${resource.channel} 渠道会检查到资源版本 ${resource.resource_version}。`,
      confirm: "确认发布",
    };
  }
  if (type === "pause-resource") {
    return {
      title: "暂停资源版本",
      description: `暂停后资源版本 ${resource.resource_version} 不再继续扩大灰度。`,
      confirm: "确认暂停",
    };
  }
  if (type === "resource-rollout") {
    return {
      title: "同步资源灰度",
      description: `将资源版本 ${resource.resource_version} 的灰度比例同步为 ${resource.rollout_percentage ?? 100}%。`,
      confirm: "确认同步",
    };
  }
  return {
    title: "回滚资源版本",
    description: `资源回滚会把当前渠道指针切回 ${resource.resource_version}，客户端下次启动后激活对应资源。`,
    confirm: "确认回滚",
  };
}

function downloadUrl(release: AppRelease, absolute = false) {
  const url = release.download_url || `/admin/api/app-releases/${encodeURIComponent(release.id)}/download`;
  if (!absolute || /^https?:\/\//i.test(url)) return url;
  return `${window.location.origin}${url.startsWith("/") ? url : `/${url}`}`;
}

function formatBytes(value?: number) {
  if (!value) return "-";
  if (value < 1024 * 1024) return `${Math.round(value / 1024)}KB`;
  return `${(value / 1024 / 1024).toFixed(1)}MB`;
}

async function copyText(value: string) {
  if (!navigator.clipboard) throw new Error("clipboard unavailable");
  await navigator.clipboard.writeText(value);
}

function validateNotesPayload(payload: UpdateReleaseNotesPayload) {
  if (!payload.title.trim()) return "请填写说明标题。";
  if (!payload.release_notes_markdown.trim()) return "请填写 Markdown 更新内容。";
  return "";
}

function validateAppForm({
  appKey,
  appName,
  appPlatform,
  appPackageName,
}: {
  appKey: string;
  appName: string;
  appPlatform: string;
  appPackageName: string;
}) {
  if (!appKey.trim()) return "请填写 app_key。";
  if (!/^[a-z0-9][a-z0-9._-]{1,62}$/i.test(appKey.trim())) return "app_key 只允许字母、数字、点、下划线和中划线。";
  if (!appName.trim()) return "请填写应用名称。";
  if (!appPlatform.trim()) return "请填写平台。";
  if (!appPackageName.trim()) return "请填写包名。";
  if (!/^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$/i.test(appPackageName.trim())) return "包名格式不正确。";
  return "";
}

function validateBuildForm({
  gitRef,
  versionName,
  versionCode,
  buildNumber,
  apiBaseUrl,
  releases,
  buildJobs,
  buildVersionSuggestion,
}: {
  gitRef: string;
  versionName: string;
  versionCode: string;
  buildNumber: string;
  apiBaseUrl: string;
  releases: AppRelease[];
  buildJobs: AppReleaseBuildJob[];
  buildVersionSuggestion: BuildVersionSuggestion;
}) {
  if (!gitRef.trim()) return "请填写 branch、tag 或 commit。";
  if (!isAllowedGitRef(gitRef.trim())) return "gitRef 只允许 main、develop、release/* 或 7-40 位 commit hash。";
  if (!versionName.trim()) return "请填写 versionName。";
  if (buildVersionSuggestion.versionNameError) return buildVersionSuggestion.versionNameError;
  if (buildVersionSuggestion.channelMismatchError) return buildVersionSuggestion.channelMismatchError;
  const code = Number(versionCode);
  if (!Number.isInteger(code) || code <= 0) return "versionCode 必须是大于 0 的整数。";
  if (code < buildVersionSuggestion.minNextCode) {
    return `versionCode 必须持续递增，当前至少应为 ${buildVersionSuggestion.minNextCode}。`;
  }
  if (buildVersionSuggestion.semanticCode && code < buildVersionSuggestion.semanticCode) {
    return `versionName ${versionName.trim()} 按规范对应 versionCode ${buildVersionSuggestion.semanticCode}，请使用不小于该值的版本号。`;
  }
  if (buildJobs.some((job) => job.version_code === code && (job.status === "queued" || job.status === "running"))) {
    return `versionCode ${code} 已有构建任务在进行中。`;
  }
  const buildNumberValue = Number(buildNumber);
  if (!Number.isInteger(buildNumberValue) || buildNumberValue <= 0) return "buildNumber 必须是大于 0 的整数。";
  const minBuildNumber = nextBuildNumberValue(buildJobs);
  if (buildNumberValue < minBuildNumber) {
    return `buildNumber 必须持续递增，当前至少应为 ${minBuildNumber}。`;
  }
  if (buildJobs.some((job) => job.build_number === buildNumberValue)) return `buildNumber ${buildNumberValue} 已存在。`;
  if (!apiBaseUrl.trim()) return "请填写 apiBaseUrl，打包时会写入 APK。";
  try {
    const url = new URL(normalizeUrl(apiBaseUrl));
    if (!["http:", "https:"].includes(url.protocol)) return "apiBaseUrl 只支持 http 或 https。";
  } catch {
    return "apiBaseUrl 格式不正确。";
  }
  return "";
}

function validateReleaseForm({
  build,
  title,
  rollout,
  minCode,
  targetType,
  targetValue,
  scheduleMode,
  scheduledAt,
}: {
  build?: AppReleaseBuildJob;
  title: string;
  rollout: string;
  minCode: string;
  targetType: string;
  targetValue: string;
  scheduleMode: string;
  scheduledAt: string;
}) {
  if (!build) return "请选择一个成功构建。";
  if (!title.trim()) return "请填写发布标题。";
  const rolloutValue = Number(rollout);
  if (!Number.isInteger(rolloutValue) || rolloutValue < 1 || rolloutValue > 100) return "灰度比例必须在 1 到 100 之间。";
  if (targetType !== "all" && !targetValue.trim()) return "请选择目标范围后填写目标值。";
  if (minCode) {
    const code = Number(minCode);
    if (!Number.isInteger(code) || code <= 0) return "最低支持 versionCode 必须是正整数。";
  }
  if (scheduleMode === "scheduled") {
    if (!scheduledAt) return "请选择定时发布时间。";
    const scheduledTime = new Date(scheduledAt).getTime();
    if (Number.isNaN(scheduledTime)) return "定时发布时间格式不正确。";
    if (scheduledTime <= Date.now()) return "定时发布时间必须晚于当前时间。";
  }
  return "";
}

function validateResourceForm({
  resourceVersion,
  title,
  rollout,
  minAppCode,
  maxAppCode,
  files,
  existingResources,
}: {
  resourceVersion: string;
  title: string;
  rollout: string;
  minAppCode: string;
  maxAppCode: string;
  files: File[];
  existingResources: AppResourceVersion[];
}) {
  if (!/^\d{8}\.\d+$/.test(resourceVersion.trim())) return "资源版本请使用 YYYYMMDD.N 格式。";
  if (existingResources.some((resource) => resource.resource_version === resourceVersion.trim())) return "该资源版本已存在。";
  if (!title.trim()) return "请填写资源更新标题。";
  const rolloutValue = Number(rollout);
  if (!Number.isInteger(rolloutValue) || rolloutValue < 1 || rolloutValue > 100) return "灰度比例必须在 1 到 100 之间。";
  if (minAppCode) {
    const code = Number(minAppCode);
    if (!Number.isInteger(code) || code <= 0) return "最低 App versionCode 必须是正整数。";
  }
  if (maxAppCode) {
    const code = Number(maxAppCode);
    if (!Number.isInteger(code) || code <= 0) return "最高 App versionCode 必须是正整数。";
  }
  if (minAppCode && maxAppCode && Number(minAppCode) > Number(maxAppCode)) {
    return "最低 App versionCode 不能大于最高 App versionCode。";
  }
  if (!files.length) return "请至少上传一个资源 ZIP。";
  if (files.some((file) => !file.name.toLowerCase().endsWith(".zip"))) return "资源包必须是 ZIP 文件。";
  return "";
}

function isAllowedGitRef(value: string) {
  return value === "main" || value === "develop" || value.startsWith("release/") || /^[0-9a-f]{7,40}$/i.test(value);
}

function normalizeUrl(value: string) {
  const trimmed = value.trim();
  return trimmed.endsWith("/") ? trimmed : `${trimmed}/`;
}

function defaultApiBaseUrl() {
  const configured = import.meta.env.VITE_API_BASE_URL;
  const value = typeof configured === "string" && configured.trim() ? configured.trim() : window.location.origin;
  try {
    return normalizeUrl(new URL(value, window.location.origin).toString());
  } catch {
    return normalizeUrl(window.location.origin);
  }
}

function suggestBuildVersion(versionName: string, channel: string, releases: AppRelease[], jobs: AppReleaseBuildJob[]): BuildVersionSuggestion {
  const minNextCode = nextVersionCode(releases, jobs);
  const recommendedName = versionNameForCode(minNextCode, channel);
  const semantic = semanticVersionCode(versionName);
  if (semantic.error) {
    return {
      minNextCode,
      recommendedCode: minNextCode,
      recommendedName,
      versionNameError: semantic.error,
      channelMismatchError: "",
    };
  }
  const semanticCode = semantic.code;
  const channelMismatchError = validateVersionNameChannel(versionName, channel);
  return {
    semanticCode,
    minNextCode,
    recommendedCode: Math.max(semanticCode ?? 1, minNextCode),
    recommendedName,
    versionNameError: "",
    channelMismatchError,
  };
}

function semanticVersionCode(versionName: string): { code?: number; error: string } {
  const trimmed = versionName.trim();
  if (!trimmed) return { error: "" };
  const match = trimmed.match(/^(\d+)\.(\d+)\.(\d+)(?:[-+][0-9A-Za-z.-]+)?$/);
  if (!match) return { error: "versionName 请使用 major.minor.patch，可带 -dev.N / -internal.N / -beta.N 预发布标签。" };

  const [, majorText, minorText, patchText] = match;
  const major = Number(majorText);
  const minor = Number(minorText);
  const patch = Number(patchText);
  if (minor > 99 || patch > 99) return { error: "versionCode 规则要求 minor 和 patch 小于 100。" };

  return { code: major * 10000 + minor * 100 + patch, error: "" };
}

function versionNameForCode(versionCode: number, channel: string) {
  const major = Math.floor(versionCode / 10000);
  const minor = Math.floor((versionCode % 10000) / 100);
  const patch = versionCode % 100;
  const base = `${major}.${minor}.${patch}`;
  return prereleaseChannels.includes(channel) ? `${base}-${channel}.1` : base;
}

function validateVersionNameChannel(versionName: string, channel: string) {
  const trimmed = versionName.trim();
  if (!trimmed || !prereleaseChannels.includes(channel)) return "";
  const suffix = `-${channel}.`;
  return trimmed.includes(suffix) ? "" : `当前渠道是 ${channel}，versionName 建议使用 ${suffix.replace(".", ".N")} 预发布标签。`;
}

function mutationIsPending(mutation: { isPending: boolean }) {
  return mutation.isPending;
}

function sortReleases(releases: AppRelease[]) {
  return [...releases].sort((left, right) => {
    if (right.version_code !== left.version_code) return right.version_code - left.version_code;
    return new Date(right.created_at || 0).getTime() - new Date(left.created_at || 0).getTime();
  });
}

function sortBuildJobs(jobs: AppReleaseBuildJob[]) {
  return [...jobs].sort((left, right) => {
    const statusOrder = buildJobStatusWeight(left.status) - buildJobStatusWeight(right.status);
    if (statusOrder !== 0) return statusOrder;
    return new Date(right.started_at || right.created_at || 0).getTime() - new Date(left.started_at || left.created_at || 0).getTime();
  });
}

function sortResources(resources: AppResourceVersion[]) {
  return [...resources].sort((left, right) => {
    const statusOrder = resourceStatusWeight(left.status) - resourceStatusWeight(right.status);
    if (statusOrder !== 0) return statusOrder;
    return right.resource_version.localeCompare(left.resource_version);
  });
}

function sortInstallations(installations: AppInstallation[]) {
  return [...installations].sort((left, right) => new Date(right.last_seen_at || 0).getTime() - new Date(left.last_seen_at || 0).getTime());
}

function sortEvents(events: AppUpgradeEvent[]) {
  return [...events].sort((left, right) => new Date(right.created_at || 0).getTime() - new Date(left.created_at || 0).getTime());
}

function sortAuditLogs(logs: AppReleaseAuditLog[]) {
  return [...logs].sort((left, right) => new Date(right.created_at || 0).getTime() - new Date(left.created_at || 0).getTime());
}

function nextVersionCode(releases: AppRelease[], jobs: AppReleaseBuildJob[]) {
  const maxReleaseCode = releases.reduce((max, release) => Math.max(max, release.version_code || 0), 0);
  const maxJobCode = jobs.reduce((max, job) => Math.max(max, job.version_code || 0), 0);
  return Math.max(maxReleaseCode, maxJobCode) + 1;
}

function nextBuildNumberValue(jobs: AppReleaseBuildJob[]) {
  const maxBuildNumber = jobs.reduce((max, job) => Math.max(max, job.build_number || 0), 0);
  return maxBuildNumber + 1;
}

function buildJobStatusWeight(status: string) {
  if (status === "running" || status === "building") return 0;
  if (status === "queued") return 1;
  if (status === "failed") return 2;
  if (status === "canceled") return 3;
  return 4;
}

function resourceStatusWeight(status?: string) {
  if (status === "rolling_out") return 0;
  if (status === "released") return 1;
  if (status === "testing" || status === "scheduled") return 2;
  if (status === "paused") return 3;
  if (status === "draft") return 4;
  return 5;
}

function calculateStats(
  releases: AppRelease[],
  resources: AppResourceVersion[],
  installations: AppInstallation[],
  events: AppUpgradeEvent[],
) {
  const latestCode = releases.reduce((max, release) => Math.max(max, release.version_code || 0), 0);
  const latestResource = resources[0]?.resource_version;
  const pendingDevices = installations.filter((item) => {
    const appOld = latestCode > 0 && (item.installed_code || 0) < latestCode;
    const resourceOld = latestResource && item.resource_version && item.resource_version !== latestResource;
    return appOld || resourceOld;
  }).length;
  const failedEvents = events.filter((event) => event.event_type.includes("failed") || event.event_type.includes("checksum")).length;
  const successEvents = events.filter((event) => event.event_type.includes("success") || event.event_type.includes("completed")).length;
  const totalTerminal = successEvents + failedEvents;
  return {
    activeDevices: installations.filter((item) => item.last_seen_at).length,
    pendingDevices,
    failedEvents,
    successRate: totalTerminal ? Math.round((successEvents / totalTerminal) * 100) : 100,
  };
}

function countBy<T>(items: T[], getLabel: (item: T) => string) {
  const counts = items.reduce<Record<string, number>>((acc, item) => {
    const label = getLabel(item) || "-";
    acc[label] = (acc[label] ?? 0) + 1;
    return acc;
  }, {});
  return Object.entries(counts)
    .map(([label, count]) => ({ label, count }))
    .sort((left, right) => right.count - left.count || left.label.localeCompare(right.label));
}

function percentage(value: number, total: number) {
  if (!total) return 0;
  return Math.max(6, Math.round((value / total) * 100));
}

function updateLevelLabel(level?: AppUpdateLevel, forceUpdate?: boolean) {
  if (forceUpdate || level === "forced") return "强制更新";
  if (level === "recommended") return "推荐更新";
  if (level === "normal" || !level) return "普通更新";
  return level;
}

function eventVersionText(event: AppUpgradeEvent) {
  if (event.from_version || event.to_version) return `${event.from_version || "-"} -> ${event.to_version || "-"}`;
  if (event.from_version_code || event.to_version_code) return `${event.from_version_code || "-"} -> ${event.to_version_code || "-"}`;
  return "-";
}

function qualityMetricLabel(category: string) {
  if (category === "apk") return "APK 升级质量";
  if (category === "resource") return "资源激活质量";
  return `${category} 质量`;
}

function qualityActionLabel(action?: string) {
  if (action === "pause_resource") return "建议暂停资源";
  if (action === "rollback_apk") return "建议回滚 APK";
  return "继续观察";
}

function deriveQualityMetrics(apkEvents: AppUpgradeEvent[], resourceEvents: AppUpgradeEvent[]): ReleaseQualityMetric[] {
  return [deriveQualityMetric("apk", apkEvents), deriveQualityMetric("resource", resourceEvents)];
}

function deriveQualityAlerts(metrics: ReleaseQualityMetric[]): QualityAlert[] {
  return metrics
    .filter((metric) => metric.recommended_action && metric.recommended_action !== "observe")
    .map((metric) => ({
      id: `quality.${metric.category}.${metric.recommended_action}`,
      category: metric.category,
      severity: metric.recommended_action === "rollback_apk" ? "critical" : "warning",
      recommended_action: metric.recommended_action || "observe",
      reason: metric.action_reason || "质量指标达到策略阈值",
      threshold: metric.policy_threshold || "",
      failure_rate: metric.failure_rate,
      failure_events: metric.failure_events,
      latest_failure: metric.latest_failure_reason,
    }));
}

function deriveQualityMetric(category: string, events: AppUpgradeEvent[]): ReleaseQualityMetric {
  const failureCounts = new Map<string, number>();
  let successEvents = 0;
  let failureEvents = 0;
  let latestFailureReason = "";
  for (const event of events) {
    if (eventTone(event.event_type) === "success") {
      successEvents += 1;
      continue;
    }
    if (eventTone(event.event_type) !== "danger") continue;
    failureEvents += 1;
    const reason = event.error_message || eventLabel(event.event_type);
    if (!latestFailureReason) latestFailureReason = reason;
    failureCounts.set(reason, (failureCounts.get(reason) ?? 0) + 1);
  }
  const terminalEvents = successEvents + failureEvents;
  return {
    category,
    total_events: events.length,
    success_events: successEvents,
    failure_events: failureEvents,
    success_rate: terminalEvents ? Math.round((successEvents / terminalEvents) * 100) : 100,
    failure_rate: terminalEvents ? Math.round((failureEvents / terminalEvents) * 100) : 0,
    latest_failure_reason: latestFailureReason,
    recommended_action: "observe",
    action_reason: "指标未达到自动干预阈值",
    policy_threshold:
      category === "resource"
        ? "activation_failed >= 3 且失败率 >= 5%"
        : "checksum_failed >= 3，或 install_failed >= 10 且失败率 >= 10%",
    failure_reasons: [...failureCounts.entries()]
      .map(([reason, count]) => ({ reason, count }))
      .sort((left, right) => right.count - left.count || left.reason.localeCompare(right.reason))
      .slice(0, 5),
  };
}

function auditSummary(log: AppReleaseAuditLog) {
  const after = asRecord(log.after_json);
  const before = asRecord(log.before_json);
  if (log.action === "resource.auto_pause") {
    const detail = autoPauseDetail(log);
    return [
      detail.resourceVersion,
      detail.device ? `设备 ${detail.device}` : "",
      detail.packageKey ? `包 ${detail.packageKey}` : "",
      detail.error,
    ]
      .filter(Boolean)
      .join(" / ") || "资源激活失败";
  }
  const version = after?.version_name ?? after?.resource_version ?? before?.version_name ?? before?.resource_version;
  const status = after?.status ?? before?.status;
  return [version, status].filter(Boolean).join(" / ") || "-";
}

function autoPauseDetail(log: AppReleaseAuditLog) {
  const after = asRecord(log.after_json);
  return {
    resourceVersion: stringField(after, "resource_version"),
    device: stringField(after, "device_id") || stringField(after, "auto_paused_device"),
    packageKey: stringField(after, "package_key") || stringField(after, "auto_paused_package"),
    error: stringField(after, "error_message") || stringField(after, "auto_paused_error"),
  };
}

function stringField(record: Record<string, unknown> | undefined, key: string) {
  const value = record?.[key];
  if (typeof value === "string") return value;
  if (typeof value === "number") return String(value);
  return "";
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === "object" && !Array.isArray(value) ? (value as Record<string, unknown>) : undefined;
}

async function refetchWithMessage(
  refetch: () => Promise<unknown>,
  message: string,
  setMessage: (value: string) => void,
  setMessageTone: (value: "default" | "success" | "warning" | "danger") => void,
) {
  setMessage(message);
  setMessageTone("success");
  await refetch();
}

function showError(
  error: unknown,
  fallback: string,
  setMessage: (value: string) => void,
  setMessageTone: (value: "default" | "success" | "warning" | "danger") => void,
) {
  setMessage(error instanceof Error ? error.message : fallback);
  setMessageTone("danger");
}

function defaultResourceVersion() {
  const date = new Date();
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}${month}${day}.1`;
}

const defaultReleaseMarkdown = `## 本次更新

- 优化任务执行稳定性。
- 修复部分设备识别异常。

## 已知问题

- 低版本 Android 首次安装可能需要手工允许未知来源安装。`;

const defaultResourceMarkdown = `## 本次更新

- 更新识图模板。
- 更新 OCR 关键词库。

## 注意事项

- 强制资源更新会在启动页阻塞下载。`;
