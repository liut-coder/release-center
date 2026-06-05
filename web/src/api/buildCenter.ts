import { apiRequest } from "@/api/client";

export type BuildRunStatus = "queued" | "running" | "success" | "failed" | "canceled" | string;
export type BuildRunAction = "fetch" | "prepare" | "build" | "image" | "upload" | "verify" | "all" | "status" | string;

export interface BuildCenterOverview {
  projects: BuildCenterProject[];
  deployment_targets: DeploymentTarget[];
  message_zh?: string;
}

export interface BuildCenterProject {
  id: string;
  project_key: string;
  name: string;
  description?: string;
  owner_account?: string;
  lifecycle_status?: string;
  default_channel?: string;
  metadata?: unknown;
  repositories?: CodeRepository[];
  build_profiles?: BuildProfile[];
  recent_runs?: BuildCenterRun[];
  deployment_targets?: DeploymentTarget[];
  webhook_routes?: WebhookRoute[];
  created_at?: string;
  updated_at?: string;
}

export interface CodeRepository {
  id: string;
  project_id: string;
  provider: string;
  repo_url: string;
  repo_full_name: string;
  default_ref: string;
  credential_ref?: string;
  webhook_secret_ref?: string;
  webhook_enabled?: boolean;
  trigger_on_push?: boolean;
  trigger_on_tag?: boolean;
  metadata?: unknown;
  created_at?: string;
  updated_at?: string;
}

export interface BuildProfile {
  id: string;
  project_id: string;
  app_id?: string;
  profile_key: string;
  name: string;
  build_center_project: string;
  stack_type: string;
  build_type: string;
  config_path?: string;
  source_workdir?: string;
  default_ref?: string;
  default_version_name?: string;
  default_version_code?: number;
  default_channel?: string;
  build_action?: string;
  commands?: unknown;
  artifact_rules?: unknown;
  enabled?: boolean;
  metadata?: unknown;
  created_at?: string;
  updated_at?: string;
}

export interface WebhookRoute {
  id: string;
  project_id: string;
  repository_id: string;
  build_profile_id?: string;
  profile_key?: string;
  event_type: string;
  ref_pattern: string;
  action: BuildRunAction;
  enabled?: boolean;
  metadata?: unknown;
  created_at?: string;
  updated_at?: string;
}

export interface WebhookRouteDryRunPayload {
  provider: string;
  repository: string;
  event_type: string;
  ref: string;
  commit_sha?: string;
  sender?: string;
}

export interface WebhookRouteDryRunMatch {
  route: {
    id: string;
    project_key: string;
    repository: string;
    profile_key: string;
    event_type: string;
    ref_pattern: string;
    action: BuildRunAction;
    trigger_on_push: boolean;
    trigger_on_tag: boolean;
  };
  event_ok: boolean;
  ref_ok: boolean;
  matched: boolean;
  build_ref?: string;
  block_reason?: string;
}

export interface WebhookRouteDryRunResponse {
  ok: boolean;
  event: {
    provider: string;
    event_type: string;
    repository: string;
    ref: string;
    commit_sha?: string;
    sender?: string;
  };
  matches: WebhookRouteDryRunMatch[];
  message_zh?: string;
}

export interface BuildCenterRun {
  id: string;
  project_id: string;
  build_profile_id?: string;
  app_build_id?: string;
  trigger_type?: string;
  trigger_source?: string;
  action: BuildRunAction;
  git_ref?: string;
  git_commit?: string;
  version_name?: string;
  version_code?: number;
  build_number?: number;
  channel?: string;
  status: BuildRunStatus;
  exit_code?: number;
  started_by?: string;
  started_at?: string;
  finished_at?: string;
  duration_ms?: number;
  workspace_dir?: string;
  artifact_dir?: string;
  log_dir?: string;
  manifest_path?: string;
  upload_status?: string;
  error_message?: string;
  artifacts?: BuildCenterRunArtifact[];
  metadata?: unknown;
  created_at?: string;
  updated_at?: string;
}

export interface BuildCenterRunArtifact {
  id: string;
  run_id: string;
  app_build_artifact_id?: string;
  name: string;
  artifact_type: string;
  file_name: string;
  local_path?: string;
  size_bytes?: number;
  sha256?: string;
  upload_status?: string;
  download_url?: string;
  metadata?: unknown;
  created_at?: string;
  updated_at?: string;
}

export interface BuildCenterRunLogs {
  run_id: string;
  log_path?: string;
  lines: string[];
  truncated?: boolean;
  message_zh?: string;
}

export interface DeploymentTarget {
  id: string;
  project_id: string;
  app_id?: string;
  target_key: string;
  name: string;
  provider: string;
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

export interface CreateBuildCenterRunPayload {
  profile_key?: string;
  action?: BuildRunAction;
  git_ref?: string;
  version_name?: string;
  version_code?: number;
  channel?: string;
  started_by?: string;
}

export interface CreateBuildCenterProjectPayload {
  project_key: string;
  name: string;
  description?: string;
  owner_account?: string;
  lifecycle_status?: string;
  default_channel?: string;
  metadata?: Record<string, unknown>;
}

export interface UpsertCodeRepositoryPayload {
  provider: string;
  repo_url: string;
  repo_full_name?: string;
  default_ref?: string;
  credential_ref?: string;
  webhook_secret_ref?: string;
  webhook_enabled?: boolean;
  trigger_on_push?: boolean;
  trigger_on_tag?: boolean;
  metadata?: Record<string, unknown>;
}

export interface UpsertBuildProfilePayload {
  app_id?: string;
  profile_key: string;
  name: string;
  build_center_project: string;
  stack_type?: string;
  build_type?: string;
  config_path?: string;
  source_workdir?: string;
  default_ref?: string;
  default_version_name?: string;
  default_version_code?: number;
  default_channel?: string;
  build_action?: BuildRunAction;
  commands?: unknown;
  artifact_rules?: unknown;
  enabled?: boolean;
  metadata?: Record<string, unknown>;
}

export interface UpsertWebhookRoutePayload {
  repository_id: string;
  profile_key?: string;
  event_type?: string;
  ref_pattern?: string;
  action?: BuildRunAction;
  enabled?: boolean;
  metadata?: Record<string, unknown>;
}

export function getBuildCenterOverview() {
  return apiRequest<BuildCenterOverview>("/admin/api/build-center/projects");
}

export function getBuildCenterProject(projectKey: string) {
  return apiRequest<BuildCenterProject>(`/admin/api/build-center/projects/${encodeURIComponent(projectKey)}`);
}

export function getBuildCenterRun(runId: string) {
  return apiRequest<{ run: BuildCenterRun }>(`/admin/api/build-center/runs/${encodeURIComponent(runId)}`);
}

export function getBuildCenterRunLogs(runId: string) {
  return apiRequest<BuildCenterRunLogs>(`/admin/api/build-center/runs/${encodeURIComponent(runId)}/logs`);
}

export function getDeploymentTargets() {
  return apiRequest<{ deployment_targets: DeploymentTarget[] }>("/admin/api/deployment-targets");
}

export function createBuildCenterRun(projectKey: string, payload: CreateBuildCenterRunPayload) {
  return apiRequest<{ run: BuildCenterRun }>(`/admin/api/build-center/projects/${encodeURIComponent(projectKey)}/runs`, {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function createBuildCenterProject(payload: CreateBuildCenterProjectPayload) {
  return apiRequest<{ ok: boolean; project: BuildCenterProject; message_zh?: string }>("/admin/api/build-center/projects", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function upsertCodeRepository(projectKey: string, payload: UpsertCodeRepositoryPayload) {
  return apiRequest<{ ok: boolean; repository: CodeRepository; message_zh?: string }>(
    `/admin/api/build-center/projects/${encodeURIComponent(projectKey)}/repositories`,
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
  );
}

export function upsertBuildProfile(projectKey: string, payload: UpsertBuildProfilePayload) {
  return apiRequest<{ ok: boolean; build_profile: BuildProfile; message_zh?: string }>(
    `/admin/api/build-center/projects/${encodeURIComponent(projectKey)}/profiles`,
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
  );
}

export function upsertWebhookRoute(projectKey: string, payload: UpsertWebhookRoutePayload) {
  return apiRequest<{ ok: boolean; webhook_route: WebhookRoute; message_zh?: string }>(
    `/admin/api/build-center/projects/${encodeURIComponent(projectKey)}/webhook-routes`,
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
  );
}

export function dryRunWebhookRoute(payload: WebhookRouteDryRunPayload) {
  return apiRequest<WebhookRouteDryRunResponse>("/admin/api/build-center/webhook-routes/dry-run", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}
