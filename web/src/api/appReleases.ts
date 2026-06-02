import { apiRequest, mockResponse } from "@/api/client";

const USE_MOCK = import.meta.env.VITE_USE_MOCK === "true";

export type AppReleaseChannel = "dev" | "beta" | "stable" | string;
export type AppBuildType = "debug" | "release" | string;
export type AppBuildStatus = "queued" | "running" | "success" | "failed" | "canceled" | string;
export type AppReleaseStatus = "draft" | "testing" | "scheduled" | "rolling_out" | "released" | "paused" | "recalled" | "archived" | string;
export type AppUpdateLevel = "normal" | "recommended" | "forced" | string;

export interface AppInfo {
  id: string;
  app_key: string;
  name: string;
  platform: string;
  package_name: string;
  description?: string;
  enabled?: boolean;
  current_stable_release?: AppRelease;
  current_beta_release?: AppRelease;
  latest_resource_version?: AppResourceVersion;
  active_devices?: number;
  created_at?: string;
  updated_at?: string;
}

export interface AppRelease {
  id: string;
  app_id?: string;
  build_id?: string;
  package_name: string;
  version_name: string;
  version_code: number;
  build_number?: number;
  channel: AppReleaseChannel;
  build_type: AppBuildType;
  signed?: boolean;
  git_ref: string;
  git_commit: string;
  api_base_url?: string;
  apk_path?: string;
  artifact_path?: string;
  file_name: string;
  size_bytes: number;
  sha256: string;
  status?: AppReleaseStatus;
  title?: string;
  summary?: string;
  release_notes?: string;
  release_notes_markdown?: string;
  upgrade_message?: string;
  update_level?: AppUpdateLevel;
  rollout_percentage?: number;
  target_type?: string;
  target_value?: string;
  min_supported_code?: number;
  block_old_versions?: boolean;
  scheduled_at?: string;
  is_latest: boolean;
  is_published: boolean;
  force_update?: boolean;
  published_at?: string;
  paused_at?: string;
  created_by?: string;
  created_at?: string;
  updated_at?: string;
  download_url?: string;
}

export interface AppReleaseBuildJob {
  id: string;
  status: AppBuildStatus;
  app_id?: string;
  git_ref: string;
  git_commit?: string;
  git_branch?: string;
  build_type: AppBuildType;
  channel: AppReleaseChannel;
  version_name: string;
  version_code: number;
  build_number?: number;
  build_environment?: string;
  artifact_type?: string;
  artifact_path?: string;
  artifact_size?: number;
  sha256?: string;
  file_name?: string;
  api_base_url?: string;
  release_notes?: string;
  started_by?: string;
  started_at?: string;
  created_at?: string;
  finished_at?: string;
  duration_ms?: number;
  log_tail?: string[];
  error_message?: string;
  release_id?: string;
  artifacts?: AppBuildArtifact[];
}

export interface AppBuildArtifact {
  id: string;
  build_id?: string;
  name: string;
  artifact_type: string;
  artifact_path?: string;
  file_name?: string;
  size_bytes?: number;
  sha256?: string;
  created_at?: string;
}

export interface AppResourcePackage {
  id?: string;
  package_key: string;
  package_type?: string;
  file_url: string;
  file_size: number;
  sha256: string;
  created_at?: string;
}

export interface AppResourceVersion {
  id: string;
  app_id?: string;
  resource_version: string;
  channel: AppReleaseChannel;
  status: AppReleaseStatus;
  min_app_version_code?: number;
  max_app_version_code?: number;
  update_level?: AppUpdateLevel;
  title: string;
  summary?: string;
  release_notes_markdown?: string;
  rollout_percentage?: number;
  manifest_url: string;
  manifest_signed?: boolean;
  signature_algorithm?: string;
  total_size: number;
  packages?: AppResourcePackage[];
  published_at?: string;
  paused_at?: string;
  created_by?: string;
  created_at?: string;
  updated_at?: string;
}

export interface AppInstallation {
  id: string;
  app_id?: string;
  user_id?: string | number;
  device_id: string;
  device_name?: string;
  platform?: string;
  os_version?: string;
  device_model?: string;
  installed_version?: string;
  installed_code?: number;
  build_number?: number;
  resource_version?: string;
  last_seen_at?: string;
  last_upgrade_at?: string;
  last_upgrade_status?: string;
  last_error?: string;
  created_at?: string;
  updated_at?: string;
}

export interface AppUpgradeEvent {
  id: string;
  app_id?: string;
  release_id?: string;
  user_id?: string | number;
  device_id: string;
  from_version_code?: number;
  to_version_code?: number;
  from_version?: string;
  to_version?: string;
  event_type: string;
  package_key?: string;
  error_message?: string;
  created_at?: string;
  category?: "apk" | "resource" | string;
}

export interface AppReleaseAuditLog {
  id: string;
  app_id?: string;
  operator: string;
  action: string;
  target_type: string;
  target_id?: string | number;
  before_json?: unknown;
  after_json?: unknown;
  created_at?: string;
}

export interface ReleaseQualityMetric {
  category: "apk" | "resource" | string;
  total_events: number;
  success_events: number;
  failure_events: number;
  success_rate: number;
  failure_rate: number;
  latest_failure_reason?: string;
  recommended_action?: string;
  action_reason?: string;
  policy_threshold?: string;
  failure_reasons?: Array<{ reason: string; count: number }>;
}

export interface QualityPolicy {
  resource_activation_failed_count: number;
  resource_failure_rate: number;
  apk_checksum_failed_count: number;
  apk_install_failed_count: number;
  apk_install_failure_rate: number;
}

export interface QualityAlert {
  id: string;
  category: "apk" | "resource" | string;
  severity: "warning" | "critical" | string;
  recommended_action: string;
  reason: string;
  threshold: string;
  failure_rate: number;
  failure_events: number;
  latest_failure?: string;
  created_at?: string;
}

export interface AppReleasesResponse {
  _mock?: boolean;
  apps?: AppInfo[];
  releases: AppRelease[];
  latest?: AppRelease;
  build_jobs: AppReleaseBuildJob[];
  resource_versions?: AppResourceVersion[];
  latest_resource?: AppResourceVersion;
  installations?: AppInstallation[];
  upgrade_events?: AppUpgradeEvent[];
  resource_update_events?: AppUpgradeEvent[];
  quality_metrics?: ReleaseQualityMetric[];
  quality_policy?: QualityPolicy;
  quality_alerts?: QualityAlert[];
  audit_logs?: AppReleaseAuditLog[];
}

export interface CreateAppReleaseBuildPayload {
  git_ref: string;
  build_type: AppBuildType;
  channel: AppReleaseChannel;
  version_name: string;
  version_code: number;
  build_number?: number;
  api_base_url?: string;
  release_notes?: string;
  apk_file?: File | null;
}

export interface CreateAppReleasePayload {
  build_id: string;
  channel: AppReleaseChannel;
  title: string;
  summary?: string;
  release_notes_markdown?: string;
  update_level: AppUpdateLevel;
  rollout_percentage: number;
  min_supported_code?: number;
  block_old_versions?: boolean;
  target_type?: string;
  target_value?: string;
  scheduled_at?: string;
}

export interface UpdateReleaseNotesPayload {
  title: string;
  summary?: string;
  release_notes_markdown: string;
}

export interface CreateAppResourceVersionPayload {
  resource_version: string;
  channel: AppReleaseChannel;
  title: string;
  summary?: string;
  release_notes_markdown?: string;
  min_app_version_code?: number;
  max_app_version_code?: number;
  update_level: AppUpdateLevel;
  rollout_percentage: number;
  files: File[];
}

export interface CreateAppPayload {
  app_key: string;
  name: string;
  platform: string;
  package_name: string;
  description?: string;
  enabled?: boolean;
}

export function getAppReleases() {
  if (USE_MOCK) return mockResponse(() => import("@/mocks/appReleases.mock").then((module) => module.appReleasesMock));
  return apiRequest<unknown>("/admin/api/app-releases").then(normalizeAppReleasesResponse);
}

export function createApp(payload: CreateAppPayload) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, app: mockAppFromPayload(payload) }));
  return apiRequest<unknown>("/admin/api/apps", {
    method: "POST",
    body: JSON.stringify(payload),
  }).then(normalizeAppActionResponse);
}

export function enableApp(id: string) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, app: mockAppAction(id, true) }));
  return apiRequest<unknown>(`/admin/api/apps/${encodeURIComponent(id)}/enable`, {
    method: "POST",
  }).then(normalizeAppActionResponse);
}

export function disableApp(id: string) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, app: mockAppAction(id, false) }));
  return apiRequest<unknown>(`/admin/api/apps/${encodeURIComponent(id)}/disable`, {
    method: "POST",
  }).then(normalizeAppActionResponse);
}

export function createAppReleaseBuild(payload: CreateAppReleaseBuildPayload) {
  if (USE_MOCK) return mockResponse(() => ({ job: mockBuildJob(payload) }));
  if (payload.apk_file) {
    const { apk_file, ...metadata } = payload;
    const formData = new FormData();
    formData.set("metadata", JSON.stringify(metadata));
    formData.set("apk", apk_file, apk_file.name);
    return apiRequest<unknown>("/admin/api/app-releases/build", {
      method: "POST",
      body: formData,
    }).then(normalizeAppReleaseBuildResponse);
  }
  return apiRequest<unknown>("/admin/api/app-releases/build", {
    method: "POST",
    body: JSON.stringify(payload),
  }).then(normalizeAppReleaseBuildResponse);
}

export function getAppReleaseBuildJob(jobId: string) {
  return apiRequest<unknown>(`/admin/api/app-releases/builds/${encodeURIComponent(jobId)}`).then(normalizeAppReleaseBuildResponse);
}

export function getAppReleaseBuildLogs(jobId: string) {
  return apiRequest<unknown>(`/admin/api/app-releases/builds/${encodeURIComponent(jobId)}/logs`).then((payload) =>
    normalizeAppReleaseBuildLogsResponse(payload, jobId),
  );
}

export function publishAppRelease(id: string, target: "latest" | "stable" = "latest") {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, release: mockReleaseAction(id, "released", true) }));
  return apiRequest<unknown>(`/admin/api/app-releases/${encodeURIComponent(id)}/publish`, {
    method: "POST",
    body: JSON.stringify({ target }),
  }).then(normalizeAppReleaseActionResponse);
}

export function createAppRelease(payload: CreateAppReleasePayload) {
  if (USE_MOCK) return mockResponse(() => ({ release: mockReleaseFromPayload(payload) }));
  return apiRequest<unknown>("/admin/api/app-releases", {
    method: "POST",
    body: JSON.stringify(payload),
  }).then(normalizeAppReleaseActionResponse);
}

export function updateAppReleaseRollout(id: string, rolloutPercentage: number) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, release: mockReleaseAction(id, "rolling_out", true, rolloutPercentage) }));
  return apiRequest<unknown>(`/admin/api/app-releases/${encodeURIComponent(id)}/rollout`, {
    method: "POST",
    body: JSON.stringify({ rollout_percentage: rolloutPercentage }),
  }).then(normalizeAppReleaseActionResponse);
}

export function updateAppReleaseNotes(id: string, payload: UpdateReleaseNotesPayload) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, release: mockReleaseNotesAction(id, payload) }));
  return apiRequest<unknown>(`/admin/api/app-releases/${encodeURIComponent(id)}/notes`, {
    method: "POST",
    body: JSON.stringify(payload),
  }).then(normalizeAppReleaseActionResponse);
}

export function pauseAppRelease(id: string) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, release: mockReleaseAction(id, "paused", true) }));
  return apiRequest<unknown>(`/admin/api/app-releases/${encodeURIComponent(id)}/pause`, {
    method: "POST",
  }).then(normalizeAppReleaseActionResponse);
}

export function recallAppRelease(id: string) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, release: mockReleaseAction(id, "recalled", false) }));
  return apiRequest<unknown>(`/admin/api/app-releases/${encodeURIComponent(id)}/recall`, {
    method: "POST",
  }).then(normalizeAppReleaseActionResponse);
}

export function unpublishAppRelease(id: string) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, release: mockReleaseAction(id, "recalled", false) }));
  return apiRequest<unknown>(`/admin/api/app-releases/${encodeURIComponent(id)}/unpublish`, {
    method: "POST",
  }).then(normalizeAppReleaseActionResponse);
}

export function rollbackAppRelease(id: string) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, release: mockReleaseAction(id, "released", true) }));
  return apiRequest<unknown>(`/admin/api/app-releases/${encodeURIComponent(id)}/rollback`, {
    method: "POST",
  }).then(normalizeAppReleaseActionResponse);
}

export function createAppResourceVersion(payload: CreateAppResourceVersionPayload) {
  if (USE_MOCK) return mockResponse(() => ({ resource_version: mockResourceFromPayload(payload) }));
  const { files, ...metadata } = payload;
  const formData = new FormData();
  formData.set("metadata", JSON.stringify(metadata));
  files.forEach((file) => formData.append("packages", file, file.name));
  return apiRequest<unknown>("/admin/api/app-resources", {
    method: "POST",
    body: formData,
  }).then(normalizeAppResourceVersionActionResponse);
}

export function publishAppResourceVersion(id: string) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, resource_version: mockResourceAction(id, "released") }));
  return apiRequest<unknown>(`/admin/api/app-resources/${encodeURIComponent(id)}/publish`, {
    method: "POST",
  }).then(normalizeAppResourceVersionActionResponse);
}

export function pauseAppResourceVersion(id: string) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, resource_version: mockResourceAction(id, "paused") }));
  return apiRequest<unknown>(`/admin/api/app-resources/${encodeURIComponent(id)}/pause`, {
    method: "POST",
  }).then(normalizeAppResourceVersionActionResponse);
}

export function rollbackAppResourceVersion(id: string) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, resource_version: mockResourceAction(id, "released") }));
  return apiRequest<unknown>(`/admin/api/app-resources/${encodeURIComponent(id)}/rollback`, {
    method: "POST",
  }).then(normalizeAppResourceVersionActionResponse);
}

export function updateAppResourceRollout(id: string, rolloutPercentage: number) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, resource_version: mockResourceAction(id, "rolling_out", rolloutPercentage) }));
  return apiRequest<unknown>(`/admin/api/app-resources/${encodeURIComponent(id)}/rollout`, {
    method: "POST",
    body: JSON.stringify({ rollout_percentage: rolloutPercentage }),
  }).then(normalizeAppResourceVersionActionResponse);
}

export function updateAppResourceNotes(id: string, payload: UpdateReleaseNotesPayload) {
  if (USE_MOCK) return mockResponse(() => ({ ok: true, resource_version: mockResourceNotesAction(id, payload) }));
  return apiRequest<unknown>(`/admin/api/app-resources/${encodeURIComponent(id)}/notes`, {
    method: "POST",
    body: JSON.stringify(payload),
  }).then(normalizeAppResourceVersionActionResponse);
}

export function normalizeAppReleasesResponse(payload: unknown): AppReleasesResponse {
  const record = asRecord(payload);
  const dataRecord = asRecord(record?.data);
  const nested = dataRecord ?? record;
  const releases = readArray<AppRelease>(Array.isArray(payload) ? payload : nested, ["releases", "items", "data"]);
  const buildJobs = readArray<AppReleaseBuildJob>(nested, ["build_jobs", "buildJobs", "jobs", "build_jobs_recent"]);
  const resourceVersions = readArray<AppResourceVersion>(nested, ["resource_versions", "resourceVersions", "resources"]);
  const resourceUpdateEvents = readArray<AppUpgradeEvent>(nested, [
    "resource_update_events",
    "resourceUpdateEvents",
    "resource_events",
  ]);
  const latest = readRecord(nested, ["latest", "latest_release", "current"]) as unknown as AppRelease | undefined;
  const latestResource = readRecord(nested, ["latest_resource", "latestResource", "current_resource"]) as unknown as
    | AppResourceVersion
    | undefined;

  return {
    _mock: Boolean(nested?._mock ?? record?._mock) || undefined,
    apps: readArray<AppInfo>(nested, ["apps", "applications"]),
    releases,
    latest: latest ?? releases.find((release) => release.is_latest && release.channel === "stable") ?? releases.find((release) => release.is_latest),
    build_jobs: buildJobs,
    resource_versions: resourceVersions,
    latest_resource:
      latestResource ??
      resourceVersions.find((resource) => resource.status === "released" && resource.channel === "stable") ??
      resourceVersions.find((resource) => resource.status === "released"),
    installations: readArray<AppInstallation>(nested, ["installations", "devices", "device_versions", "deviceVersions"]),
    upgrade_events: readArray<AppUpgradeEvent>(nested, ["upgrade_events", "upgradeEvents", "events"]),
    resource_update_events: resourceUpdateEvents,
    quality_metrics: readArray<ReleaseQualityMetric>(nested, ["quality_metrics", "qualityMetrics", "release_quality_metrics"]),
    quality_policy: readRecord(nested, ["quality_policy", "qualityPolicy"]) as unknown as QualityPolicy | undefined,
    quality_alerts: readArray<QualityAlert>(nested, ["quality_alerts", "qualityAlerts"]),
    audit_logs: readArray<AppReleaseAuditLog>(nested, ["audit_logs", "auditLogs", "operation_logs", "audits"]),
  };
}

export function normalizeAppReleaseBuildResponse(payload: unknown): { job: AppReleaseBuildJob } {
  const record = unwrapDataRecord(payload);
  const job = readRecord(record, ["job", "build_job", "buildJob"]) ?? record;
  if (!job) throw new Error("构建任务响应格式不正确");
  return { job: job as unknown as AppReleaseBuildJob };
}

export function normalizeAppReleaseBuildLogsResponse(payload: unknown, fallbackJobId = "") {
  const record = asRecord(payload);
  const logs = Array.isArray(payload)
    ? payload
    : readArray<unknown>(record, ["logs", "log_tail", "lines"]);
  return {
    job_id: stringValue(record?.job_id ?? record?.jobId ?? record?.id, fallbackJobId),
    logs: logs.map((line) => String(line)),
  };
}

export function normalizeAppActionResponse(payload: unknown): { ok: boolean; app: AppInfo } {
  const root = asRecord(payload);
  const record = unwrapDataRecord(payload);
  const app = readRecord(record, ["app", "application"]) ?? record;
  if (!app) throw new Error("应用响应格式不正确");
  return { ok: root?.ok !== false && record?.ok !== false, app: app as unknown as AppInfo };
}

export function normalizeAppReleaseActionResponse(payload: unknown): { ok: boolean; release: AppRelease } {
  const root = asRecord(payload);
  const record = unwrapDataRecord(payload);
  const release = readRecord(record, ["release", "app_release"]) ?? record;
  if (!release) throw new Error("版本响应格式不正确");
  return { ok: root?.ok !== false && record?.ok !== false, release: release as unknown as AppRelease };
}

export function normalizeAppResourceVersionActionResponse(payload: unknown): { ok: boolean; resource_version: AppResourceVersion } {
  const root = asRecord(payload);
  const record = unwrapDataRecord(payload);
  const resourceVersion = readRecord(record, ["resource_version", "resourceVersion", "resource"]) ?? record;
  if (!resourceVersion) throw new Error("资源版本响应格式不正确");
  return {
    ok: root?.ok !== false && record?.ok !== false,
    resource_version: resourceVersion as unknown as AppResourceVersion,
  };
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === "object" && !Array.isArray(value) ? (value as Record<string, unknown>) : undefined;
}

function unwrapDataRecord(value: unknown) {
  const record = asRecord(value);
  const dataRecord = asRecord(record?.data);
  return dataRecord ?? record;
}

function readRecord(record: Record<string, unknown> | undefined, keys: string[]) {
  if (!record) return undefined;
  for (const key of keys) {
    const value = record[key];
    const nested = asRecord(value);
    if (nested) return nested;
  }
  return undefined;
}

function readArray<T>(source: unknown, keys: string[]): T[] {
  if (Array.isArray(source)) return source as T[];
  const record = asRecord(source);
  if (!record) return [];
  for (const key of keys) {
    const value = record[key];
    if (Array.isArray(value)) return value as T[];
  }
  return [];
}

function stringValue(value: unknown, fallback: string) {
  return typeof value === "string" && value.trim() ? value : fallback;
}

function mockBuildJob(payload: CreateAppReleaseBuildPayload): AppReleaseBuildJob {
  return {
    id: `build_mock_${Date.now()}`,
    status: "queued",
    git_ref: payload.git_ref,
    build_type: payload.build_type,
    channel: payload.channel,
    version_name: payload.version_name,
    version_code: payload.version_code,
    build_number: payload.build_number ?? payload.version_code,
    build_environment: payload.channel === "stable" ? "release" : "test",
    started_by: "mock-admin",
    started_at: new Date().toISOString(),
    log_tail: ["Mock 构建任务已创建", "等待后端构建队列处理"],
  };
}

function mockAppFromPayload(payload: CreateAppPayload): AppInfo {
  return {
    id: `app_mock_${Date.now()}`,
    app_key: payload.app_key,
    name: payload.name,
    platform: payload.platform,
    package_name: payload.package_name,
    description: payload.description,
    enabled: payload.enabled ?? true,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  };
}

function mockAppAction(id: string, enabled: boolean): AppInfo {
  return {
    id,
    app_key: "game-helper-android",
    name: "游戏助手",
    platform: "android",
    package_name: "com.kingdomhelper.executor",
    enabled,
    updated_at: new Date().toISOString(),
  };
}

function mockReleaseAction(id: string, status: AppReleaseStatus, published: boolean, rolloutPercentage = 100): AppRelease {
  return {
    id,
    package_name: "com.kingdomhelper.executor",
    version_name: "0.1.0",
    version_code: 1,
    channel: "dev",
    build_type: "debug",
    git_ref: "mock",
    git_commit: "mock",
    file_name: "mock.apk",
    size_bytes: 1,
    sha256: "mock",
    status,
    rollout_percentage: rolloutPercentage,
    is_latest: published,
    is_published: published,
    created_at: new Date().toISOString(),
  };
}

function mockReleaseFromPayload(payload: CreateAppReleasePayload): AppRelease {
  return {
    id: `rel_mock_${Date.now()}`,
    build_id: payload.build_id,
    package_name: "com.kingdomhelper.executor",
    version_name: "0.1.0-mock",
    version_code: Date.now() % 100_000,
    channel: payload.channel,
    build_type: "release",
    git_ref: "mock",
    git_commit: "mock",
    file_name: `kingdom-helper-${payload.channel}-mock.apk`,
    size_bytes: 1,
    sha256: "mock",
    title: payload.title,
    summary: payload.summary,
    release_notes_markdown: payload.release_notes_markdown,
    update_level: payload.update_level,
    rollout_percentage: payload.rollout_percentage,
    target_type: payload.target_type,
    target_value: payload.target_value,
    min_supported_code: payload.min_supported_code,
    block_old_versions: payload.block_old_versions,
    scheduled_at: payload.scheduled_at,
    status: payload.scheduled_at ? "scheduled" : "draft",
    is_latest: false,
    is_published: false,
    created_by: "mock-admin",
    created_at: new Date().toISOString(),
  };
}

function mockReleaseNotesAction(id: string, payload: UpdateReleaseNotesPayload): AppRelease {
  return {
    ...mockReleaseAction(id, "draft", false),
    title: payload.title,
    summary: payload.summary,
    release_notes_markdown: payload.release_notes_markdown,
  };
}

function mockResourceFromPayload(payload: CreateAppResourceVersionPayload): AppResourceVersion {
  const packages = payload.files.map((file) => ({
    package_key: file.name.replace(/-\d{8}\.\d+\.zip$/i, "").replace(/\.zip$/i, "") || "resource-package",
    package_type: "zip",
    file_url: `/admin/api/app-resources/mock/packages/${encodeURIComponent(file.name)}`,
    file_size: file.size,
    sha256: "mock",
    created_at: new Date().toISOString(),
  }));
  return {
    id: `res_mock_${Date.now()}`,
    resource_version: payload.resource_version,
    channel: payload.channel,
    status: "draft",
    min_app_version_code: payload.min_app_version_code,
    max_app_version_code: payload.max_app_version_code,
    update_level: payload.update_level,
    title: payload.title,
    summary: payload.summary,
    release_notes_markdown: payload.release_notes_markdown,
    rollout_percentage: payload.rollout_percentage,
    manifest_url: `/admin/api/app-resources/mock/manifest-${payload.resource_version}.json`,
    total_size: packages.reduce((sum, item) => sum + item.file_size, 0),
    packages,
    created_by: "mock-admin",
    created_at: new Date().toISOString(),
  };
}

function mockResourceNotesAction(id: string, payload: UpdateReleaseNotesPayload): AppResourceVersion {
  return {
    ...mockResourceAction(id, "draft"),
    title: payload.title,
    summary: payload.summary,
    release_notes_markdown: payload.release_notes_markdown,
  };
}

function mockResourceAction(id: string, status: AppReleaseStatus, rolloutPercentage = 100): AppResourceVersion {
  return {
    id,
    resource_version: "20260601.1",
    channel: "dev",
    status,
    title: "Mock 资源版本",
    rollout_percentage: rolloutPercentage,
    manifest_url: "/admin/api/app-resources/mock/manifest.json",
    total_size: 1,
    created_at: new Date().toISOString(),
  };
}
