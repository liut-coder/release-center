import { apiRequest } from "@/api/client";

export interface IntegrationCredentialStatus {
  ref: string;
  name: string;
  kind: string;
  provider: string;
  usage: string;
  env_vars: string[];
  configured: boolean;
  configured_env_vars?: string[];
}

export interface IntegrationCredentialOverview {
  credentials: IntegrationCredentialStatus[];
  message_zh?: string;
}

export function getIntegrationCredentials() {
  return apiRequest<IntegrationCredentialOverview>("/admin/api/integration-credentials");
}
