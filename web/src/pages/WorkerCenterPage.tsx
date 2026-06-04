import { useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { CheckCircle2, Clock3, Cpu, Play, RefreshCw, Server, TerminalSquare, XCircle } from "lucide-react";
import { getBuildCenterOverview } from "@/api/buildCenter";
import { createWorkerTask, getWorkerOverview, type BuildWorker, type WorkerTask, type WorkerTaskStatus } from "@/api/workers";
import { PageHeader } from "@/components/layout/PageHeader";
import { ApiErrorState } from "@/components/stable/StableAdminComponents";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Table, Td, Th } from "@/components/ui/Table";
import { formatDateTime } from "@/lib/format";

const taskTypeOptions = ["build", "deploy", "verify", "artifact"];
const actionOptions = ["all", "build", "upload", "verify", "deploy", "cloudflare_pages", "cloudflare_worker", "cloudflare_r2"];
const labelPresets = ["linux", "windows", "arm64", "amd64", "android", "node", "go", "docker", "cloudflare"];

export function WorkerCenterPage() {
  const [projectKey, setProjectKey] = useState("release-center");
  const [taskType, setTaskType] = useState("build");
  const [action, setAction] = useState("all");
  const [requiredLabels, setRequiredLabels] = useState("linux,node");
  const [priority, setPriority] = useState("10");
  const [metadataCommand, setMetadataCommand] = useState("buildctl all release-center");
  const [filter, setFilter] = useState("全部");
  const [message, setMessage] = useState("Worker 接入已就绪。");
  const [messageTone, setMessageTone] = useState<"default" | "success" | "warning" | "danger">("default");

  const overviewQuery = useQuery({
    queryKey: ["worker-overview"],
    queryFn: getWorkerOverview,
    refetchInterval: (query) => (hasActiveWorkerTasks(query.state.data?.tasks ?? []) ? 5_000 : false),
  });
  const projectsQuery = useQuery({
    queryKey: ["worker-center-projects"],
    queryFn: getBuildCenterOverview,
  });

  const workers = overviewQuery.data?.workers ?? [];
  const tasks = overviewQuery.data?.tasks ?? [];
  const projectKeys = useMemo(() => unique(["release-center", ...(projectsQuery.data?.projects ?? []).map((project) => project.project_key)]), [
    projectsQuery.data?.projects,
  ]);
  const filteredTasks = useMemo(
    () => tasks.filter((task) => filter === "全部" || task.status === filter || task.action === filter || task.required_labels.includes(filter)),
    [filter, tasks],
  );
  const metrics = useMemo(() => workerMetrics(workers, tasks), [tasks, workers]);
  const createValidation = validateCreateTaskForm({ taskType, action, priority });

  const createTaskMutation = useMutation({
    mutationFn: () => {
      if (createValidation) throw new Error(createValidation);
      return createWorkerTask({
        project_key: projectKey.trim(),
        task_type: taskType,
        action,
        required_labels: splitLabels(requiredLabels),
        priority: Number(priority),
        metadata: {
          source: "admin_web",
          command: metadataCommand.trim(),
        },
      });
    },
    onSuccess: async (result) => {
      setMessage(result.message_zh || `Worker 任务已创建：${shortId(result.task.id)}`);
      setMessageTone("success");
      await overviewQuery.refetch();
    },
    onError: (error) => showWorkerError(error, "创建 Worker 任务失败", setMessage, setMessageTone),
  });

  return (
    <>
      <PageHeader title="Worker 接入">
        <Button
          variant="secondary"
          onClick={() => {
            overviewQuery.refetch();
            projectsQuery.refetch();
          }}
          disabled={overviewQuery.isFetching || projectsQuery.isFetching}
        >
          <RefreshCw className="mr-2 h-4 w-4" />
          刷新
        </Button>
      </PageHeader>

      {overviewQuery.isError ? <ApiErrorState error={overviewQuery.error} title="Worker 数据读取失败" /> : null}
      {projectsQuery.isError ? <ApiErrorState error={projectsQuery.error} title="项目列表读取失败" /> : null}
      {createTaskMutation.error ? <ApiErrorState error={createTaskMutation.error} title="Worker 任务创建失败" /> : null}

      <div className="mb-4">
        <StatusMessage text={message || overviewQuery.data?.message_zh || "Worker 接入已就绪。"} tone={messageTone} />
      </div>

      <div className="mb-4 grid gap-3 md:grid-cols-4">
        <Metric label="Worker" value={metrics.workers} note={`${metrics.onlineWorkers} 个在线`} icon={Server} />
        <Metric label="容量" value={metrics.capacity} note={`${metrics.runningTasks} 个运行任务`} icon={Cpu} />
        <Metric label="任务" value={metrics.tasks} note={`${metrics.queuedTasks} 个排队`} icon={Clock3} />
        <Metric label="成功" value={metrics.successTasks} note={`${metrics.failedTasks} 个失败`} icon={CheckCircle2} />
      </div>

      <div className="grid gap-4 xl:grid-cols-[420px_minmax(0,1fr)]">
        <div className="grid gap-4">
          <Card>
            <SectionTitle title="投递任务" badge="queued" />
            <div className="grid gap-2">
              <Select label="项目" value={projectKey} onChange={setProjectKey} options={projectKeys} />
              <div className="grid grid-cols-2 gap-2">
                <Select label="任务类型" value={taskType} onChange={setTaskType} options={taskTypeOptions} />
                <Select label="动作" value={action} onChange={setAction} options={actionOptions} />
              </div>
              <div className="grid grid-cols-[minmax(0,1fr)_90px] gap-2">
                <Input placeholder="required labels" value={requiredLabels} onChange={(event) => setRequiredLabels(event.target.value)} />
                <Input placeholder="priority" type="number" value={priority} onChange={(event) => setPriority(event.target.value)} />
              </div>
              <div className="flex flex-wrap gap-1">
                {labelPresets.map((label) => (
                  <button
                    key={label}
                    className="rounded-full border px-2 py-0.5 text-[11px] text-muted-foreground transition hover:border-black hover:text-foreground"
                    onClick={() => setRequiredLabels(addLabel(requiredLabels, label))}
                    type="button"
                  >
                    {label}
                  </button>
                ))}
              </div>
              <Input placeholder="metadata command" value={metadataCommand} onChange={(event) => setMetadataCommand(event.target.value)} />
              {createValidation ? <InlineWarning text={createValidation} /> : null}
              <Button onClick={() => createTaskMutation.mutate()} disabled={Boolean(createValidation) || createTaskMutation.isPending}>
                <Play className="mr-2 h-4 w-4" />
                {createTaskMutation.isPending ? "创建中" : "创建 Worker 任务"}
              </Button>
            </div>
          </Card>

          <Card>
            <SectionTitle title="Worker 列表" badge={`${workers.length} 台`} />
            <div className="grid gap-3">
              {workers.map((worker) => (
                <WorkerCard key={worker.id} worker={worker} />
              ))}
              {!workers.length ? <EmptyBox text={overviewQuery.isLoading ? "正在读取 Worker" : "暂无 Worker 注册"} /> : null}
            </div>
          </Card>
        </div>

        <Card>
          <div className="mb-4 grid gap-3 md:grid-cols-[180px_minmax(0,1fr)]">
            <Select
              label="筛选"
              value={filter}
              onChange={setFilter}
              options={["全部", "queued", "leased", "running", "success", "failed", ...labelPresets]}
            />
            <div className="flex items-center justify-end">
              <Badge>{filteredTasks.length} / {tasks.length} 条</Badge>
            </div>
          </div>
          <WorkerTasksTable tasks={filteredTasks} workers={workers} />
        </Card>
      </div>
    </>
  );
}

function WorkerCard({ worker }: { worker: BuildWorker }) {
  return (
    <div className="rounded-lg border p-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="truncate text-sm font-medium">{worker.name || worker.worker_key}</div>
          <div className="mt-1 truncate font-mono text-[11px] text-muted-foreground">{worker.worker_key}</div>
        </div>
        <Badge tone={workerStatusTone(worker.status)}>{workerStatusLabel(worker.status)}</Badge>
      </div>
      <div className="mt-3 grid gap-2 text-xs">
        <Info label="容量" value={`${worker.running_tasks}/${worker.capacity}`} />
        <Info label="最近心跳" value={formatCleanDate(worker.last_seen_at)} />
        <Info label="Endpoint" value={worker.endpoint_url || "-"} />
      </div>
      <div className="mt-3 flex flex-wrap gap-1">
        {worker.labels.map((label) => (
          <Badge key={label}>{label}</Badge>
        ))}
        {!worker.labels.length ? <Badge>no-label</Badge> : null}
      </div>
    </div>
  );
}

function WorkerTasksTable({ tasks, workers }: { tasks: WorkerTask[]; workers: BuildWorker[] }) {
  return (
    <Table>
      <thead>
        <tr>
          <Th>任务</Th>
          <Th>状态</Th>
          <Th>动作</Th>
          <Th>标签</Th>
          <Th>Worker</Th>
          <Th>尝试</Th>
          <Th>时间</Th>
          <Th>日志</Th>
        </tr>
      </thead>
      <tbody>
        {tasks.map((task) => (
          <tr key={task.id}>
            <Td>
              <div className="truncate font-mono text-[11px]">{shortId(task.id)}</div>
              <div className="truncate text-[11px] text-muted-foreground">{task.task_type}</div>
            </Td>
            <Td>
              <WorkerTaskStatusBadge status={task.status} />
            </Td>
            <Td>
              <div className="truncate font-medium">{task.action}</div>
              <div className="truncate text-[11px] text-muted-foreground">priority {task.priority}</div>
            </Td>
            <Td>
              <div className="flex max-w-[220px] flex-wrap gap-1">
                {task.required_labels.map((label) => (
                  <Badge key={label}>{label}</Badge>
                ))}
                {!task.required_labels.length ? <Badge>any</Badge> : null}
              </div>
            </Td>
            <Td>{workerName(workers, task.worker_id)}</Td>
            <Td>{task.attempts}</Td>
            <Td>{formatCleanDate(task.updated_at || task.created_at)}</Td>
            <Td>
              <div className="max-w-[240px] truncate text-xs text-muted-foreground">{latestLogLine(task) || task.error_message || "-"}</div>
            </Td>
          </tr>
        ))}
        {!tasks.length ? (
          <tr>
            <Td colSpan={8}>暂无 Worker 任务</Td>
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
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: Array<string | { value: string; label: string }>;
}) {
  return (
    <select
      aria-label={label}
      className="h-9 w-full rounded-lg border bg-white px-3 text-xs outline-none transition focus:border-black focus:ring-2 focus:ring-black/10"
      value={value}
      onChange={(event) => onChange(event.target.value)}
    >
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

function WorkerTaskStatusBadge({ status }: { status: WorkerTaskStatus }) {
  const Icon = status === "success" ? CheckCircle2 : status === "failed" || status === "canceled" ? XCircle : status === "queued" ? Clock3 : TerminalSquare;
  return (
    <Badge tone={workerTaskStatusTone(status)}>
      <Icon className="mr-1 h-3 w-3" />
      {workerTaskStatusLabel(status)}
    </Badge>
  );
}

function validateCreateTaskForm({ taskType, action, priority }: { taskType: string; action: string; priority: string }) {
  if (!taskType.trim() || !action.trim()) return "任务类型和动作不能为空";
  const value = Number(priority);
  if (!Number.isInteger(value) || value < 0) return "优先级必须是非负整数";
  return "";
}

function workerMetrics(workers: BuildWorker[], tasks: WorkerTask[]) {
  return {
    workers: workers.length,
    onlineWorkers: workers.filter((worker) => ["registered", "online", "busy"].includes(worker.status)).length,
    capacity: workers.reduce((sum, worker) => sum + (worker.capacity || 0), 0),
    runningTasks: workers.reduce((sum, worker) => sum + (worker.running_tasks || 0), 0),
    tasks: tasks.length,
    queuedTasks: tasks.filter((task) => task.status === "queued").length,
    successTasks: tasks.filter((task) => task.status === "success").length,
    failedTasks: tasks.filter((task) => task.status === "failed").length,
  };
}

function hasActiveWorkerTasks(tasks: WorkerTask[]) {
  return tasks.some((task) => ["queued", "leased", "running"].includes(task.status));
}

function splitLabels(value: string) {
  return value
    .split(/[,\s]+/)
    .map((label) => label.trim().toLowerCase())
    .filter(Boolean);
}

function addLabel(current: string, label: string) {
  const labels = new Set(splitLabels(current));
  labels.add(label);
  return Array.from(labels).join(",");
}

function workerName(workers: BuildWorker[], workerID?: string) {
  if (!workerID) return "-";
  const worker = workers.find((item) => item.id === workerID);
  return worker?.name || worker?.worker_key || shortId(workerID);
}

function latestLogLine(task: WorkerTask) {
  const lines = task.log_tail ?? [];
  return lines.length ? lines[lines.length - 1] : "";
}

function workerStatusTone(status?: string): "default" | "success" | "warning" | "danger" {
  if (status === "online" || status === "registered") return "success";
  if (status === "busy" || status === "draining") return "warning";
  if (status === "offline") return "danger";
  return "default";
}

function workerStatusLabel(status?: string) {
  return (
    {
      registered: "已注册",
      online: "在线",
      busy: "忙碌",
      draining: "排空",
      offline: "离线",
    }[status ?? ""] ?? status ?? "-"
  );
}

function workerTaskStatusTone(status?: string): "default" | "success" | "warning" | "danger" {
  if (status === "success") return "success";
  if (status === "failed" || status === "canceled") return "danger";
  if (status === "queued" || status === "leased" || status === "running") return "warning";
  return "default";
}

function workerTaskStatusLabel(status?: string) {
  return (
    {
      queued: "排队",
      leased: "已领取",
      running: "运行中",
      success: "成功",
      failed: "失败",
      canceled: "取消",
    }[status ?? ""] ?? status ?? "-"
  );
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

function showWorkerError(
  error: unknown,
  fallback: string,
  setMessage: (value: string) => void,
  setTone: (value: "default" | "success" | "warning" | "danger") => void,
) {
  setMessage(error instanceof Error ? error.message : fallback);
  setTone("danger");
}
