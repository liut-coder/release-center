import { ApiError } from "@/api/errors";

export { ApiError, isApiError } from "@/api/errors";

const DEFAULT_TIMEOUT_MS = 30_000;

export interface ApiRequestInit extends RequestInit {
  auth?: "admin" | "app" | "none";
}

export async function apiRequest<T>(path: string, init: ApiRequestInit = {}): Promise<T> {
  const { auth: _auth, ...requestInit } = init;
  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), DEFAULT_TIMEOUT_MS);
  const requestId = crypto.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  const token = localStorage.getItem("release-center-admin-token") ?? "";

  try {
    const response = await fetch(`${apiBaseUrl()}${path}`, {
      ...requestInit,
      signal: combineSignals(controller.signal, requestInit.signal),
      headers: buildHeaders(
        {
          Accept: "application/json",
          "X-Request-Id": requestId,
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        requestInit,
      ),
    });
    const text = await response.text();
    const data = text ? safeJson(text) : null;
    if (!response.ok) {
      throw new ApiError({
        status: response.status,
        code: recordString(data, "code") || `http.${response.status}`,
        message: recordString(data, "message") || response.statusText || "请求失败",
        requestId: recordString(data, "request_id") || response.headers.get("x-request-id") || requestId,
        detail: data,
      });
    }
    return data as T;
  } catch (error) {
    if (error instanceof ApiError) throw error;
    throw new ApiError({
      status: 0,
      message: error instanceof DOMException && error.name === "AbortError" ? "请求后端超时" : "无法连接后端服务",
      detail: error,
    });
  } finally {
    window.clearTimeout(timeout);
  }
}

export async function mockResponse<T>(loader: () => Promise<T> | T): Promise<T> {
  return loader();
}

export function saveAppTokens(tokens: { configToken?: string; runtimeControlToken?: string }) {
  if (tokens.configToken !== undefined) localStorage.setItem("release-center-admin-token", tokens.configToken.trim());
  if (tokens.runtimeControlToken !== undefined) localStorage.setItem("release-center-admin-token", tokens.runtimeControlToken.trim());
}

export function getAppTokens() {
  return {
    configToken: localStorage.getItem("release-center-admin-token") ?? "",
    runtimeControlToken: localStorage.getItem("release-center-admin-token") ?? "",
  };
}

function apiBaseUrl() {
  const value = import.meta.env.VITE_API_BASE_URL;
  return typeof value === "string" && value.trim() ? value.replace(/\/$/, "") : "";
}

function buildHeaders(defaults: HeadersInit, init: RequestInit) {
  const headers = new Headers(defaults);
  const customHeaders = new Headers(init.headers);
  const shouldSendJsonContentType = typeof init.body === "string";
  if (shouldSendJsonContentType) headers.set("Content-Type", "application/json");
  customHeaders.forEach((value, key) => headers.set(key, value));
  if (!shouldSendJsonContentType && !customHeaders.has("Content-Type")) headers.delete("Content-Type");
  return headers;
}

function combineSignals(timeoutSignal: AbortSignal, externalSignal?: AbortSignal | null) {
  if (!externalSignal) return timeoutSignal;
  if (typeof AbortSignal.any === "function") return AbortSignal.any([timeoutSignal, externalSignal]);
  const controller = new AbortController();
  timeoutSignal.addEventListener("abort", () => controller.abort(), { once: true });
  externalSignal.addEventListener("abort", () => controller.abort(), { once: true });
  return controller.signal;
}

function safeJson(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

function recordString(value: unknown, key: string) {
  return value && typeof value === "object" && !Array.isArray(value) && typeof (value as Record<string, unknown>)[key] === "string"
    ? ((value as Record<string, string>)[key] ?? "")
    : "";
}
