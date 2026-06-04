import { apiRequest } from "@/api/client";

export type ArtifactSource = "app_build" | "build_center" | string;

export interface ArtifactCenterItem {
  id: string;
  source: ArtifactSource;
  project_key?: string;
  project_name?: string;
  app_key?: string;
  app_name?: string;
  build_id?: string;
  run_id?: string;
  app_build_artifact_id?: string;
  name: string;
  artifact_type: string;
  file_name?: string;
  location?: string;
  immutable_ref?: string;
  size_bytes?: number;
  sha256?: string;
  upload_status?: string;
  version_name?: string;
  build_number?: number;
  git_commit?: string;
  channel?: string;
  status?: string;
  metadata?: unknown;
  created_at: string;
}

export interface ArtifactCenterOverview {
  artifacts: ArtifactCenterItem[];
  message_zh?: string;
}

export interface ArtifactCenterItemResponse {
  ok: boolean;
  artifact: ArtifactCenterItem;
  message_zh?: string;
}

export function getArtifactCenterOverview() {
  return apiRequest<ArtifactCenterOverview>("/admin/api/artifacts");
}

export function getArtifactCenterItem(artifactId: string) {
  return apiRequest<ArtifactCenterItemResponse>(`/admin/api/artifacts/${encodeURIComponent(artifactId)}`);
}
