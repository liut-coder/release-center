import { useMemo, useState, type ReactNode } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { CheckCircle2, Cloud, GitBranch, Hammer, Loader2, Package, Play, RefreshCw, Server, XCircle } from "lucide-react";
import {
  createBuildCenterRun,
  getBuildCenterRun,
  getBuildCenterRunLogs,
  getBuildCenterOverview,
  type BuildCenterOverview,
  type BuildCenterProject,
  type BuildCenterRun,
  type BuildProfile,
  type CodeRepository,
  type DeploymentTarget,
} from "@/api/buildCenter";
import { PageHeader } from "@/components/layout/PageHeader";
import { ApiErrorState } from "@/components/stable/StableAdminComponents";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Table, Td, Th } from "@/components/ui/Table";
import { cn } from "@/lib/cn";
import { formatDateTime } from "@/lib/format";

const actionOptions = ["all", "status", "fetch", "prepare", "build", "image", "upload", "verify"] as const;
const runningStatuses = new Set(["queued", "running"]);

type ProjectProfileRow = {
  project: BuildCenterProject;
  profile?: BuildProfile;
};

export function BuildCenterPage() {
  const [action, setAction] = useState<(typeof actionOptions)[number]>("all");
  const [message, setMessage] = useState("");
  const [messageTone, setMessageTone] = useState<"default" | "success" | "warning" | "danger">("default");
  const [selectedRunId, setSelectedRunId] = useState("");

  const overviewQuery = useQuery({
    queryKey: ["build-center-overview"],
    queryFn: getBuildCenterOverview,
    refetchInterval: (query) => (hasActiveRun(query.state.data) ? 5_000 : false),
  });
  const overview = overviewQuery.data ?? emptyOverview;
  const projects = overview.projects ?? [];
  const deploymentTargets = overview.deployment_targets ?? [];
  const profileRows = useMemo(() => projectProfileRows(projects), [projects]);
  const recentRuns = useMemo(() => sortRuns(projects.flatMap((project) => project.recent_runs ?? [])).slice(0, 12), [projects]);
  const selectedRunFallback = useMemo(() => recentRuns.find((run) => run.id === selectedRunId), [recentRuns, selectedRunId]);
  const metrics = useMemo(() => buildMetrics(projects, deploymentTargets), [deploymentTargets, projects]);
  const selectedRunQuery = useQuery({
    queryKey: ["build-center-run", selectedRunId],
    queryFn: () => getBuildCenterRun(selectedRunId),
    enabled: Boolean(selectedRunId),
    refetchInterval: (query) => {
      const run = query.state.data?.run ?? selectedRunFallback;
      return run && runningStatuses.has(run.status) ? 5_000 : false;
    },
  });
  const selectedLogsQuery = useQuery({
    queryKey: ["build-center-run-logs", selectedRunId],
    queryFn: () => getBuildCenterRunLogs(selectedRunId),
    enabled: Boolean(selectedRunId),
    refetchInterval: () => {
      const run = selectedRunQuery.data?.run ?? selectedRunFallback;
      return run && runningStatuses.has(run.status) ? 5_000 : false;
    },
  });

  const runMutation = useMutation({
    mutationFn: ({ project, profile }: { project: BuildCenterProject; profile: BuildProfile }) =>
      createBuildCenterRun(project.project_key, {
        profile_key: profile.profile_key,
        action,
        git_ref: profile.default_ref,
        version_name: profile.default_version_name,
        version_code: profile.default_version_code,
        channel: profile.default_channel || project.default_channel,
        started_by: "admin-ui",
      }),
    onSuccess: async (result) => {
      setMessage(`运行已创建：${shortId(result.run.id)} / ${result.run.status}`);
      setMessageTone("success");
      setSelectedRunId(result.run.id);
      await overviewQuery.refetch();
    },
    onError: (error) => {
      setMessage(error instanceof Error ? error.message : "创建运行失败");
      setMessageTone("danger");
    },
  });

  return (
    <>
      <PageHeader title="构建中心">
        <select
          className="h-9 rounded-lg border bg-white px-2.5 text-xs font-medium outline-none focus:border-black"
          value={action}
          onChange={(event) => setAction(event.target.value as (typeof actionOptions)[number])}
          aria-label="构建动作"
        >
          {actionOptions.map((item) => (
            <option key={item} value={item}>
              {actionLabel(item)}
            </option>
          ))}
        </select>
        <Button variant="secondary" onClick={() => overviewQuery.refetch()} disabled={overviewQuery.isFetching}>
          <RefreshCw className={cn("mr-2 h-4 w-4", overviewQuery.isFetching && "animate-spin")} />
          刷新
        </Button>
      </PageHeader>

      {overviewQuery.isError ? <ApiErrorState error={overviewQuery.error} title="构建中心数据读取失败" /> : null}
      {runMutation.error ? <ApiErrorState error={runMutation.error} title="构建运行创建失败" /> : null}

      <div className="mb-4 grid gap-3 md:grid-cols-4">
        <Metric label="项目" value={metrics.projects} note={`${metrics.repositories} 个仓库`} icon={Package} />
        <Metric label="构建配置" value={metrics.profiles} note={`${metrics.enabledProfiles} 个启用`} icon={Hammer} />
        <Metric label="运行中" value={metrics.activeRuns} note={`${metrics.recentRuns} 条运行记录`} icon={Loader2} active={metrics.activeRuns > 0} />
        <Metric label="部署目标" value={metrics.targets} note={`${metrics.cloudflareTargets} 个 Cloudflare`} icon={Cloud} />
      </div>

      <div className="mb-4">
        <StatusMessage tone={messageTone} text={message || overview.message_zh || "构建中心已连接项目、GitHub 仓库、构建配置、运行记录和部署目标。"} />
      </div>

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
        <Card>
          <SectionTitle title="项目流水线" badge={`${profileRows.length} 个配置`} />
          <Table>
            <thead>
              <tr>
                <Th className="w-[18%]">项目</Th>
                <Th className="w-[23%]">仓库</Th>
                <Th className="w-[19%]">构建配置</Th>
                <Th className="w-[14%]">默认版本</Th>
                <Th className="w-[14%]">部署目标</Th>
                <Th className="w-[12%] text-right">操作</Th>
              </tr>
            </thead>
            <tbody>
              {profileRows.map((row) => (
                <tr key={`${row.project.id}:${row.profile?.id ?? "empty"}`}>
                  <Td>
                    <div className="min-w-0">
                      <div className="truncate font-medium">{row.project.name}</div>
                      <div className="truncate font-mono text-[11px] text-muted-foreground">{row.project.project_key}</div>
                    </div>
                  </Td>
                  <Td>
                    <RepositoryCell repository={primaryRepository(row.project)} />
                  </Td>
                  <Td>
                    {row.profile ? (
                      <div className="min-w-0">
                        <div className="truncate font-medium">{row.profile.name}</div>
                        <div className="mt-1 flex flex-wrap gap-1">
                          <Badge>{row.profile.stack_type || "-"}</Badge>
                          <Badge tone={row.profile.enabled === false ? "warning" : "success"}>{row.profile.build_type || row.profile.profile_key}</Badge>
                        </div>
                      </div>
                    ) : (
                      <span className="text-muted-foreground">未配置</span>
                    )}
                  </Td>
                  <Td>
                    <div className="truncate">{row.profile?.default_version_name || "-"}</div>
                    <div className="text-[11px] text-muted-foreground">
                      {row.profile?.default_ref || row.project.default_channel || "-"} / {row.profile?.default_channel || row.project.default_channel || "-"}
                    </div>
                  </Td>
                  <Td>
                    <TargetSummary targets={row.project.deployment_targets ?? []} />
                  </Td>
                  <Td className="text-right">
                    <Button
                      size="sm"
                      disabled={!row.profile || row.profile.enabled === false || runMutation.isPending}
                      onClick={() => row.profile && runMutation.mutate({ project: row.project, profile: row.profile })}
                    >
                      <Play className="mr-1.5 h-3.5 w-3.5" />
                      触发
                    </Button>
                  </Td>
                </tr>
              ))}
              {profileRows.length === 0 ? <EmptyTableRow colSpan={6} label={overviewQuery.isLoading ? "正在读取项目" : "暂无构建项目"} /> : null}
            </tbody>
          </Table>
        </Card>

        <Card>
          <SectionTitle title="接入状态" badge={metrics.webhookEnabled ? "已接入" : "待接入"} />
          <div className="grid gap-2">
            <IntegrationRow icon={GitBranch} label="GitHub" value={`${metrics.webhookEnabled}/${metrics.repositories}`} note="webhook" />
            <IntegrationRow icon={Cloud} label="Cloudflare" value={String(metrics.cloudflareTargets)} note="targets" />
            <IntegrationRow icon={Server} label="机器 API" value={String(metrics.activeRuns)} note="active runs" />
          </div>
        </Card>
      </div>

      <div className="mt-4 grid gap-4 xl:grid-cols-[minmax(0,1fr)_420px]">
        <Card>
          <SectionTitle title="最近运行" badge={`${recentRuns.length} 条`} />
          <Table>
            <thead>
              <tr>
                <Th className="w-[18%]">运行</Th>
                <Th className="w-[13%]">状态</Th>
                <Th className="w-[14%]">动作</Th>
                <Th className="w-[20%]">版本</Th>
                <Th className="w-[12%]">产物</Th>
                <Th className="w-[10%]">耗时</Th>
                <Th className="w-[11%]">时间</Th>
                <Th className="w-[8%] text-right">操作</Th>
              </tr>
            </thead>
            <tbody>
              {recentRuns.map((run) => (
                <tr key={run.id} className={cn(selectedRunId === run.id && "bg-muted/40")}>
                  <Td>
                    <div className="truncate font-mono text-[11px]">{shortId(run.id)}</div>
                    <div className="truncate text-[11px] text-muted-foreground">{projectName(projects, run.project_id)}</div>
                  </Td>
                  <Td>
                    <RunStatusBadge status={run.status} />
                  </Td>
                  <Td>{actionLabel(run.action)}</Td>
                  <Td>
                    <div className="truncate">{run.version_name || "-"}</div>
                    <div className="truncate font-mono text-[11px] text-muted-foreground">{run.git_commit || run.git_ref || "-"}</div>
                  </Td>
                  <Td>{run.artifacts?.length ?? 0}</Td>
                  <Td>{formatDuration(run.duration_ms)}</Td>
                  <Td>{formatDateTime(run.created_at).slice(5)}</Td>
                  <Td className="text-right">
                    <Button variant="secondary" size="sm" onClick={() => setSelectedRunId(run.id)}>
                      查看
                    </Button>
                  </Td>
                </tr>
              ))}
              {recentRuns.length === 0 ? <EmptyTableRow colSpan={8} label={overviewQuery.isLoading ? "正在读取运行记录" : "暂无运行记录"} /> : null}
            </tbody>
          </Table>
        </Card>

        <div className="grid gap-4">
          <RunDetailPanel
            run={selectedRunQuery.data?.run ?? selectedRunFallback}
            logs={selectedLogsQuery.data?.lines ?? []}
            logsLoading={selectedLogsQuery.isFetching}
            logsTruncated={Boolean(selectedLogsQuery.data?.truncated)}
            error={selectedRunQuery.error ?? selectedLogsQuery.error}
            onRefresh={() => {
              if (!selectedRunId) return;
              selectedRunQuery.refetch();
              selectedLogsQuery.refetch();
            }}
          />
          <Card>
            <SectionTitle title="部署目标" badge={`${deploymentTargets.length} 个`} />
            <div className="grid gap-2">
              {deploymentTargets.map((target) => (
                <DeploymentTargetItem key={target.id} target={target} />
              ))}
              {deploymentTargets.length === 0 ? <EmptyBlock label={overviewQuery.isLoading ? "正在读取部署目标" : "暂无部署目标"} /> : null}
            </div>
          </Card>
        </div>
      </div>
    </>
  );
}

const emptyOverview: BuildCenterOverview = {
  projects: [],
  deployment_targets: [],
};

function Metric({
  label,
  value,
  note,
  icon: Icon,
  active = false,
}: {
  label: string;
  value: string | number;
  note: string;
  icon: typeof Package;
  active?: boolean;
}) {
  return (
    <div className="rounded-lg border bg-white p-3">
      <div className="flex items-center justify-between gap-3">
        <span className="text-xs text-muted-foreground">{label}</span>
        <Icon className={cn("h-4 w-4 text-muted-foreground", active && "animate-spin text-foreground")} />
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

function RepositoryCell({ repository }: { repository?: CodeRepository }) {
  if (!repository) return <span className="text-muted-foreground">未配置</span>;
  return (
    <div className="min-w-0">
      <div className="truncate font-medium">{repository.repo_full_name || repository.repo_url}</div>
      <div className="mt-1 flex min-w-0 items-center gap-1 text-[11px] text-muted-foreground">
        <GitBranch className="h-3.5 w-3.5 shrink-0" />
        <span className="truncate">
          {repository.provider} / {repository.default_ref || "-"}
        </span>
      </div>
    </div>
  );
}

function TargetSummary({ targets }: { targets: DeploymentTarget[] }) {
  if (!targets.length) return <span className="text-muted-foreground">未配置</span>;
  const enabled = targets.filter((target) => target.enabled !== false).length;
  const cloudflare = targets.filter((target) => target.provider.startsWith("cloudflare_")).length;
  return (
    <div className="min-w-0">
      <div className="truncate">{enabled}/{targets.length} 启用</div>
      <div className="text-[11px] text-muted-foreground">{cloudflare} Cloudflare</div>
    </div>
  );
}

function IntegrationRow({ icon: Icon, label, value, note }: { icon: typeof GitBranch; label: string; value: string; note: string }) {
  return (
    <div className="flex h-16 items-center justify-between gap-3 rounded-lg border px-3">
      <div className="flex min-w-0 items-center gap-2">
        <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
        <div className="min-w-0">
          <div className="truncate text-sm font-medium">{label}</div>
          <div className="truncate text-[11px] text-muted-foreground">{note}</div>
        </div>
      </div>
      <div className="font-mono text-sm font-semibold">{value}</div>
    </div>
  );
}

function RunStatusBadge({ status }: { status: string }) {
  const tone = status === "success" ? "success" : status === "failed" || status === "canceled" ? "danger" : runningStatuses.has(status) ? "warning" : "default";
  const Icon = status === "success" ? CheckCircle2 : status === "failed" || status === "canceled" ? XCircle : runningStatuses.has(status) ? Loader2 : Package;
  return (
    <Badge tone={tone}>
      <Icon className={cn("mr-1 h-3 w-3", runningStatuses.has(status) && "animate-spin")} />
      {statusLabel(status)}
    </Badge>
  );
}

function DeploymentTargetItem({ target }: { target: DeploymentTarget }) {
  return (
    <div className="rounded-lg border p-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="truncate text-sm font-medium">{target.name}</div>
          <div className="mt-1 truncate font-mono text-[11px] text-muted-foreground">{target.target_key}</div>
        </div>
        <Badge tone={target.enabled === false ? "warning" : "success"}>{target.enabled === false ? "停用" : target.environment || "启用"}</Badge>
      </div>
      <div className="mt-3 grid gap-2 text-xs">
        <InfoRow label="Provider" value={target.provider || "-"} />
        <InfoRow label="Pages" value={target.cloudflare_project_name || "-"} />
        <InfoRow label="Worker" value={target.cloudflare_script_name || "-"} />
        <InfoRow label="R2" value={target.cloudflare_bucket_name || "-"} />
        <InfoRow label="Endpoint" value={target.endpoint_url || "-"} />
      </div>
    </div>
  );
}

function RunDetailPanel({
  run,
  logs,
  logsLoading,
  logsTruncated,
  error,
  onRefresh,
}: {
  run?: BuildCenterRun;
  logs: string[];
  logsLoading: boolean;
  logsTruncated: boolean;
  error: unknown;
  onRefresh: () => void;
}) {
  const artifacts = run?.artifacts ?? [];
  const logText = logs.join("\n");
  return (
    <Card>
      <div className="mb-3 flex items-center justify-between gap-3">
        <div className="font-medium">运行详情</div>
        <div className="flex items-center gap-2">
          <Badge>{run ? shortId(run.id) : "未选择"}</Badge>
          <Button variant="secondary" size="sm" disabled={!run || logsLoading} onClick={onRefresh}>
            <RefreshCw className={cn("mr-1.5 h-3.5 w-3.5", logsLoading && "animate-spin")} />
            刷新
          </Button>
        </div>
      </div>
      {!run ? <EmptyBlock label="选择一条运行查看日志和产物" /> : null}
      {run ? (
        <div className="grid gap-3">
          {error ? <InlineError error={error} /> : null}
          <div className="grid gap-2 text-xs">
            <InfoRow label="Status" value={<RunStatusBadge status={run.status} />} />
            <InfoRow label="Action" value={actionLabel(run.action)} />
            <InfoRow label="Ref" value={run.git_commit || run.git_ref || "-"} />
            <InfoRow label="Version" value={`${run.version_name || "-"} / ${run.version_code || 0}`} />
            <InfoRow label="Duration" value={formatDuration(run.duration_ms)} />
            <InfoRow label="Upload" value={run.upload_status || "-"} />
            {run.error_message ? <InfoRow label="Error" value={run.error_message} /> : null}
          </div>
          <div>
            <div className="mb-2 flex items-center justify-between gap-3 text-xs">
              <span className="font-medium">产物</span>
              <Badge>{artifacts.length}</Badge>
            </div>
            <div className="grid gap-1.5">
              {artifacts.map((artifact) => (
                <div key={artifact.id} className="rounded-lg border px-2.5 py-2 text-xs">
                  <div className="truncate font-medium">{artifact.name || artifact.file_name}</div>
                  <div className="mt-1 flex flex-wrap gap-2 text-[11px] text-muted-foreground">
                    <span>{artifact.artifact_type}</span>
                    <span>{formatSize(artifact.size_bytes)}</span>
                    <span className="font-mono">{artifact.sha256 ? artifact.sha256.slice(0, 12) : "-"}</span>
                  </div>
                </div>
              ))}
              {artifacts.length === 0 ? <EmptyBlock label="暂无产物" /> : null}
            </div>
          </div>
          <div>
            <div className="mb-2 flex items-center justify-between gap-3 text-xs">
              <span className="font-medium">日志</span>
              <Badge>{logsTruncated ? "Tail" : `${logs.length} 行`}</Badge>
            </div>
            <pre className="max-h-72 min-h-28 overflow-auto rounded-lg bg-zinc-950 p-3 font-mono text-[11px] leading-5 text-zinc-100">
              {logText || (logsLoading ? "正在读取日志" : "暂无日志")}
            </pre>
          </div>
        </div>
      ) : null}
    </Card>
  );
}

function InlineError({ error }: { error: unknown }) {
  return <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700">{error instanceof Error ? error.message : "请求失败"}</div>;
}

function InfoRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="grid grid-cols-[76px_minmax(0,1fr)] gap-2">
      <span className="text-muted-foreground">{label}</span>
      <span className="min-w-0 truncate font-mono">{value}</span>
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

function EmptyBlock({ label }: { label: string }) {
  return <div className="rounded-lg border border-dashed py-8 text-center text-xs text-muted-foreground">{label}</div>;
}

function StatusMessage({ text, tone }: { text: string; tone: "default" | "success" | "warning" | "danger" }) {
  const styles: Record<typeof tone, string> = {
    default: "border-border bg-white text-foreground",
    success: "border-black/10 bg-black text-white",
    warning: "border-zinc-300 bg-zinc-100 text-zinc-700",
    danger: "border-red-200 bg-red-50 text-red-700",
  };
  return <div className={cn("rounded-lg border px-3 py-2 text-xs", styles[tone])}>{text}</div>;
}

function projectProfileRows(projects: BuildCenterProject[]): ProjectProfileRow[] {
  return projects.flatMap<ProjectProfileRow>((project) => {
    const profiles = project.build_profiles ?? [];
    if (!profiles.length) return [{ project, profile: undefined }];
    return profiles.map((profile) => ({ project, profile }));
  });
}

function primaryRepository(project: BuildCenterProject) {
  return (project.repositories ?? [])[0];
}

function projectName(projects: BuildCenterProject[], projectId: string) {
  return projects.find((project) => project.id === projectId)?.name ?? projectId;
}

function buildMetrics(projects: BuildCenterProject[], targets: DeploymentTarget[]) {
  const repositories = projects.flatMap((project) => project.repositories ?? []);
  const profiles = projects.flatMap((project) => project.build_profiles ?? []);
  const runs = projects.flatMap((project) => project.recent_runs ?? []);
  return {
    projects: projects.length,
    repositories: repositories.length,
    webhookEnabled: repositories.filter((repository) => repository.webhook_enabled).length,
    profiles: profiles.length,
    enabledProfiles: profiles.filter((profile) => profile.enabled !== false).length,
    activeRuns: runs.filter((run) => runningStatuses.has(run.status)).length,
    recentRuns: runs.length,
    targets: targets.length,
    cloudflareTargets: targets.filter((target) => target.provider.startsWith("cloudflare_")).length,
  };
}

function hasActiveRun(overview?: BuildCenterOverview) {
  return Boolean(overview?.projects?.some((project) => project.recent_runs?.some((run) => runningStatuses.has(run.status))));
}

function sortRuns(runs: BuildCenterRun[]) {
  return [...runs].sort((a, b) => dateValue(b.created_at) - dateValue(a.created_at));
}

function dateValue(value?: string) {
  if (!value) return 0;
  const time = new Date(value).getTime();
  return Number.isNaN(time) ? 0 : time;
}

function shortId(id?: string) {
  return id ? id.slice(0, 8) : "-";
}

function formatDuration(durationMs?: number) {
  if (!durationMs || durationMs < 0) return "-";
  if (durationMs < 1000) return `${durationMs}ms`;
  const seconds = Math.round(durationMs / 1000);
  if (seconds < 60) return `${seconds}s`;
  return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
}

function formatSize(bytes?: number) {
  if (!bytes || bytes <= 0) return "-";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} GB`;
}

function actionLabel(action?: string) {
  const labels: Record<string, string> = {
    all: "完整流水线",
    status: "状态检查",
    fetch: "拉取代码",
    prepare: "准备环境",
    build: "构建产物",
    image: "镜像构建",
    upload: "上传产物",
    verify: "发布校验",
  };
  return labels[action || ""] ?? action ?? "-";
}

function statusLabel(status: string) {
  const labels: Record<string, string> = {
    queued: "排队",
    running: "运行中",
    success: "成功",
    failed: "失败",
    canceled: "取消",
  };
  return labels[status] ?? status;
}
