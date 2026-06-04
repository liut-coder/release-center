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

export function getBuildCenterOverview() {
  return apiRequest<BuildCenterOverview>("/admin/api/build-center/projects");
}

export function getBuildCenterProject(projectKey: string) {
  return apiRequest<BuildCenterProject>(`/admin/api/build-center/projects/${encodeURIComponent(projectKey)}`);
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
