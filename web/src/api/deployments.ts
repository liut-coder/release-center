import { apiRequest } from "@/api/client";
import type { WorkerTask } from "@/api/workers";

export type DeploymentProvider = "cloudflare_pages" | "cloudflare_worker" | "cloudflare_r2" | "generic_webhook" | "ssh" | "docker" | "kubernetes" | string;
export type DeploymentStatus = "queued" | "running" | "success" | "failed" | "canceled" | "dry_run" | "external" | string;

export interface DeploymentTarget {
  id: string;
  project_id: string;
  app_id?: string;
  target_key: string;
  name: string;
  provider: DeploymentProvider;
  environment: string;
  endpoint_url?: string;
  cloudflare_account_id?: string;
  cloudflare_project_name?: string;
  cloudflare_script_name?: string;
  cloudflare_bucket_name?: string;
  credential_ref?: string;
  enabled?: boolean;
  metadata?: unknown;
  created_at?: string;
  updated_at?: string;
}

export interface DeploymentRecord {
  id: string;
  target_id: string;
  target_key?: string;
  target_name?: string;
  project_id?: string;
  run_id?: string;
  app_build_id?: string;
  app_build_artifact_id?: string;
  provider: DeploymentProvider;
  environment?: string;
  external_deployment_id?: string;
  provider_status: DeploymentStatus;
  deployment_url?: string;
  version_name?: string;
  build_number?: number;
  git_commit?: string;
  triggered_by?: string;
  started_at?: string;
  finished_at?: string;
  duration_ms?: number;
  log_tail?: string[];
  error_message?: string;
  metadata?: unknown;
  created_at?: string;
  updated_at?: string;
}

export interface CreateDeploymentTargetPayload {
  project_key: string;
  app_id?: string;
  target_key: string;
  name: string;
  provider: DeploymentProvider;
  environment: string;
  endpoint_url?: string;
  cloudflare_account_id?: string;
  cloudflare_project_name?: string;
  cloudflare_script_name?: string;
  cloudflare_bucket_name?: string;
  credential_ref?: string;
  enabled?: boolean;
  metadata?: Record<string, unknown>;
}

export interface CreateDeploymentPayload {
  project_key?: string;
  target_id?: string;
  target_key?: string;
  run_id?: string;
  app_build_id?: string;
  app_build_artifact_id?: string;
  external_deployment_id?: string;
  provider_status?: DeploymentStatus;
  deployment_url?: string;
  version_name?: string;
  build_number?: number;
  git_commit?: string;
  triggered_by?: string;
  log_tail?: string[];
  error_message?: string;
  dry_run?: boolean;
  metadata?: Record<string, unknown>;
}

export interface UpdateDeploymentStatusPayload {
  external_deployment_id?: string;
  provider_status?: DeploymentStatus;
  deployment_url?: string;
  log_tail?: string[];
  error_message?: string;
  metadata?: Record<string, unknown>;
}

export function getDeploymentTargets() {
  return apiRequest<{ deployment_targets: DeploymentTarget[] }>("/admin/api/deployment-targets");
}

export function createDeploymentTarget(payload: CreateDeploymentTargetPayload) {
  return apiRequest<{ ok: boolean; target: DeploymentTarget; message_zh?: string }>("/admin/api/deployment-targets", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function getDeployments() {
  return apiRequest<{ deployment_records: DeploymentRecord[]; message_zh?: string }>("/admin/api/deployments");
}

export function createDeployment(payload: CreateDeploymentPayload) {
  return apiRequest<{ ok: boolean; record: DeploymentRecord; worker_task?: WorkerTask; message_zh?: string }>("/admin/api/deployments", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function completeDeployment(deploymentId: string, payload: UpdateDeploymentStatusPayload = {}) {
  return apiRequest<{ ok: boolean; record: DeploymentRecord; message_zh?: string }>(
    `/admin/api/deployments/${encodeURIComponent(deploymentId)}/complete`,
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
  );
}

export function failDeployment(deploymentId: string, payload: UpdateDeploymentStatusPayload = {}) {
  return apiRequest<{ ok: boolean; record: DeploymentRecord; message_zh?: string }>(
    `/admin/api/deployments/${encodeURIComponent(deploymentId)}/fail`,
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
  );
}
