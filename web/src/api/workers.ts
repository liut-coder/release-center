import { apiRequest } from "@/api/client";

export type WorkerStatus = "registered" | "online" | "busy" | "draining" | "offline" | string;
export type WorkerTaskStatus = "queued" | "leased" | "running" | "success" | "failed" | "canceled" | string;

export interface BuildWorker {
  id: string;
  worker_key: string;
  name: string;
  endpoint_url?: string;
  labels: string[];
  status: WorkerStatus;
  capacity: number;
  running_tasks: number;
  last_seen_at?: string;
  metadata?: unknown;
  created_at?: string;
  updated_at?: string;
}

export interface WorkerTask {
  id: string;
  worker_id?: string;
  build_run_id?: string;
  project_id?: string;
  build_profile_id?: string;
  task_type: string;
  action: string;
  status: WorkerTaskStatus;
  required_labels: string[];
  priority: number;
  lease_token?: string;
  leased_until?: string;
  attempts: number;
  log_tail?: string[];
  artifact_manifest?: unknown;
  error_message?: string;
  metadata?: unknown;
  started_at?: string;
  finished_at?: string;
  created_at?: string;
  updated_at?: string;
}

export interface WorkerOverview {
  workers: BuildWorker[];
  tasks: WorkerTask[];
  message_zh?: string;
}

export interface CreateWorkerTaskPayload {
  project_key?: string;
  build_profile_id?: string;
  build_run_id?: string;
  task_type?: string;
  action?: string;
  required_labels?: string[];
  priority?: number;
  metadata?: Record<string, unknown>;
}

export function getWorkerOverview() {
  return apiRequest<WorkerOverview>("/admin/api/workers");
}

export function createWorkerTask(payload: CreateWorkerTaskPayload) {
  return apiRequest<{ ok: boolean; task: WorkerTask; message_zh?: string }>("/admin/api/workers/tasks", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}
