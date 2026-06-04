import { useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { CheckCircle2, Cloud, History, Play, RefreshCw, Server, TerminalSquare, XCircle } from "lucide-react";
import { getArtifactCenterOverview, type ArtifactCenterItem } from "@/api/artifacts";
import { getBuildCenterOverview } from "@/api/buildCenter";
import {
  completeDeployment,
  createDeployment,
  createDeploymentTarget,
  failDeployment,
  getDeployments,
  getDeploymentTargets,
  type DeploymentProvider,
  type DeploymentRecord,
  type DeploymentStatus,
  type DeploymentTarget,
} from "@/api/deployments";
import { PageHeader } from "@/components/layout/PageHeader";
import { ApiErrorState } from "@/components/stable/StableAdminComponents";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Switch } from "@/components/ui/Switch";
import { Table, Td, Th } from "@/components/ui/Table";
import { formatDateTime } from "@/lib/format";

const providerOptions: Array<{ value: DeploymentProvider; label: string }> = [
  { value: "cloudflare_pages", label: "Cloudflare Pages" },
  { value: "cloudflare_worker", label: "Cloudflare Worker" },
  { value: "cloudflare_r2", label: "Cloudflare R2" },
  { value: "generic_webhook", label: "Generic Webhook" },
  { value: "ssh", label: "SSH" },
  { value: "docker", label: "Docker" },
  { value: "kubernetes", label: "Kubernetes" },
];

const environmentOptions = ["dev", "test", "staging", "prod"];

export function DeploymentCenterPage() {
  const [targetProjectKey, setTargetProjectKey] = useState("release-center");
  const [targetKey, setTargetKey] = useState("admin-pages");
  const [targetName, setTargetName] = useState("Release Center Admin Pages");
  const [provider, setProvider] = useState<DeploymentProvider>("cloudflare_pages");
  const [environment, setEnvironment] = useState("prod");
  const [endpointURL, setEndpointURL] = useState("");
  const [cloudflareAccountID, setCloudflareAccountID] = useState("");
  const [cloudflareProjectName, setCloudflareProjectName] = useState("release-center-admin");
  const [cloudflareScriptName, setCloudflareScriptName] = useState("");
  const [cloudflareBucketName, setCloudflareBucketName] = useState("");
  const [credentialRef, setCredentialRef] = useState("cf_token_release_prod");
  const [targetEnabled, setTargetEnabled] = useState(false);
  const [selectedTargetId, setSelectedTargetId] = useState("");
  const [deployVersion, setDeployVersion] = useState(defaultDeployVersion());
  const [deployBuildNumber, setDeployBuildNumber] = useState("1");
  const [deployGitCommit, setDeployGitCommit] = useState("");
  const [artifactPath, setArtifactPath] = useState("dist");
  const [objectKey, setObjectKey] = useState("");
  const [artifactRef, setArtifactRef] = useState("");
  const [selectedArtifactId, setSelectedArtifactId] = useState("");
  const [artifactRunId, setArtifactRunId] = useState("");
  const [artifactAppBuildId, setArtifactAppBuildId] = useState("");
  const [artifactAppBuildArtifactId, setArtifactAppBuildArtifactId] = useState("");
  const [deploymentURL, setDeploymentURL] = useState("");
  const [deploymentDryRun, setDeploymentDryRun] = useState(true);
  const [filter, setFilter] = useState("全部");
  const [message, setMessage] = useState("部署中心已就绪。");
  const [messageTone, setMessageTone] = useState<"default" | "success" | "warning" | "danger">("default");

  const projectsQuery = useQuery({
    queryKey: ["deployment-center-projects"],
    queryFn: getBuildCenterOverview,
  });
  const targetsQuery = useQuery({
    queryKey: ["deployment-targets"],
    queryFn: getDeploymentTargets,
  });
  const recordsQuery = useQuery({
    queryKey: ["deployment-records"],
    queryFn: getDeployments,
    refetchInterval: (query) => (hasActiveDeployments(query.state.data?.deployment_records ?? []) ? 5_000 : false),
  });
  const artifactsQuery = useQuery({
    queryKey: ["deployment-center-artifacts"],
    queryFn: getArtifactCenterOverview,
  });

  const projects = projectsQuery.data?.projects ?? [];
  const targets = targetsQuery.data?.deployment_targets ?? [];
  const records = recordsQuery.data?.deployment_records ?? [];
  const artifacts = artifactsQuery.data?.artifacts ?? [];
  const projectKeys = useMemo(() => unique(["release-center", ...projects.map((project) => project.project_key)]), [projects]);
  const selectedTarget = useMemo(
    () => targets.find((target) => target.id === selectedTargetId) ?? targets[0],
    [selectedTargetId, targets],
  );
  const filteredRecords = useMemo(
    () => records.filter((record) => filter === "全部" || record.provider_status === filter || record.environment === filter || record.provider === filter),
    [filter, records],
  );
  const metrics = useMemo(() => deploymentMetrics(targets, records), [records, targets]);
  const targetValidation = validateTargetForm({ projectKey: targetProjectKey, targetKey, provider, name: targetName });
  const deploymentValidation = validateDeploymentForm({ target: selectedTarget, version: deployVersion, buildNumber: deployBuildNumber });

  const createTargetMutation = useMutation({
    mutationFn: () => {
      if (targetValidation) throw new Error(targetValidation);
      return createDeploymentTarget({
        project_key: targetProjectKey.trim(),
        target_key: targetKey.trim(),
        name: targetName.trim(),
        provider,
        environment,
        endpoint_url: endpointURL.trim(),
        cloudflare_account_id: cloudflareAccountID.trim(),
        cloudflare_project_name: cloudflareProjectName.trim(),
        cloudflare_script_name: cloudflareScriptName.trim(),
        cloudflare_bucket_name: cloudflareBucketName.trim(),
        credential_ref: credentialRef.trim(),
        enabled: targetEnabled,
        metadata: { source: "admin_web", ui: "deployment_center" },
      });
    },
    onSuccess: async (result) => {
      setSelectedTargetId(result.target.id);
      setMessage(result.message_zh || `部署目标已保存：${result.target.name}`);
      setMessageTone("success");
      await Promise.all([targetsQuery.refetch(), projectsQuery.refetch()]);
    },
    onError: (error) => showDeploymentError(error, "保存部署目标失败", setMessage, setMessageTone),
  });

  const createDeploymentMutation = useMutation({
    mutationFn: () => {
      if (deploymentValidation || !selectedTarget) throw new Error(deploymentValidation || "请选择部署目标");
      return createDeployment({
        target_id: selectedTarget.id,
        run_id: artifactRunId.trim(),
        app_build_id: artifactAppBuildId.trim(),
        app_build_artifact_id: artifactAppBuildArtifactId.trim(),
        version_name: deployVersion.trim(),
        build_number: Number(deployBuildNumber),
        git_commit: deployGitCommit.trim(),
        deployment_url: deploymentURL.trim(),
        triggered_by: "admin-web",
        dry_run: deploymentDryRun,
        metadata: {
          source: "admin_web",
          artifact_path: artifactPath.trim(),
          object_key: objectKey.trim(),
          immutable_ref: artifactRef.trim(),
          artifact_center_id: selectedArtifactId || undefined,
          target_key: selectedTarget.target_key,
        },
      });
    },
    onSuccess: async (result) => {
      setMessage(result.message_zh || `部署记录已创建：${shortId(result.record.id)}`);
      setMessageTone("success");
      await recordsQuery.refetch();
    },
    onError: (error) => showDeploymentError(error, "创建部署记录失败", setMessage, setMessageTone),
  });

  const completeMutation = useMutation({
    mutationFn: (record: DeploymentRecord) =>
      completeDeployment(record.id, {
        provider_status: "success",
        deployment_url: record.deployment_url,
        log_tail: ["deployment marked success from admin web"],
        metadata: { source: "admin_web" },
      }),
    onSuccess: async (result) => {
      setMessage(result.message_zh || "部署已完成");
      setMessageTone("success");
      await recordsQuery.refetch();
    },
    onError: (error) => showDeploymentError(error, "完成部署失败", setMessage, setMessageTone),
  });

  const failMutation = useMutation({
    mutationFn: (record: DeploymentRecord) =>
      failDeployment(record.id, {
        provider_status: "failed",
        error_message: "marked failed from admin web",
        log_tail: ["deployment marked failed from admin web"],
        metadata: { source: "admin_web" },
      }),
    onSuccess: async (result) => {
      setMessage(result.message_zh || "部署失败已记录");
      setMessageTone("success");
      await recordsQuery.refetch();
    },
    onError: (error) => showDeploymentError(error, "记录部署失败失败", setMessage, setMessageTone),
  });

  const mutationError = createTargetMutation.error || createDeploymentMutation.error || completeMutation.error || failMutation.error;
  const busy = createTargetMutation.isPending || createDeploymentMutation.isPending || completeMutation.isPending || failMutation.isPending;

  function applyArtifact(artifact?: ArtifactCenterItem) {
    if (!artifact) {
      setSelectedArtifactId("");
      setArtifactRunId("");
      setArtifactAppBuildId("");
      setArtifactAppBuildArtifactId("");
      setArtifactRef("");
      return;
    }
    setSelectedArtifactId(artifact.id);
    setArtifactRunId(artifact.run_id || "");
    setArtifactAppBuildId(artifact.build_id || "");
    setArtifactAppBuildArtifactId(artifact.app_build_artifact_id || "");
    setArtifactRef(artifact.immutable_ref || `${artifact.source}:${artifact.id}`);
    setArtifactPath(firstNonEmpty(artifact.location, artifact.file_name, artifact.name, artifactPath));
    setObjectKey(firstNonEmpty(artifact.file_name, artifact.name, objectKey));
    if (artifact.version_name) setDeployVersion(artifact.version_name);
    if (artifact.build_number) setDeployBuildNumber(String(artifact.build_number));
    if (artifact.git_commit) setDeployGitCommit(artifact.git_commit);
  }

  return (
    <>
      <PageHeader title="部署中心">
        <Button
          variant="secondary"
          onClick={() => {
            projectsQuery.refetch();
            targetsQuery.refetch();
            recordsQuery.refetch();
            artifactsQuery.refetch();
          }}
          disabled={projectsQuery.isFetching || targetsQuery.isFetching || recordsQuery.isFetching || artifactsQuery.isFetching}
        >
          <RefreshCw className="mr-2 h-4 w-4" />
          刷新
        </Button>
      </PageHeader>

      {projectsQuery.isError ? <ApiErrorState error={projectsQuery.error} title="项目列表读取失败" /> : null}
      {targetsQuery.isError ? <ApiErrorState error={targetsQuery.error} title="部署目标读取失败" /> : null}
      {recordsQuery.isError ? <ApiErrorState error={recordsQuery.error} title="部署记录读取失败" /> : null}
      {artifactsQuery.isError ? <ApiErrorState error={artifactsQuery.error} title="制品中心读取失败" /> : null}
      {mutationError ? <ApiErrorState error={mutationError} title="部署操作失败" /> : null}

      <div className="mb-4">
        <StatusMessage text={message} tone={messageTone} />
      </div>

      <div className="mb-4 grid gap-3 md:grid-cols-4">
        <Metric label="部署目标" value={metrics.targets} note={`${metrics.enabledTargets} 个启用`} icon={Server} />
        <Metric label="Cloudflare" value={metrics.cloudflareTargets} note="Pages / Worker / R2" icon={Cloud} />
        <Metric label="部署记录" value={metrics.records} note={`${metrics.dryRuns} 个 dry-run`} icon={History} />
        <Metric label="成功部署" value={metrics.successRecords} note={`${metrics.failedRecords} 个失败`} icon={CheckCircle2} />
      </div>

      <div className="grid gap-4 xl:grid-cols-[420px_minmax(0,1fr)]">
        <div className="grid gap-4">
          <Card>
            <SectionTitle title="部署目标" badge={`${targets.length} 个`} />
            <div className="grid gap-2">
              <div className="grid grid-cols-2 gap-2">
                <Select label="项目" value={targetProjectKey} onChange={setTargetProjectKey} options={projectKeys} />
                <Select label="环境" value={environment} onChange={setEnvironment} options={environmentOptions} />
              </div>
              <div className="grid grid-cols-2 gap-2">
                <Input placeholder="target key" value={targetKey} onChange={(event) => setTargetKey(event.target.value)} />
                <Input placeholder="显示名称" value={targetName} onChange={(event) => setTargetName(event.target.value)} />
              </div>
              <Select label="Provider" value={provider} onChange={(value) => setProvider(value)} options={providerOptions} />
              <Input placeholder="endpoint url" value={endpointURL} onChange={(event) => setEndpointURL(event.target.value)} />
              <div className="rounded-lg border bg-muted p-3">
                <div className="mb-2 text-xs font-medium">Cloudflare</div>
                <div className="grid gap-2">
                  <Input placeholder="account id" value={cloudflareAccountID} onChange={(event) => setCloudflareAccountID(event.target.value)} />
                  <Input placeholder="pages project name" value={cloudflareProjectName} onChange={(event) => setCloudflareProjectName(event.target.value)} />
                  <Input placeholder="worker script name" value={cloudflareScriptName} onChange={(event) => setCloudflareScriptName(event.target.value)} />
                  <Input placeholder="r2 bucket name" value={cloudflareBucketName} onChange={(event) => setCloudflareBucketName(event.target.value)} />
                </div>
              </div>
              <Input placeholder="credential ref" value={credentialRef} onChange={(event) => setCredentialRef(event.target.value)} />
              <div className="flex items-center justify-between gap-3 rounded-lg border p-3">
                <div className="text-xs font-medium">启用目标</div>
                <Switch checked={targetEnabled} onCheckedChange={setTargetEnabled} aria-label="启用部署目标" />
              </div>
              {targetValidation ? <InlineWarning text={targetValidation} /> : null}
              <Button onClick={() => createTargetMutation.mutate()} disabled={Boolean(targetValidation) || createTargetMutation.isPending}>
                {createTargetMutation.isPending ? "保存中" : "保存部署目标"}
              </Button>
            </div>
          </Card>

          <Card>
            <SectionTitle title="部署投递" badge={deploymentDryRun ? "dry-run" : "worker"} />
            <div className="grid gap-2">
              <Select
                label="目标"
                value={selectedTarget?.id ?? ""}
                onChange={setSelectedTargetId}
                options={targets.map((target) => ({
                  value: target.id,
                  label: `${target.name} / ${providerLabel(target.provider)} / ${target.environment}`,
                }))}
                placeholder="暂无部署目标"
              />
              {selectedTarget ? <TargetPreview target={selectedTarget} /> : <EmptyBox text="暂无部署目标" />}
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
              <div className="grid grid-cols-[minmax(0,1fr)_100px] gap-2">
                <Input placeholder="version" value={deployVersion} onChange={(event) => setDeployVersion(event.target.value)} />
                <Input placeholder="build" type="number" value={deployBuildNumber} onChange={(event) => setDeployBuildNumber(event.target.value)} />
              </div>
              <Input placeholder="git commit / tag" value={deployGitCommit} onChange={(event) => setDeployGitCommit(event.target.value)} />
              <Input placeholder="artifact path" value={artifactPath} onChange={(event) => setArtifactPath(event.target.value)} />
              <Input placeholder="object key / R2 key" value={objectKey} onChange={(event) => setObjectKey(event.target.value)} />
              <Input placeholder="immutable ref" value={artifactRef} onChange={(event) => setArtifactRef(event.target.value)} />
              {selectedArtifactId ? (
                <div className="grid gap-1 rounded-md border bg-muted p-2 font-mono text-[11px] text-muted-foreground">
                  <span>run={artifactRunId || "-"}</span>
                  <span>build={artifactAppBuildId || "-"}</span>
                  <span>artifact={artifactAppBuildArtifactId || "-"}</span>
                </div>
              ) : null}
              <Input placeholder="deployment url" value={deploymentURL} onChange={(event) => setDeploymentURL(event.target.value)} />
              <div className="flex items-center justify-between gap-3 rounded-lg border p-3">
                <div className="text-xs font-medium">Dry-run</div>
                <Switch checked={deploymentDryRun} onCheckedChange={setDeploymentDryRun} aria-label="Dry-run 部署" />
              </div>
              {deploymentValidation ? <InlineWarning text={deploymentValidation} /> : null}
              <Button onClick={() => createDeploymentMutation.mutate()} disabled={Boolean(deploymentValidation) || createDeploymentMutation.isPending}>
                <Play className="mr-2 h-4 w-4" />
                {createDeploymentMutation.isPending ? "创建中" : deploymentDryRun ? "创建 dry-run 部署" : "投递 Worker 部署"}
              </Button>
            </div>
          </Card>
        </div>

        <div className="grid gap-4">
          <Card>
            <SectionTitle title="目标列表" badge={`${targets.length} 个`} />
            <div className="grid gap-3 md:grid-cols-2">
              {targets.map((target) => (
                <TargetCard
                  key={target.id}
                  target={target}
                  active={selectedTarget?.id === target.id}
                  onSelect={() => setSelectedTargetId(target.id)}
                />
              ))}
              {!targets.length ? <EmptyBox text={targetsQuery.isLoading ? "正在读取部署目标" : "暂无部署目标"} /> : null}
            </div>
          </Card>

          <Card>
            <div className="mb-4 grid gap-3 md:grid-cols-[180px_minmax(0,1fr)]">
              <Select
                label="筛选"
                value={filter}
                onChange={setFilter}
                options={["全部", "prod", "staging", "dev", "dry_run", "running", "success", "failed", "cloudflare_pages", "cloudflare_worker", "cloudflare_r2"]}
              />
              <div className="flex items-center justify-end">
                <Badge>{filteredRecords.length} / {records.length} 条</Badge>
              </div>
            </div>
            <DeploymentRecordsTable
              records={filteredRecords}
              busy={busy}
              onComplete={(record) => completeMutation.mutate(record)}
              onFail={(record) => failMutation.mutate(record)}
            />
          </Card>
        </div>
      </div>
    </>
  );
}

function TargetPreview({ target }: { target: DeploymentTarget }) {
  const command = preparedCommandFromTarget(target);
  return (
    <div className="rounded-lg border bg-muted p-3 text-xs">
      <Info label="Provider" value={providerLabel(target.provider)} />
      <Info label="目标" value={`${target.target_key} / ${target.environment}`} />
      <Info label="凭证引用" value={target.credential_ref || "-"} />
      {command ? <Info label="命令类型" value={command} /> : null}
    </div>
  );
}

function TargetCard({ target, active, onSelect }: { target: DeploymentTarget; active: boolean; onSelect: () => void }) {
  return (
    <button className={`rounded-lg border p-3 text-left transition ${active ? "border-black bg-muted/50" : "hover:border-black"}`} onClick={onSelect}>
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="truncate text-sm font-medium">{target.name}</div>
          <div className="mt-1 truncate font-mono text-[11px] text-muted-foreground">{target.target_key}</div>
        </div>
        <Badge tone={target.enabled ? "success" : "warning"}>{target.enabled ? "启用" : "停用"}</Badge>
      </div>
      <div className="mt-3 grid gap-1 text-xs">
        <Info label="Provider" value={providerLabel(target.provider)} />
        <Info label="环境" value={target.environment || "-"} />
        <Info label="Pages" value={target.cloudflare_project_name || "-"} />
        <Info label="Worker" value={target.cloudflare_script_name || "-"} />
        <Info label="R2" value={target.cloudflare_bucket_name || "-"} />
      </div>
    </button>
  );
}

function DeploymentRecordsTable({
  records,
  busy,
  onComplete,
  onFail,
}: {
  records: DeploymentRecord[];
  busy: boolean;
  onComplete: (record: DeploymentRecord) => void;
  onFail: (record: DeploymentRecord) => void;
}) {
  return (
    <Table>
      <thead>
        <tr>
          <Th>部署</Th>
          <Th>状态</Th>
          <Th>目标</Th>
          <Th>版本</Th>
          <Th>Cloudflare 命令</Th>
          <Th>时间</Th>
          <Th>操作</Th>
        </tr>
      </thead>
      <tbody>
        {records.map((record) => {
          const command = preparedCommand(record);
          const active = ["queued", "running", "dry_run", "external"].includes(record.provider_status);
          return (
            <tr key={record.id}>
              <Td>
                <div className="truncate font-mono text-[11px]">{shortId(record.id)}</div>
                <div className="truncate text-[11px] text-muted-foreground">{providerLabel(record.provider)} / {record.environment || "-"}</div>
              </Td>
              <Td>
                <DeploymentStatusBadge status={record.provider_status} />
              </Td>
              <Td>
                <div className="truncate font-medium">{record.target_name || record.target_key || "-"}</div>
                <div className="truncate text-[11px] text-muted-foreground">{record.deployment_url || record.external_deployment_id || "-"}</div>
              </Td>
              <Td>
                <div className="truncate">{record.version_name || "-"}</div>
                <div className="truncate font-mono text-[11px] text-muted-foreground">{record.git_commit || `build ${record.build_number ?? "-"}`}</div>
              </Td>
              <Td>
                {command.length ? (
                  <div className="max-w-[320px] truncate rounded-md bg-zinc-950 px-2 py-1 font-mono text-[11px] text-zinc-100">
                    {command.join(" ")}
                  </div>
                ) : (
                  <span className="text-muted-foreground">-</span>
                )}
              </Td>
              <Td>{formatDeploymentTime(record)}</Td>
              <Td>
                <div className="flex flex-wrap gap-2">
                  <Button variant="secondary" size="sm" disabled={busy || !active} onClick={() => onComplete(record)}>
                    完成
                  </Button>
                  <Button variant="secondary" size="sm" disabled={busy || record.provider_status === "failed"} onClick={() => onFail(record)}>
                    失败
                  </Button>
                </div>
              </Td>
            </tr>
          );
        })}
        {!records.length ? (
          <tr>
            <Td colSpan={7}>暂无部署记录</Td>
          </tr>
        ) : null}
      </tbody>
    </Table>
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

function Metric({ label, value, note, icon: Icon }: { label: string; value: string | number; note: string; icon: typeof Server }) {
  return (
    <Card>
      <div className="flex items-center justify-between gap-3">
        <span className="text-xs text-muted-foreground">{label}</span>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </div>
      <div className="mt-3 truncate text-2xl font-semibold">{value}</div>
      <div className="mt-1 truncate text-xs text-muted-foreground">{note}</div>
    </Card>
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

function InlineWarning({ text }: { text: string }) {
  return <div className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs leading-5 text-amber-800">{text}</div>;
}

function EmptyBox({ text }: { text: string }) {
  return <div className="rounded-lg border bg-muted p-4 text-sm text-muted-foreground">{text}</div>;
}

function DeploymentStatusBadge({ status }: { status: DeploymentStatus }) {
  const tone = deploymentStatusTone(status);
  const Icon = status === "success" ? CheckCircle2 : status === "failed" || status === "canceled" ? XCircle : status === "dry_run" ? TerminalSquare : Cloud;
  return (
    <Badge tone={tone}>
      <Icon className="mr-1 h-3 w-3" />
      {deploymentStatusLabel(status)}
    </Badge>
  );
}

function validateTargetForm({
  projectKey,
  targetKey,
  provider,
  name,
}: {
  projectKey: string;
  targetKey: string;
  provider: string;
  name: string;
}) {
  if (!projectKey.trim() || !targetKey.trim() || !provider.trim()) return "项目、target key 和 provider 不能为空";
  if (!name.trim()) return "显示名称不能为空";
  return "";
}

function validateDeploymentForm({ target, version, buildNumber }: { target?: DeploymentTarget; version: string; buildNumber: string }) {
  if (!target) return "请选择部署目标";
  if (!version.trim()) return "版本不能为空";
  const build = Number(buildNumber);
  if (!Number.isInteger(build) || build < 0) return "build 必须是非负整数";
  return "";
}

function deploymentMetrics(targets: DeploymentTarget[], records: DeploymentRecord[]) {
  return {
    targets: targets.length,
    enabledTargets: targets.filter((target) => target.enabled).length,
    cloudflareTargets: targets.filter((target) => target.provider.startsWith("cloudflare_")).length,
    records: records.length,
    dryRuns: records.filter((record) => record.provider_status === "dry_run").length,
    successRecords: records.filter((record) => record.provider_status === "success").length,
    failedRecords: records.filter((record) => record.provider_status === "failed").length,
  };
}

function hasActiveDeployments(records: DeploymentRecord[]) {
  return records.some((record) => record.provider_status === "queued" || record.provider_status === "running");
}

function preparedCommand(record: DeploymentRecord) {
  const command = metadataRecord(record.metadata).prepared_command;
  return Array.isArray(command) ? command.map(String) : [];
}

function metadataRecord(value: unknown) {
  return value && typeof value === "object" && !Array.isArray(value) ? (value as Record<string, unknown>) : {};
}

function preparedCommandFromTarget(target: DeploymentTarget) {
  if (target.provider === "cloudflare_pages") return "wrangler pages deploy";
  if (target.provider === "cloudflare_worker") return "wrangler deploy";
  if (target.provider === "cloudflare_r2") return "wrangler r2 object put";
  return "";
}

function providerLabel(provider?: string) {
  return providerOptions.find((item) => item.value === provider)?.label ?? provider ?? "-";
}

function artifactOptionLabel(artifact: ArtifactCenterItem) {
  const scope = artifact.project_key || artifact.app_key || artifact.source;
  const version = artifact.version_name ? `${artifact.version_name}${artifact.build_number ? ` #${artifact.build_number}` : ""}` : artifact.artifact_type;
  return `${scope} / ${artifact.name || artifact.file_name || artifact.id.slice(0, 8)} / ${version}`;
}

function firstNonEmpty(...values: Array<string | undefined>) {
  return values.find((value) => value && value.trim())?.trim() ?? "";
}

function deploymentStatusTone(status?: DeploymentStatus): "default" | "success" | "warning" | "danger" {
  if (status === "success") return "success";
  if (status === "failed" || status === "canceled") return "danger";
  if (status === "queued" || status === "running" || status === "dry_run" || status === "external") return "warning";
  return "default";
}

function deploymentStatusLabel(status?: DeploymentStatus) {
  return (
    {
      queued: "排队",
      running: "运行中",
      success: "成功",
      failed: "失败",
      canceled: "取消",
      dry_run: "Dry-run",
      external: "外部",
    }[status ?? ""] ?? status ?? "-"
  );
}

function formatDeploymentTime(record: DeploymentRecord) {
  return formatCleanDate(record.finished_at) || formatCleanDate(record.started_at) || formatCleanDate(record.created_at);
}

function formatCleanDate(value?: string) {
  if (!value || value.startsWith("0001-")) return "-";
  return formatDateTime(value);
}

function shortId(id?: string) {
  return id ? id.slice(0, 8) : "-";
}

function unique(values: string[]) {
  return Array.from(new Set(values.filter(Boolean)));
}

function defaultDeployVersion() {
  const now = new Date();
  const yyyy = now.getFullYear();
  const mm = String(now.getMonth() + 1).padStart(2, "0");
  const dd = String(now.getDate()).padStart(2, "0");
  return `${yyyy}.${mm}.${dd}`;
}

function showDeploymentError(
  error: unknown,
  fallback: string,
  setMessage: (value: string) => void,
  setTone: (value: "default" | "success" | "warning" | "danger") => void,
) {
  setMessage(error instanceof Error ? error.message : fallback);
  setTone("danger");
}
