import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Cloud, FileText, History, PauseCircle, RefreshCw, Rocket, Search, ShieldCheck, SlidersHorizontal, Upload } from "lucide-react";
import { getAdminIdentity } from "@/api/client";
import { getArtifactCenterOverview, type ArtifactCenterItem } from "@/api/artifacts";
import { getBuildCenterOverview } from "@/api/buildCenter";
import { getDeploymentTargets, type DeploymentTarget } from "@/api/deployments";
import {
  approveReleasePlan,
  createReleasePlan,
  createReleasePlanDeployment,
  createReleaseUnit,
  getReleasePlanOverview,
  pauseReleasePlan,
  publishReleasePlan,
  rollbackReleasePlan,
  type CreateReleasePlanPayload,
  type ReleaseEnvironment,
  type ReleasePlan,
  type ReleasePlanStatus,
  type ReleaseUnit,
  type ReleaseUnitType,
} from "@/api/releasePlans";
import { ApiErrorState } from "@/components/stable/StableAdminComponents";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Switch } from "@/components/ui/Switch";
import { Table, Td, Th } from "@/components/ui/Table";
import { formatDateTime } from "@/lib/format";
import { hasPermission, missingPermissionText } from "@/lib/permissions";

const RELEASE_PLANS_QUERY_KEY = ["release-plans"] as const;
const BUILD_CENTER_PROJECTS_QUERY_KEY = ["release-plans-build-center-projects"] as const;
const ARTIFACT_CENTER_QUERY_KEY = ["release-plans-artifacts"] as const;
const DEPLOYMENT_TARGETS_QUERY_KEY = ["release-plans-deployment-targets"] as const;

const unitTypeOptions: Array<{ value: ReleaseUnitType; label: string }> = [
  { value: "android", label: "Android" },
  { value: "web", label: "Web" },
  { value: "docs", label: "Docs" },
  { value: "worker", label: "Worker" },
  { value: "server", label: "Server" },
  { value: "docker", label: "Docker" },
  { value: "config", label: "Config" },
];

const targetTypeOptions = [
  { value: "all", label: "全部" },
  { value: "user_group", label: "用户组" },
  { value: "user_id", label: "指定用户" },
  { value: "device_id", label: "指定设备" },
  { value: "region", label: "区域" },
];

const rolloutOptions = ["5", "10", "20", "50", "100"];
const channelOptions = ["dev", "internal", "beta", "stable", "emergency"];

type PlanAction = "publish" | "approve" | "pause" | "rollback" | "rollback_deploy";

export type ReleasePlanDraftSeed = {
  seedKey: string;
  projectKey?: string;
  unitKey?: string;
  environmentKey?: string;
  channel?: string;
  title?: string;
  versionName?: string;
  buildNumber?: string | number;
  gitCommit?: string;
  artifactName?: string;
  artifactType?: string;
  artifactFileName?: string;
  artifactRef?: string;
  artifactAppBuildId?: string;
  artifactAppBuildArtifactId?: string;
  unitTypeFilter?: string;
  message?: string;
};

export function ReleasePlansPanel({ draftSeed }: { draftSeed?: ReleasePlanDraftSeed } = {}) {
  const [projectKey, setProjectKey] = useState(draftSeed?.projectKey ?? "release-center");
  const [unitKey, setUnitKey] = useState(draftSeed?.unitKey ?? "admin-web");
  const [environmentKey, setEnvironmentKey] = useState(draftSeed?.environmentKey ?? "dev");
  const [planTitle, setPlanTitle] = useState(draftSeed?.title ?? "管理后台 Web 发布");
  const [versionName, setVersionName] = useState(draftSeed?.versionName ?? defaultVersionName());
  const [buildNumber, setBuildNumber] = useState(draftSeed?.buildNumber !== undefined ? String(draftSeed.buildNumber) : "1");
  const [gitCommit, setGitCommit] = useState(draftSeed?.gitCommit ?? "");
  const [channel, setChannel] = useState(draftSeed?.channel ?? "dev");
  const [rollout, setRollout] = useState("100");
  const [targetType, setTargetType] = useState("all");
  const [targetValue, setTargetValue] = useState("");
  const [artifactName, setArtifactName] = useState(draftSeed?.artifactName ?? "web-dist");
  const [artifactType, setArtifactType] = useState(draftSeed?.artifactType ?? "web_dist");
  const [artifactFileName, setArtifactFileName] = useState(draftSeed?.artifactFileName ?? "");
  const [artifactRef, setArtifactRef] = useState(draftSeed?.artifactRef ?? "");
  const [selectedArtifactId, setSelectedArtifactId] = useState("");
  const [artifactBuildRunId, setArtifactBuildRunId] = useState("");
  const [artifactAppBuildId, setArtifactAppBuildId] = useState(draftSeed?.artifactAppBuildId ?? "");
  const [artifactAppBuildArtifactId, setArtifactAppBuildArtifactId] = useState(draftSeed?.artifactAppBuildArtifactId ?? "");
  const [newUnitProjectKey, setNewUnitProjectKey] = useState("release-center");
  const [newUnitKey, setNewUnitKey] = useState("");
  const [newUnitName, setNewUnitName] = useState("");
  const [newUnitType, setNewUnitType] = useState<ReleaseUnitType>("web");
  const [newUnitChannel, setNewUnitChannel] = useState("dev");
  const [environmentFilter, setEnvironmentFilter] = useState("全部");
  const [unitTypeFilter, setUnitTypeFilter] = useState(draftSeed?.unitTypeFilter ?? "全部");
  const [selectedDeploymentTargetId, setSelectedDeploymentTargetId] = useState("");
  const [deploymentDryRun, setDeploymentDryRun] = useState(true);
  const [query, setQuery] = useState("");
  const [message, setMessage] = useState("发布计划已就绪。");
  const [messageTone, setMessageTone] = useState<"default" | "success" | "warning" | "danger">("default");
  const role = getAdminIdentity().role;
  const permissions = {
    releaseWrite: hasPermission(role, "release:write"),
    releaseApprove: hasPermission(role, "release:approve"),
    releaseRollback: hasPermission(role, "release:rollback"),
    deployWrite: hasPermission(role, "deploy:write"),
  };

  useEffect(() => {
    if (!draftSeed?.seedKey) return;
    if (draftSeed.projectKey) setProjectKey(draftSeed.projectKey);
    if (draftSeed.unitKey) setUnitKey(draftSeed.unitKey);
    if (draftSeed.environmentKey) setEnvironmentKey(draftSeed.environmentKey);
    if (draftSeed.channel) setChannel(draftSeed.channel);
    if (draftSeed.title) setPlanTitle(draftSeed.title);
    if (draftSeed.versionName) setVersionName(draftSeed.versionName);
    if (draftSeed.buildNumber !== undefined) setBuildNumber(String(draftSeed.buildNumber));
    if (draftSeed.gitCommit !== undefined) setGitCommit(draftSeed.gitCommit);
    if (draftSeed.artifactName) setArtifactName(draftSeed.artifactName);
    if (draftSeed.artifactType) setArtifactType(draftSeed.artifactType);
    if (draftSeed.artifactFileName !== undefined) setArtifactFileName(draftSeed.artifactFileName);
    if (draftSeed.artifactRef !== undefined) setArtifactRef(draftSeed.artifactRef);
    if (draftSeed.artifactAppBuildId !== undefined) setArtifactAppBuildId(draftSeed.artifactAppBuildId);
    if (draftSeed.artifactAppBuildArtifactId !== undefined) setArtifactAppBuildArtifactId(draftSeed.artifactAppBuildArtifactId);
    if (draftSeed.unitTypeFilter) setUnitTypeFilter(draftSeed.unitTypeFilter);
    setSelectedArtifactId("");
    setArtifactBuildRunId("");
    setMessage(draftSeed.message || "发布计划草稿已带入。");
    setMessageTone("success");
  }, [draftSeed?.seedKey]);

  const overviewQuery = useQuery({
    queryKey: RELEASE_PLANS_QUERY_KEY,
    queryFn: getReleasePlanOverview,
  });
  const projectsQuery = useQuery({
    queryKey: BUILD_CENTER_PROJECTS_QUERY_KEY,
    queryFn: getBuildCenterOverview,
  });
  const artifactsQuery = useQuery({
    queryKey: ARTIFACT_CENTER_QUERY_KEY,
    queryFn: getArtifactCenterOverview,
  });
  const deploymentTargetsQuery = useQuery({
    queryKey: DEPLOYMENT_TARGETS_QUERY_KEY,
    queryFn: getDeploymentTargets,
  });

  const environments = useMemo(() => [...(overviewQuery.data?.environments ?? [])].sort((left, right) => left.sort_order - right.sort_order), [
    overviewQuery.data?.environments,
  ]);
  const releaseUnits = overviewQuery.data?.release_units ?? [];
  const releasePlans = overviewQuery.data?.release_plans ?? [];
  const artifacts = artifactsQuery.data?.artifacts ?? [];
  const deploymentTargets = deploymentTargetsQuery.data?.deployment_targets ?? [];
  const effectiveDeploymentTargetId = selectedDeploymentTargetId || deploymentTargets[0]?.id || "";
  const projectKeys = useMemo(
    () =>
      unique([
        "release-center",
        ...(projectsQuery.data?.projects ?? []).map((project) => project.project_key),
        ...releaseUnits.map((unit) => unit.project_key ?? "").filter(Boolean),
      ]),
    [projectsQuery.data?.projects, releaseUnits],
  );
  const selectedProjectUnits = useMemo(
    () => releaseUnits.filter((unit) => !projectKey || unit.project_key === projectKey || !unit.project_key),
    [projectKey, releaseUnits],
  );
  const filteredPlans = useMemo(() => {
    return releasePlans.filter((plan) => {
      const text = `${plan.project_key} ${plan.unit_key} ${plan.unit_type} ${plan.environment_key} ${plan.plan_key} ${plan.title} ${plan.version_name} ${plan.git_commit}`.toLowerCase();
      return (
        (environmentFilter === "全部" || plan.environment_key === environmentFilter) &&
        (unitTypeFilter === "全部" || plan.unit_type === unitTypeFilter) &&
        text.includes(query.toLowerCase())
      );
    });
  }, [environmentFilter, query, releasePlans, unitTypeFilter]);
  const latestPlanMap = useMemo(() => latestPlanByUnitEnv(releasePlans), [releasePlans]);
  const createPlanValidation = validatePlanForm({
    projectKey,
    unitKey,
    environmentKey,
    title: planTitle,
    rollout,
    targetType,
    targetValue,
    buildNumber,
  });
  const createUnitValidation = validateUnitForm({
    projectKey: newUnitProjectKey,
    unitKey: newUnitKey,
    name: newUnitName,
    unitType: newUnitType,
  });

  const createPlanMutation = useMutation({
    mutationFn: () => {
      if (createPlanValidation) throw new Error(createPlanValidation);
      return createReleasePlan(buildCreatePlanPayload());
    },
    onSuccess: async (result) => {
      setMessage(result.message_zh || `发布计划已创建：${result.plan.title}`);
      setMessageTone("success");
      await overviewQuery.refetch();
    },
    onError: (error) => showPanelError(error, "创建发布计划失败", setMessage, setMessageTone),
  });
  const createUnitMutation = useMutation({
    mutationFn: () => {
      if (createUnitValidation) throw new Error(createUnitValidation);
      return createReleaseUnit({
        project_key: newUnitProjectKey.trim(),
        unit_key: newUnitKey.trim(),
        name: newUnitName.trim(),
        unit_type: newUnitType,
        default_channel: newUnitChannel,
        enabled: true,
        metadata: { source: "admin_web" },
      });
    },
    onSuccess: async (result) => {
      setProjectKey(result.release_unit.project_key || newUnitProjectKey.trim());
      setUnitKey(result.release_unit.unit_key);
      setMessage(result.message_zh || `发布单元已保存：${result.release_unit.name}`);
      setMessageTone("success");
      await overviewQuery.refetch();
    },
    onError: (error) => showPanelError(error, "保存发布单元失败", setMessage, setMessageTone),
  });
  const planActionMutation = useMutation({
    mutationFn: ({ plan, action }: { plan: ReleasePlan; action: PlanAction }) => {
      if (action === "rollback_deploy" && !effectiveDeploymentTargetId) throw new Error("请选择部署目标");
      const payload = {
        approved_by: "admin-web",
        target_id: action === "rollback_deploy" ? effectiveDeploymentTargetId : undefined,
        dry_run: action === "rollback_deploy" ? deploymentDryRun : undefined,
        triggered_by: action === "rollback_deploy" ? "admin-web" : undefined,
        metadata: { source: "admin_web", action },
      };
      if (action === "publish") return publishReleasePlan(plan.id, payload);
      if (action === "approve") return approveReleasePlan(plan.id, payload);
      if (action === "pause") return pauseReleasePlan(plan.id, payload);
      return rollbackReleasePlan(plan.id, payload);
    },
    onSuccess: async (result) => {
      const rollbackSuffix = result.rollback_plan?.plan_key ? `：${result.rollback_plan.plan_key}` : "";
      setMessage(result.message_zh || `发布计划状态已更新${rollbackSuffix}`);
      setMessageTone("success");
      await overviewQuery.refetch();
    },
    onError: (error) => showPanelError(error, "发布计划操作失败", setMessage, setMessageTone),
  });
  const deploymentMutation = useMutation({
    mutationFn: (plan: ReleasePlan) => {
      if (!effectiveDeploymentTargetId) throw new Error("请选择部署目标");
      return createReleasePlanDeployment(plan.id, {
        target_id: effectiveDeploymentTargetId,
        dry_run: deploymentDryRun,
        triggered_by: "admin-web",
        metadata: { source: "admin_web", ui: "release_plans_panel" },
      });
    },
    onSuccess: (result) => {
      setMessage(result.message_zh || `部署记录已创建：${result.deployment_records.length} 条`);
      setMessageTone("success");
    },
    onError: (error) => showPanelError(error, "创建部署记录失败", setMessage, setMessageTone),
  });

  const mutationError = createPlanMutation.error || createUnitMutation.error || planActionMutation.error || deploymentMutation.error;
  const busy = createPlanMutation.isPending || createUnitMutation.isPending || planActionMutation.isPending || deploymentMutation.isPending;

  function buildCreatePlanPayload(): CreateReleasePlanPayload {
    const hasArtifact = [artifactName, artifactType, artifactFileName, artifactRef, artifactBuildRunId, artifactAppBuildId, artifactAppBuildArtifactId].some((value) =>
      value.trim(),
    );
    return {
      project_key: projectKey.trim(),
      unit_key: unitKey.trim(),
      environment_key: environmentKey,
      title: planTitle.trim(),
      version_name: versionName.trim(),
      build_number: buildNumber.trim() ? Number(buildNumber) : undefined,
      git_commit: gitCommit.trim(),
      channel,
      rollout_percentage: Number(rollout),
      target_type: targetType,
      target_value: targetType === "all" ? undefined : targetValue.trim(),
      created_by: "admin-web",
      artifacts: hasArtifact
        ? [
            {
              artifact_name: artifactName.trim() || `${unitKey.trim()}-artifact`,
              artifact_type: artifactType.trim() || "artifact",
              file_name: artifactFileName.trim(),
              immutable_ref: artifactRef.trim(),
              build_run_id: artifactBuildRunId.trim(),
              app_build_id: artifactAppBuildId.trim(),
              app_build_artifact_id: artifactAppBuildArtifactId.trim(),
              metadata: { source: "admin_web", artifact_center_id: selectedArtifactId || undefined },
            },
          ]
        : [],
      metadata: { source: "admin_web", ui: "release_plans_panel" },
    };
  }

  function applyArtifact(artifact?: ArtifactCenterItem) {
    if (!artifact) {
      setSelectedArtifactId("");
      setArtifactBuildRunId("");
      setArtifactAppBuildId("");
      setArtifactAppBuildArtifactId("");
      return;
    }
    setSelectedArtifactId(artifact.id);
    setArtifactName(artifact.name || artifact.file_name || "artifact");
    setArtifactType(artifact.artifact_type || "artifact");
    setArtifactFileName(artifact.file_name || "");
    setArtifactRef(artifact.immutable_ref || `${artifact.source}:${artifact.id}`);
    setArtifactBuildRunId(artifact.run_id || "");
    setArtifactAppBuildId(artifact.build_id || "");
    setArtifactAppBuildArtifactId(artifact.app_build_artifact_id || "");
    if (artifact.project_key) setProjectKey(artifact.project_key);
    if (artifact.version_name) setVersionName(artifact.version_name);
    if (artifact.build_number) setBuildNumber(String(artifact.build_number));
    if (artifact.git_commit) setGitCommit(artifact.git_commit);
    if (artifact.channel) setChannel(artifact.channel);
  }

  return (
    <div className="grid gap-4">
      {overviewQuery.isError ? <ApiErrorState error={overviewQuery.error} title="发布计划读取失败" /> : null}
      {projectsQuery.isError ? <ApiErrorState error={projectsQuery.error} title="项目列表读取失败" /> : null}
      {artifactsQuery.isError ? <ApiErrorState error={artifactsQuery.error} title="制品中心读取失败" /> : null}
      {deploymentTargetsQuery.isError ? <ApiErrorState error={deploymentTargetsQuery.error} title="部署目标读取失败" /> : null}
      {mutationError ? <ApiErrorState error={mutationError} title="发布计划操作失败" /> : null}
      <StatusMessage text={message} tone={messageTone} />

      <div className="grid gap-3 md:grid-cols-4">
        <Metric label="发布单元" value={releaseUnits.length} note={`${projectKeys.length} 个项目 / ${countEnabledUnits(releaseUnits)} 个启用`} />
        <Metric label="环境" value={environments.length} note={environments.map((item) => item.environment_key).join(" / ") || "-"} />
        <Metric label="发布计划" value={releasePlans.length} note={`${countPlansByStatus(releasePlans, "released")} 个已发布`} />
        <Metric label="待处理" value={countPendingPlans(releasePlans)} note="draft / scheduled / queued / pending" />
      </div>

      <div className="grid gap-4 xl:grid-cols-[420px_minmax(0,1fr)]">
        <div className="grid gap-4">
          <Card>
            <div className="mb-4 flex items-center gap-2 font-medium">
              <Rocket className="h-4 w-4" />
              创建发布计划
            </div>
            <div className="grid gap-2">
              <div className="grid grid-cols-2 gap-2">
                <Select label="项目" value={projectKey} onChange={setProjectKey} options={projectKeys} />
                <Select
                  label="发布单元"
                  value={unitKey}
                  onChange={setUnitKey}
                  options={selectedProjectUnits.map((unit) => ({
                    value: unit.unit_key,
                    label: `${unit.name} / ${unitTypeLabel(unit.unit_type)}`,
                  }))}
                  placeholder="暂无发布单元"
                />
              </div>
              <div className="grid grid-cols-2 gap-2">
                <Select
                  label="环境"
                  value={environmentKey}
                  onChange={setEnvironmentKey}
                  options={environments.map((environment) => ({
                    value: environment.environment_key,
                    label: `${environment.name} / ${environment.environment_key}`,
                  }))}
                  placeholder="暂无环境"
                />
                <Select label="渠道" value={channel} onChange={setChannel} options={channelOptions} />
              </div>
              <Input placeholder="发布标题" value={planTitle} onChange={(event) => setPlanTitle(event.target.value)} />
              <div className="grid grid-cols-[minmax(0,1fr)_110px] gap-2">
                <Input placeholder="版本名" value={versionName} onChange={(event) => setVersionName(event.target.value)} />
                <Input placeholder="build" type="number" value={buildNumber} onChange={(event) => setBuildNumber(event.target.value)} />
              </div>
              <Input placeholder="git commit / tag" value={gitCommit} onChange={(event) => setGitCommit(event.target.value)} />
              <div className="grid grid-cols-2 gap-2">
                <Select label="灰度比例" value={rollout} onChange={setRollout} options={rolloutOptions} />
                <Select label="目标" value={targetType} onChange={setTargetType} options={targetTypeOptions} />
              </div>
              <Input
                placeholder={targetType === "all" ? "全部目标" : "目标值"}
                value={targetValue}
                disabled={targetType === "all"}
                onChange={(event) => setTargetValue(event.target.value)}
              />
              <div className="rounded-lg border bg-muted p-3">
                <div className="mb-2 flex items-center gap-2 text-xs font-medium">
                  <Upload className="h-3.5 w-3.5" />
                  制品引用
                </div>
                <div className="grid gap-2">
                  <Select
                    label="制品"
                    value={selectedArtifactId}
                    onChange={(value) => applyArtifact(artifacts.find((artifact) => artifact.id === value))}
                    options={[
                      { value: "", label: artifactsQuery.isLoading ? "正在读取制品" : "手动填写" },
                      ...artifacts.slice(0, 80).map((artifact) => ({
                        value: artifact.id,
                        label: artifactOptionLabel(artifact),
                      })),
                    ]}
                  />
                  <div className="grid grid-cols-2 gap-2">
                    <Input placeholder="artifact name" value={artifactName} onChange={(event) => setArtifactName(event.target.value)} />
                    <Input placeholder="artifact type" value={artifactType} onChange={(event) => setArtifactType(event.target.value)} />
                  </div>
                  <Input placeholder="file name / object key" value={artifactFileName} onChange={(event) => setArtifactFileName(event.target.value)} />
                  <Input placeholder="immutable ref / sha256 / r2 key" value={artifactRef} onChange={(event) => setArtifactRef(event.target.value)} />
                  {selectedArtifactId ? (
                    <div className="grid gap-1 rounded-md border bg-white p-2 font-mono text-[11px] text-muted-foreground">
                      <span>run={artifactBuildRunId || "-"}</span>
                      <span>build={artifactAppBuildId || "-"}</span>
                      <span>artifact={artifactAppBuildArtifactId || "-"}</span>
                    </div>
                  ) : null}
                </div>
              </div>
              {createPlanValidation ? <InlineWarning text={createPlanValidation} /> : null}
              <Button
                onClick={() => createPlanMutation.mutate()}
                disabled={Boolean(createPlanValidation) || createPlanMutation.isPending || !permissions.releaseWrite}
                title={!permissions.releaseWrite ? missingPermissionText("release:write") : undefined}
              >
                <FileText className="mr-2 h-4 w-4" />
                {createPlanMutation.isPending ? "创建中" : "创建发布计划"}
              </Button>
            </div>
          </Card>

          <Card>
            <div className="mb-4 flex items-center gap-2 font-medium">
              <SlidersHorizontal className="h-4 w-4" />
              发布单元维护
            </div>
            <div className="grid gap-2">
              <div className="grid grid-cols-2 gap-2">
                <Select label="项目" value={newUnitProjectKey} onChange={setNewUnitProjectKey} options={projectKeys} />
                <Select
                  label="类型"
                  value={newUnitType}
                  onChange={(value) => setNewUnitType(value)}
                  options={unitTypeOptions}
                />
              </div>
              <div className="grid grid-cols-2 gap-2">
                <Input placeholder="unit key" value={newUnitKey} onChange={(event) => setNewUnitKey(event.target.value)} />
                <Input placeholder="显示名称" value={newUnitName} onChange={(event) => setNewUnitName(event.target.value)} />
              </div>
              <Select label="默认渠道" value={newUnitChannel} onChange={setNewUnitChannel} options={channelOptions} />
              {createUnitValidation ? <InlineWarning text={createUnitValidation} /> : null}
              <Button
                variant="secondary"
                onClick={() => createUnitMutation.mutate()}
                disabled={Boolean(createUnitValidation) || createUnitMutation.isPending || !permissions.releaseWrite}
                title={!permissions.releaseWrite ? missingPermissionText("release:write") : undefined}
              >
                {createUnitMutation.isPending ? "保存中" : "保存发布单元"}
              </Button>
              <div className="grid gap-2 pt-2">
                {releaseUnits.slice(0, 6).map((unit) => (
                  <div key={unit.id} className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2 text-xs">
                    <div className="min-w-0">
                      <div className="truncate font-medium">{unit.name}</div>
                      <div className="truncate text-muted-foreground">{unit.project_key} / {unit.unit_key}</div>
                    </div>
                    <Badge tone={unit.enabled ? "success" : "warning"}>{unitTypeLabel(unit.unit_type)}</Badge>
                  </div>
                ))}
                {!releaseUnits.length ? <EmptyBox text="暂无发布单元" /> : null}
              </div>
            </div>
          </Card>
        </div>

        <div className="grid gap-4">
          <ReleaseMatrix environments={environments} units={releaseUnits} latestPlanMap={latestPlanMap} />
          <Card>
            <div className="mb-4 grid gap-3 md:grid-cols-[110px_110px_220px_120px_minmax(0,1fr)_auto]">
              <Select
                label="环境"
                value={environmentFilter}
                onChange={setEnvironmentFilter}
                options={["全部", ...environments.map((environment) => environment.environment_key)]}
              />
              <Select label="类型" value={unitTypeFilter} onChange={setUnitTypeFilter} options={["全部", ...unitTypeOptions]} />
              <Select
                label="部署目标"
                value={effectiveDeploymentTargetId}
                onChange={setSelectedDeploymentTargetId}
                options={deploymentTargets.map((target) => ({
                  value: target.id,
                  label: deploymentTargetLabel(target),
                }))}
                placeholder="暂无部署目标"
              />
              <div className="flex h-9 items-center justify-between gap-2 rounded-lg border px-3">
                <span className="text-xs font-medium">Dry-run</span>
                <Switch checked={deploymentDryRun} onCheckedChange={setDeploymentDryRun} aria-label="Dry-run 发布计划部署" />
              </div>
              <div className="relative">
                <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                <Input
                  className="pl-9"
                  placeholder="搜索项目 / 单元 / 环境 / 版本"
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                />
              </div>
              <Button
                variant="secondary"
                onClick={() => {
                  overviewQuery.refetch();
                  artifactsQuery.refetch();
                  deploymentTargetsQuery.refetch();
                }}
                disabled={overviewQuery.isFetching || artifactsQuery.isFetching || deploymentTargetsQuery.isFetching}
              >
                <RefreshCw className="mr-2 h-4 w-4" />
                刷新
              </Button>
            </div>
            <ReleasePlanList
              plans={filteredPlans}
              busy={busy}
              deploymentTargetId={effectiveDeploymentTargetId}
              deploymentDryRun={deploymentDryRun}
              permissions={permissions}
              onAction={(plan, action) => planActionMutation.mutate({ plan, action })}
              onDeploy={(plan) => deploymentMutation.mutate(plan)}
            />
          </Card>
        </div>
      </div>
    </div>
  );
}

function ReleaseMatrix({
  environments,
  units,
  latestPlanMap,
}: {
  environments: ReleaseEnvironment[];
  units: ReleaseUnit[];
  latestPlanMap: Map<string, ReleasePlan>;
}) {
  return (
    <Card>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div className="font-medium">环境发布矩阵</div>
        <Badge>{units.length} 个单元</Badge>
      </div>
      <div className="overflow-x-auto">
        <Table>
          <thead>
            <tr>
              <Th>发布单元</Th>
              {environments.map((environment) => (
                <Th key={environment.id}>{environment.name}</Th>
              ))}
            </tr>
          </thead>
          <tbody>
            {units.map((unit) => (
              <tr key={unit.id}>
                <Td>
                  <div className="min-w-[160px]">
                    <div className="font-medium">{unit.name}</div>
                    <div className="mt-1 text-xs text-muted-foreground">{unit.project_key} / {unit.unit_key}</div>
                  </div>
                </Td>
                {environments.map((environment) => (
                  <Td key={`${unit.id}-${environment.id}`}>
                    <PlanMatrixCell plan={latestPlanMap.get(`${unit.id}:${environment.id}`)} />
                  </Td>
                ))}
              </tr>
            ))}
            {!units.length ? (
              <tr>
                <Td colSpan={Math.max(1, environments.length + 1)}>暂无发布单元</Td>
              </tr>
            ) : null}
          </tbody>
        </Table>
      </div>
    </Card>
  );
}

function PlanMatrixCell({ plan }: { plan?: ReleasePlan }) {
  if (!plan) return <span className="text-xs text-muted-foreground">未发布</span>;
  return (
    <div className="min-w-[150px] text-xs">
      <div className="mb-1 flex items-center gap-1">
        <Badge tone={releasePlanStatusTone(plan.status)}>{releasePlanStatusLabel(plan.status)}</Badge>
        <Badge>{plan.channel}</Badge>
      </div>
      <div className="truncate font-medium">{plan.version_name || plan.title}</div>
      <div className="mt-1 truncate text-muted-foreground">{formatDateTime(plan.published_at || plan.updated_at || plan.created_at)}</div>
    </div>
  );
}

function ReleasePlanList({
  plans,
  busy,
  deploymentTargetId,
  deploymentDryRun,
  permissions,
  onAction,
  onDeploy,
}: {
  plans: ReleasePlan[];
  busy: boolean;
  deploymentTargetId: string;
  deploymentDryRun: boolean;
  permissions: { releaseWrite: boolean; releaseApprove: boolean; releaseRollback: boolean; deployWrite: boolean };
  onAction: (plan: ReleasePlan, action: PlanAction) => void;
  onDeploy: (plan: ReleasePlan) => void;
}) {
  return (
    <div className="grid gap-3">
      {plans.map((plan) => {
        const released = plan.status === "released" || plan.status === "rolling_out";
        const pendingApproval = plan.status === "pending_approval";
        const requiresApproval = releasePlanRequiresApproval(plan);
        const deploymentBlockedByApproval = !deploymentDryRun && requiresApproval && !released;
        const rollbackDeployBlockedByApproval = !deploymentDryRun && requiresApproval;
        return (
          <div key={plan.id} className="rounded-lg border p-3">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <div className="font-medium">{plan.title}</div>
                  <Badge>{unitTypeLabel(plan.unit_type)}</Badge>
                  <Badge>{plan.environment_key || "-"}</Badge>
                  <Badge tone={releasePlanStatusTone(plan.status)}>{releasePlanStatusLabel(plan.status)}</Badge>
                </div>
                <div className="mt-1 truncate text-xs text-muted-foreground">
                  {plan.project_key} / {plan.unit_key} / {plan.plan_key}
                </div>
              </div>
              <div className="flex flex-wrap gap-2">
                <Button variant="secondary" size="sm" onClick={() => onAction(plan, "publish")} disabled={busy || released || pendingApproval || !permissions.releaseWrite} title={!permissions.releaseWrite ? missingPermissionText("release:write") : undefined}>
                  <ShieldCheck className="mr-2 h-3.5 w-3.5" />
                  发布
                </Button>
                <Button variant="secondary" size="sm" onClick={() => onAction(plan, "approve")} disabled={busy || !pendingApproval || !permissions.releaseApprove} title={!permissions.releaseApprove ? missingPermissionText("release:approve") : undefined}>
                  <ShieldCheck className="mr-2 h-3.5 w-3.5" />
                  审批
                </Button>
                <Button variant="secondary" size="sm" onClick={() => onAction(plan, "pause")} disabled={busy || !released || plan.status === "paused" || !permissions.releaseWrite} title={!permissions.releaseWrite ? missingPermissionText("release:write") : undefined}>
                  <PauseCircle className="mr-2 h-3.5 w-3.5" />
                  暂停
                </Button>
                <Button variant="secondary" size="sm" onClick={() => onAction(plan, "rollback")} disabled={busy || plan.status === "rolled_back" || !permissions.releaseRollback} title={!permissions.releaseRollback ? missingPermissionText("release:rollback") : undefined}>
                  <History className="mr-2 h-3.5 w-3.5" />
                  回滚计划
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => onAction(plan, "rollback_deploy")}
                  disabled={busy || !deploymentTargetId || plan.status === "rolled_back" || rollbackDeployBlockedByApproval || !permissions.releaseRollback}
                  title={!permissions.releaseRollback ? missingPermissionText("release:rollback") : undefined}
                >
                  <History className="mr-2 h-3.5 w-3.5" />
                  {deploymentDryRun ? "回滚部署" : "回滚投递"}
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => onDeploy(plan)}
                  disabled={busy || !deploymentTargetId || !(plan.artifacts?.length) || deploymentBlockedByApproval || !permissions.deployWrite}
                  title={!permissions.deployWrite ? missingPermissionText("deploy:write") : undefined}
                >
                  <Cloud className="mr-2 h-3.5 w-3.5" />
                  {deploymentDryRun ? "部署" : "投递"}
                </Button>
              </div>
            </div>
            <div className="mt-3 grid gap-2 text-xs md:grid-cols-4">
              <Info label="版本" value={plan.version_name || "-"} />
              <Info label="build" value={String(plan.build_number ?? "-")} />
              <Info label="灰度" value={`${plan.rollout_percentage ?? 100}%`} />
              <Info label="目标" value={releaseTargetLabel(plan.target_type, plan.target_value)} />
            </div>
            <div className="mt-3 grid gap-2 text-xs md:grid-cols-3">
              <Info label="Git" value={plan.git_commit || "-"} />
              <Info label="发布人" value={plan.approved_by || plan.created_by || "-"} />
              <Info label="更新时间" value={formatDateTime(plan.updated_at || plan.created_at)} />
            </div>
            <PlanArtifactList plan={plan} />
          </div>
        );
      })}
      {!plans.length ? <EmptyBox text="暂无发布计划" /> : null}
    </div>
  );
}

function PlanArtifactList({ plan }: { plan: ReleasePlan }) {
  const artifacts = plan.artifacts ?? [];
  if (!artifacts.length) return null;
  return (
    <div className="mt-3 grid gap-2 rounded-lg border bg-muted/60 p-3 text-xs">
      <div className="flex items-center justify-between gap-3">
        <div className="font-medium">制品</div>
        <Badge>{artifacts.length} 个</Badge>
      </div>
      <div className="grid gap-2 md:grid-cols-2">
        {artifacts.map((artifact) => (
          <div key={artifact.id || artifact.artifact_name} className="min-w-0 rounded-md border bg-white p-2">
            <div className="flex items-center justify-between gap-2">
              <span className="truncate font-medium">{artifact.artifact_name}</span>
              <Badge>{artifact.artifact_type}</Badge>
            </div>
            <div className="mt-1 truncate text-[11px] text-muted-foreground">{artifact.file_name || artifact.immutable_ref || "-"}</div>
          </div>
        ))}
      </div>
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

function Metric({ label, value, note }: { label: string; value: string | number; note: string }) {
  return (
    <Card>
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-2 truncate text-xl font-semibold">{value}</div>
      <div className="mt-1 truncate text-xs text-muted-foreground">{note}</div>
    </Card>
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

function InlineWarning({ text }: { text: string }) {
  return (
    <div className="flex items-start gap-2 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs leading-5 text-amber-800">
      <span>{text}</span>
    </div>
  );
}

function EmptyBox({ text }: { text: string }) {
  return <div className="rounded-lg border bg-muted p-4 text-sm text-muted-foreground">{text}</div>;
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

function validatePlanForm({
  projectKey,
  unitKey,
  environmentKey,
  title,
  rollout,
  targetType,
  targetValue,
  buildNumber,
}: {
  projectKey: string;
  unitKey: string;
  environmentKey: string;
  title: string;
  rollout: string;
  targetType: string;
  targetValue: string;
  buildNumber: string;
}) {
  if (!projectKey.trim() || !unitKey.trim() || !environmentKey.trim() || !title.trim()) return "项目、发布单元、环境和标题不能为空";
  const rolloutNumber = Number(rollout);
  if (!Number.isInteger(rolloutNumber) || rolloutNumber < 1 || rolloutNumber > 100) return "灰度比例必须在 1 到 100 之间";
  if (targetType !== "all" && !targetValue.trim()) return "定向发布需要填写目标值";
  if (buildNumber.trim()) {
    const value = Number(buildNumber);
    if (!Number.isInteger(value) || value < 0) return "build 必须是非负整数";
  }
  return "";
}

function validateUnitForm({
  projectKey,
  unitKey,
  name,
  unitType,
}: {
  projectKey: string;
  unitKey: string;
  name: string;
  unitType: ReleaseUnitType;
}) {
  if (!projectKey.trim() || !unitKey.trim() || !name.trim() || !unitType.trim()) return "项目、unit key、显示名称和类型不能为空";
  return "";
}

function latestPlanByUnitEnv(plans: ReleasePlan[]) {
  const map = new Map<string, ReleasePlan>();
  for (const plan of plans) {
    const key = `${plan.release_unit_id}:${plan.environment_id}`;
    const current = map.get(key);
    if (!current || timestamp(plan.updated_at || plan.created_at) > timestamp(current.updated_at || current.created_at)) {
      map.set(key, plan);
    }
  }
  return map;
}

function releasePlanStatusTone(status?: ReleasePlanStatus): "default" | "success" | "warning" | "danger" {
  if (status === "released") return "success";
  if (status === "scheduled" || status === "queued" || status === "pending_approval" || status === "rolling_out" || status === "paused") return "warning";
  if (status === "recalled" || status === "rolled_back" || status === "archived") return "danger";
  return "default";
}

function releasePlanStatusLabel(status?: ReleasePlanStatus) {
  return (
    {
      draft: "草稿",
      scheduled: "已定时",
      queued: "排队中",
      pending_approval: "待审批",
      released: "已发布",
      rolling_out: "灰度中",
      paused: "已暂停",
      recalled: "已召回",
      rolled_back: "已回滚",
      archived: "已归档",
    }[status ?? "draft"] ?? status
  );
}

function releasePlanRequiresApproval(plan: ReleasePlan) {
  const environmentKey = (plan.environment_key || "").trim().toLowerCase();
  return Boolean(plan.environment_requires_approval) || environmentKey === "prod" || environmentKey === "production";
}

function unitTypeLabel(type?: ReleaseUnitType) {
  return unitTypeOptions.find((item) => item.value === type)?.label ?? type ?? "-";
}

function releaseTargetLabel(type?: string, value?: string) {
  if (!type || type === "all") return "全部";
  return `${targetTypeOptions.find((item) => item.value === type)?.label ?? type}: ${value || "-"}`;
}

function artifactOptionLabel(artifact: ArtifactCenterItem) {
  const scope = artifact.project_key || artifact.app_key || artifact.source;
  const version = artifact.version_name ? `${artifact.version_name}${artifact.build_number ? ` #${artifact.build_number}` : ""}` : artifact.artifact_type;
  return `${scope} / ${artifact.name || artifact.file_name || artifact.id.slice(0, 8)} / ${version}`;
}

function deploymentTargetLabel(target: DeploymentTarget) {
  return `${target.name || target.target_key} / ${target.provider} / ${target.environment}`;
}

function countEnabledUnits(units: ReleaseUnit[]) {
  return units.filter((unit) => unit.enabled).length;
}

function countPlansByStatus(plans: ReleasePlan[], status: ReleasePlanStatus) {
  return plans.filter((plan) => plan.status === status).length;
}

function countPendingPlans(plans: ReleasePlan[]) {
  return plans.filter((plan) => ["draft", "scheduled", "queued", "pending_approval"].includes(plan.status)).length;
}

function unique(values: string[]) {
  return Array.from(new Set(values.filter(Boolean)));
}

function timestamp(value?: string) {
  return value ? new Date(value).getTime() || 0 : 0;
}

function defaultVersionName() {
  const now = new Date();
  const yyyy = now.getFullYear();
  const mm = String(now.getMonth() + 1).padStart(2, "0");
  const dd = String(now.getDate()).padStart(2, "0");
  return `${yyyy}.${mm}.${dd}`;
}

function showPanelError(
  error: unknown,
  fallback: string,
  setMessage: (value: string) => void,
  setTone: (value: "default" | "success" | "warning" | "danger") => void,
) {
  setMessage(error instanceof Error ? error.message : fallback);
  setTone("danger");
}
