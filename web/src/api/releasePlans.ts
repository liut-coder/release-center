import { apiRequest } from "@/api/client";
import type { DeploymentRecord } from "@/api/deployments";
import type { WorkerTask } from "@/api/workers";

export type ReleaseUnitType = "android" | "web" | "docs" | "worker" | "server" | "docker" | "config" | string;
export type ReleasePlanStatus =
  | "draft"
  | "scheduled"
  | "queued"
  | "pending_approval"
  | "released"
  | "rolling_out"
  | "paused"
  | "recalled"
  | "rolled_back"
  | "archived"
  | string;

export interface ReleaseEnvironment {
  id: string;
  environment_key: string;
  name: string;
  sort_order: number;
  requires_approval: boolean;
  metadata?: unknown;
  created_at?: string;
  updated_at?: string;
}

export interface ReleaseUnit {
  id: string;
  project_id: string;
  project_key?: string;
  app_id?: string;
  unit_key: string;
  name: string;
  unit_type: ReleaseUnitType;
  default_channel: string;
  enabled: boolean;
  metadata?: unknown;
  created_at?: string;
  updated_at?: string;
}

export interface ReleasePlanArtifact {
  id?: string;
  release_plan_id?: string;
  build_run_id?: string;
  app_build_id?: string;
  app_build_artifact_id?: string;
  artifact_name: string;
  artifact_type: string;
  file_name?: string;
  immutable_ref?: string;
  metadata?: unknown;
  created_at?: string;
  updated_at?: string;
}

export interface ReleasePlan {
  id: string;
  project_id: string;
  project_key?: string;
  release_unit_id: string;
  unit_key?: string;
  unit_type?: ReleaseUnitType;
  environment_id: string;
  environment_key?: string;
  environment_requires_approval?: boolean;
  plan_key: string;
  title: string;
  description?: string;
  version_name?: string;
  build_number?: number;
  git_commit?: string;
  channel: string;
  status: ReleasePlanStatus;
  rollout_percentage: number;
  target_type: string;
  target_value?: string;
  scheduled_at?: string;
  published_at?: string;
  paused_at?: string;
  approved_by?: string;
  metadata?: unknown;
  created_by?: string;
  artifacts?: ReleasePlanArtifact[];
  created_at?: string;
  updated_at?: string;
}

export interface ReleasePlanOverview {
  environments: ReleaseEnvironment[];
  release_units: ReleaseUnit[];
  release_plans: ReleasePlan[];
  message_zh?: string;
}

export interface CreateReleaseUnitPayload {
  project_key: string;
  app_id?: string;
  unit_key: string;
  name: string;
  unit_type: ReleaseUnitType;
  default_channel?: string;
  enabled?: boolean;
  metadata?: Record<string, unknown>;
}

export interface CreateReleasePlanPayload {
  project_key: string;
  unit_key: string;
  environment_key: string;
  plan_key?: string;
  title: string;
  description?: string;
  version_name?: string;
  build_number?: number;
  git_commit?: string;
  channel?: string;
  status?: ReleasePlanStatus;
  rollout_percentage?: number;
  target_type?: string;
  target_value?: string;
  scheduled_at?: string;
  approved_by?: string;
  created_by?: string;
  artifacts?: Array<{
    build_run_id?: string;
    app_build_id?: string;
    app_build_artifact_id?: string;
    artifact_name: string;
    artifact_type: string;
    file_name?: string;
    immutable_ref?: string;
    metadata?: Record<string, unknown>;
  }>;
  metadata?: Record<string, unknown>;
}

export interface ReleasePlanActionPayload {
  approved_by?: string;
  target_id?: string;
  target_key?: string;
  dry_run?: boolean;
  triggered_by?: string;
  deployment_url?: string;
  metadata?: Record<string, unknown>;
}

export interface CreateReleasePlanDeploymentPayload {
  target_id?: string;
  target_key?: string;
  dry_run?: boolean;
  triggered_by?: string;
  deployment_url?: string;
  metadata?: Record<string, unknown>;
}

export function getReleasePlanOverview() {
  return apiRequest<ReleasePlanOverview>("/admin/api/release-plans");
}

export function createReleaseUnit(payload: CreateReleaseUnitPayload) {
  return apiRequest<{ ok: boolean; release_unit: ReleaseUnit; message_zh?: string }>("/admin/api/release-units", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function createReleasePlan(payload: CreateReleasePlanPayload) {
  return apiRequest<{ ok: boolean; plan: ReleasePlan; message_zh?: string }>("/admin/api/release-plans", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function publishReleasePlan(planId: string, payload: ReleasePlanActionPayload = {}) {
  return releasePlanAction(planId, "publish", payload);
}

export function pauseReleasePlan(planId: string, payload: ReleasePlanActionPayload = {}) {
  return releasePlanAction(planId, "pause", payload);
}

export function approveReleasePlan(planId: string, payload: ReleasePlanActionPayload = {}) {
  return releasePlanAction(planId, "approve", payload);
}

export function rollbackReleasePlan(planId: string, payload: ReleasePlanActionPayload = {}) {
  return releasePlanAction(planId, "rollback", payload);
}

export function createReleasePlanDeployment(planId: string, payload: CreateReleasePlanDeploymentPayload) {
  return apiRequest<{ ok: boolean; plan: ReleasePlan; deployment_records: DeploymentRecord[]; worker_tasks?: WorkerTask[]; message_zh?: string }>(
    `/admin/api/release-plans/${encodeURIComponent(planId)}/deployments`,
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
  );
}

function releasePlanAction(planId: string, action: "publish" | "approve" | "pause" | "rollback", payload: ReleasePlanActionPayload) {
  return apiRequest<{
    ok: boolean;
    plan: ReleasePlan;
    rollback_plan?: ReleasePlan;
    deployment_records?: DeploymentRecord[];
    worker_tasks?: WorkerTask[];
    message_zh?: string;
  }>(
    `/admin/api/release-plans/${encodeURIComponent(planId)}/${action}`,
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
  );
}
