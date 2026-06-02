export class ApiError extends Error {
  status: number;
  code?: string;
  requestId?: string;
  detail?: unknown;

  constructor(input: { status: number; message: string; code?: string; requestId?: string; detail?: unknown }) {
    super(input.message);
    this.name = "ApiError";
    this.status = input.status;
    this.code = input.code;
    this.requestId = input.requestId;
    this.detail = input.detail;
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}
