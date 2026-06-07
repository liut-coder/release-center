import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Copy, Download, FileArchive, Link2, Package, RefreshCw, Rocket, Search, ShieldCheck } from "lucide-react";
import { getArtifactCenterItem, getArtifactCenterOverview, type ArtifactCenterItem } from "@/api/artifacts";
import type { ReleasePlanDraftSeed } from "@/pages/ReleasePlansPanel";
import { PageHeader } from "@/components/layout/PageHeader";
import { ApiErrorState } from "@/components/stable/StableAdminComponents";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Table, Td, Th } from "@/components/ui/Table";
import { cn } from "@/lib/cn";
import { formatDateTime } from "@/lib/format";

const allOption = "全部";

export function ArtifactCenterPage({ onCreateReleasePlan }: { onCreateReleasePlan?: (seed: ReleasePlanDraftSeed) => void } = {}) {
  const [query, setQuery] = useState("");
  const [source, setSource] = useState(allOption);
  const [artifactType, setArtifactType] = useState(allOption);
  const [status, setStatus] = useState(allOption);
  const [project, setProject] = useState(allOption);
  const [selectedId, setSelectedId] = useState("");
  const [copiedRef, setCopiedRef] = useState("");

  const overviewQuery = useQuery({
    queryKey: ["artifact-center-overview"],
    queryFn: getArtifactCenterOverview,
  });
  const artifacts = overviewQuery.data?.artifacts ?? [];
  const filteredArtifacts = useMemo(
    () => filterArtifacts(artifacts, { query, source, artifactType, status, project }),
    [artifactType, artifacts, project, query, source, status],
  );
  const selectedFallback = useMemo(() => {
    if (selectedId) return artifacts.find((artifact) => artifact.id === selectedId);
    return filteredArtifacts[0];
  }, [artifacts, filteredArtifacts, selectedId]);
  const detailQuery = useQuery({
    queryKey: ["artifact-center-item", selectedId],
    queryFn: () => getArtifactCenterItem(selectedId),
    enabled: Boolean(selectedId),
  });
  const selectedArtifact = detailQuery.data?.artifact ?? selectedFallback;
  const metrics = useMemo(() => artifactMetrics(artifacts), [artifacts]);
  const options = useMemo(() => artifactFilterOptions(artifacts), [artifacts]);

  return (
    <>
      <PageHeader title="制品中心">
        <Button variant="secondary" onClick={() => overviewQuery.refetch()} disabled={overviewQuery.isFetching}>
          <RefreshCw className={cn("mr-2 h-4 w-4", overviewQuery.isFetching && "animate-spin")} />
          刷新
        </Button>
      </PageHeader>

      {overviewQuery.isError ? <ApiErrorState error={overviewQuery.error} title="制品中心数据读取失败" /> : null}
      {detailQuery.isError ? <ApiErrorState error={detailQuery.error} title="制品详情读取失败" /> : null}

      <div className="mb-4 grid gap-3 md:grid-cols-4">
        <Metric label="制品" value={metrics.total} note={`${metrics.types} 种类型`} icon={Package} />
        <Metric label="构建中心" value={metrics.buildCenter} note={`${metrics.appBuild} 个 App 构建制品`} icon={FileArchive} />
        <Metric label="已上传" value={metrics.uploaded} note={`${metrics.pending} 个待上传`} icon={ShieldCheck} />
        <Metric label="总大小" value={formatBytes(metrics.totalSize)} note={`${metrics.withSha} 个有 SHA`} icon={Download} />
      </div>

      <Card className="mb-4">
        <div className="grid gap-2 lg:grid-cols-[minmax(220px,1fr)_140px_140px_140px_160px]">
          <div className="relative">
            <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input className="pl-8" placeholder="搜索制品 / 版本 / SHA / 路径" value={query} onChange={(event) => setQuery(event.target.value)} />
          </div>
          <Select label="来源" value={source} onChange={setSource} options={[allOption, "build_center", "app_build"]} />
          <Select label="类型" value={artifactType} onChange={setArtifactType} options={[allOption, ...options.types]} />
          <Select label="状态" value={status} onChange={setStatus} options={[allOption, ...options.statuses]} />
          <Select label="项目" value={project} onChange={setProject} options={[allOption, ...options.projects]} />
        </div>
      </Card>

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_380px]">
        <Card>
          <div className="mb-3 flex items-center justify-between gap-3">
            <div className="font-medium">制品列表</div>
            <Badge>{filteredArtifacts.length} / {artifacts.length} 条</Badge>
          </div>
          <ArtifactTable artifacts={filteredArtifacts} selectedId={selectedArtifact?.id ?? ""} onSelect={setSelectedId} loading={overviewQuery.isLoading} />
        </Card>

        <ArtifactDetail
          artifact={selectedArtifact}
          copiedRef={copiedRef}
          onCreateReleasePlan={onCreateReleasePlan}
          onCopy={(value) => {
            void navigator.clipboard?.writeText(value);
            setCopiedRef(value);
          }}
        />
      </div>
    </>
  );
}

function ArtifactTable({
  artifacts,
  selectedId,
  onSelect,
  loading,
}: {
  artifacts: ArtifactCenterItem[];
  selectedId: string;
  onSelect: (id: string) => void;
  loading: boolean;
}) {
  return (
    <Table>
      <thead>
        <tr>
          <Th className="w-[20%]">制品</Th>
          <Th className="w-[12%]">来源</Th>
          <Th className="w-[12%]">项目</Th>
          <Th className="w-[12%]">版本</Th>
          <Th className="w-[11%]">状态</Th>
          <Th className="w-[10%]">大小</Th>
          <Th className="w-[13%]">SHA</Th>
          <Th className="w-[10%] text-right">时间</Th>
        </tr>
      </thead>
      <tbody>
        {artifacts.map((artifact) => (
          <tr
            key={`${artifact.source}:${artifact.id}`}
            className={cn("cursor-pointer transition hover:bg-muted/50", selectedId === artifact.id && "bg-muted/60")}
            onClick={() => onSelect(artifact.id)}
          >
            <Td>
              <div className="truncate font-medium">{artifact.name || artifact.file_name || shortId(artifact.id)}</div>
              <div className="truncate text-[11px] text-muted-foreground">{artifact.file_name || artifact.artifact_type || "-"}</div>
            </Td>
            <Td>
              <SourceBadge source={artifact.source} />
            </Td>
            <Td>
              <div className="truncate">{artifact.project_key || artifact.app_key || "-"}</div>
              <div className="truncate text-[11px] text-muted-foreground">{artifact.project_name || artifact.app_name || "-"}</div>
            </Td>
            <Td>
              <div className="truncate">{artifact.version_name || "-"}</div>
              <div className="text-[11px] text-muted-foreground">#{artifact.build_number || "-"}</div>
            </Td>
            <Td>
              <StatusBadge status={artifact.upload_status || artifact.status || "-"} />
            </Td>
            <Td>{formatBytes(artifact.size_bytes)}</Td>
            <Td>
              <div className="truncate font-mono text-[11px]">{artifact.sha256 ? artifact.sha256.slice(0, 12) : "-"}</div>
            </Td>
            <Td className="text-right">{formatDateTime(artifact.created_at).slice(5)}</Td>
          </tr>
        ))}
        {!artifacts.length ? (
          <tr>
            <Td colSpan={8}>{loading ? "正在读取制品" : "暂无制品"}</Td>
          </tr>
        ) : null}
      </tbody>
    </Table>
  );
}

function ArtifactDetail({
  artifact,
  copiedRef,
  onCopy,
  onCreateReleasePlan,
}: {
  artifact?: ArtifactCenterItem;
  copiedRef: string;
  onCopy: (value: string) => void;
  onCreateReleasePlan?: (seed: ReleasePlanDraftSeed) => void;
}) {
  if (!artifact) {
    return (
      <Card>
        <div className="flex min-h-[240px] items-center justify-center rounded-lg border border-dashed text-xs text-muted-foreground">暂无制品</div>
      </Card>
    );
  }
  const immutableRef = artifact.immutable_ref || `${artifact.source}:${artifact.id}`;

  return (
    <Card>
      <div className="mb-3 flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="truncate font-medium">{artifact.name || artifact.file_name}</div>
          <div className="mt-1 flex flex-wrap gap-1">
            <SourceBadge source={artifact.source} />
            <Badge>{artifact.artifact_type || "artifact"}</Badge>
            <StatusBadge status={artifact.upload_status || artifact.status || "-"} />
          </div>
        </div>
        <div className="flex shrink-0 gap-2">
          {onCreateReleasePlan ? (
            <Button variant="secondary" size="icon" onClick={() => onCreateReleasePlan(releasePlanSeedFromArtifact(artifact))} title="创建发布计划">
              <Rocket className="h-4 w-4" />
            </Button>
          ) : null}
          {isOpenableLocation(artifact.location) ? (
            <Button variant="secondary" size="icon" onClick={() => window.open(artifact.location, "_blank", "noopener,noreferrer")} title="打开制品">
              <Download className="h-4 w-4" />
            </Button>
          ) : null}
        </div>
      </div>

      <div className="grid gap-2 text-xs">
        <Info label="文件" value={artifact.file_name || "-"} />
        <Info label="项目" value={artifact.project_key || artifact.app_key || "-"} />
        <Info label="版本" value={[artifact.version_name, artifact.build_number ? `#${artifact.build_number}` : ""].filter(Boolean).join(" / ") || "-"} />
        <Info label="构建" value={shortId(artifact.build_id || artifact.run_id || "") || "-"} mono />
        <Info label="Commit" value={artifact.git_commit ? artifact.git_commit.slice(0, 12) : "-"} mono />
        <Info label="大小" value={formatBytes(artifact.size_bytes)} />
        <Info label="时间" value={formatDateTime(artifact.created_at)} />
      </div>

      <div className="mt-4 rounded-lg border bg-muted p-3">
        <div className="mb-2 flex items-center justify-between gap-2">
          <div className="flex items-center gap-1.5 text-xs font-medium">
            <Link2 className="h-3.5 w-3.5" />
            制品引用
          </div>
          <Button variant="secondary" size="sm" onClick={() => onCopy(immutableRef)}>
            <Copy className="mr-1.5 h-3.5 w-3.5" />
            {copiedRef === immutableRef ? "已复制" : "复制"}
          </Button>
        </div>
        <div className="break-all rounded-md bg-white p-2 font-mono text-[11px] text-muted-foreground">{immutableRef}</div>
      </div>

      <div className="mt-3 grid gap-2 text-xs">
        <Info label="SHA256" value={artifact.sha256 || "-"} mono />
        <Info label="位置" value={artifact.location || "-"} mono />
      </div>
    </Card>
  );
}


function releasePlanSeedFromArtifact(artifact: ArtifactCenterItem): ReleasePlanDraftSeed {
  const unitType = releaseUnitTypeFromArtifact(artifact);
  const projectKey = artifact.project_key || artifact.app_key || "release-center";
  const artifactRef = artifact.immutable_ref || `${artifact.source}:${artifact.id}`;
  const artifactName = artifact.name || artifact.file_name || artifact.artifact_type || "artifact";
  return {
    seedKey: `${artifact.source}:${artifact.id}:${Date.now()}`,
    projectKey,
    unitKey: defaultUnitKeyFromArtifact(artifact, unitType),
    environmentKey: artifact.channel === "stable" ? "prod" : "dev",
    channel: artifact.channel || "dev",
    title: `${artifactName} 发布`,
    versionName: artifact.version_name || defaultVersionNameFromDate(),
    buildNumber: artifact.build_number ?? 1,
    gitCommit: artifact.git_commit || "",
    artifactName,
    artifactType: artifact.artifact_type || "artifact",
    artifactFileName: artifact.file_name || "",
    artifactRef,
    artifactBuildRunId: artifact.run_id || "",
    artifactAppBuildId: artifact.build_id || "",
    artifactAppBuildArtifactId: artifact.app_build_artifact_id || "",
    unitTypeFilter: unitType,
    message: `已从制品中心带入 ${artifactName}。`,
  };
}

function releaseUnitTypeFromArtifact(artifact: ArtifactCenterItem) {
  const artifactType = (artifact.artifact_type || "").toLowerCase();
  if (artifactType.startsWith("windows_")) return "windows";
  if (artifactType === "web_dist") return "web";
  if (artifactType === "docker_image") return "docker";
  if (["server_binary", "binary"].includes(artifactType)) return "server";
  if (["apk", "aab"].includes(artifactType)) return "android";
  return "config";
}

function defaultUnitKeyFromArtifact(artifact: ArtifactCenterItem, unitType: string) {
  if (unitType === "windows") return "windows-app";
  if (unitType === "web") return "admin-web";
  if (unitType === "docker") return "container";
  if (unitType === "server") return "server";
  if (unitType === "android") return artifact.app_key || "android-app";
  return artifact.project_key || artifact.app_key || "config";
}

function defaultVersionNameFromDate() {
  return new Date().toISOString().slice(0, 10).replace(/-/g, ".");
}

function Metric({ label, value, note, icon: Icon }: { label: string; value: string | number; note: string; icon: typeof Package }) {
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

function Select({ label, value, onChange, options }: { label: string; value: string; onChange: (value: string) => void; options: string[] }) {
  return (
    <select
      aria-label={label}
      className="h-9 w-full rounded-lg border bg-white px-3 text-xs outline-none transition focus:border-black focus:ring-2 focus:ring-black/10"
      value={value}
      onChange={(event) => onChange(event.target.value)}
    >
      {options.map((option) => (
        <option key={option} value={option}>
          {optionLabel(option)}
        </option>
      ))}
    </select>
  );
}

function SourceBadge({ source }: { source: string }) {
  return <Badge tone={source === "build_center" ? "success" : "default"}>{sourceLabel(source)}</Badge>;
}

function StatusBadge({ status }: { status: string }) {
  return <Badge tone={statusTone(status)}>{status}</Badge>;
}

function Info({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="grid grid-cols-[72px_minmax(0,1fr)] gap-2 rounded-md border px-2 py-1.5">
      <div className="text-muted-foreground">{label}</div>
      <div className={cn("truncate", mono && "font-mono text-[11px]")}>{value}</div>
    </div>
  );
}

function artifactMetrics(artifacts: ArtifactCenterItem[]) {
  const types = new Set(artifacts.map((artifact) => artifact.artifact_type).filter(Boolean));
  return {
    total: artifacts.length,
    types: types.size,
    buildCenter: artifacts.filter((artifact) => artifact.source === "build_center").length,
    appBuild: artifacts.filter((artifact) => artifact.source === "app_build").length,
    uploaded: artifacts.filter((artifact) => (artifact.upload_status || artifact.status) === "uploaded").length,
    pending: artifacts.filter((artifact) => ["pending", "queued", "running"].includes(artifact.upload_status || artifact.status || "")).length,
    withSha: artifacts.filter((artifact) => Boolean(artifact.sha256)).length,
    totalSize: artifacts.reduce((sum, artifact) => sum + (artifact.size_bytes ?? 0), 0),
  };
}

function artifactFilterOptions(artifacts: ArtifactCenterItem[]) {
  return {
    types: unique(artifacts.map((artifact) => artifact.artifact_type).filter(Boolean)),
    statuses: unique(artifacts.map((artifact) => artifact.upload_status || artifact.status || "").filter(Boolean)),
    projects: unique(artifacts.map((artifact) => artifact.project_key || artifact.app_key || "").filter(Boolean)),
  };
}

function filterArtifacts(
  artifacts: ArtifactCenterItem[],
  filters: { query: string; source: string; artifactType: string; status: string; project: string },
) {
  const needle = filters.query.trim().toLowerCase();
  return artifacts.filter((artifact) => {
    if (filters.source !== allOption && artifact.source !== filters.source) return false;
    if (filters.artifactType !== allOption && artifact.artifact_type !== filters.artifactType) return false;
    if (filters.status !== allOption && (artifact.upload_status || artifact.status || "") !== filters.status) return false;
    if (filters.project !== allOption && (artifact.project_key || artifact.app_key || "") !== filters.project) return false;
    if (!needle) return true;
    return [
      artifact.name,
      artifact.file_name,
      artifact.project_key,
      artifact.app_key,
      artifact.version_name,
      artifact.git_commit,
      artifact.sha256,
      artifact.location,
      artifact.immutable_ref,
    ]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(needle));
  });
}

function statusTone(status: string): "default" | "success" | "warning" | "danger" {
  if (["uploaded", "success", "released"].includes(status)) return "success";
  if (["failed", "error"].includes(status)) return "danger";
  if (["pending", "queued", "running"].includes(status)) return "warning";
  return "default";
}

function sourceLabel(source: string) {
  if (source === "build_center") return "构建中心";
  if (source === "app_build") return "App 构建";
  return source || "-";
}

function optionLabel(option: string) {
  if (option === "build_center" || option === "app_build") return sourceLabel(option);
  return option;
}

function unique(values: string[]) {
  return Array.from(new Set(values)).sort((a, b) => a.localeCompare(b));
}

function shortId(value: string) {
  return value ? value.slice(0, 8) : "";
}

function formatBytes(value?: number) {
  if (!value || value <= 0) return "-";
  const units = ["B", "KB", "MB", "GB"];
  let size = value;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }
  return `${size >= 10 || unitIndex === 0 ? size.toFixed(0) : size.toFixed(1)} ${units[unitIndex]}`;
}

function isOpenableLocation(value?: string) {
  return Boolean(value && (/^https?:\/\//.test(value) || value.startsWith("/")));
}
