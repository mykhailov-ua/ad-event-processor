export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  // status 0 + code TIMEOUT: client deadline in api/client.ts, not a control-plane HTTP status.
  // Other codes mirror OpenAPI error.code when parseApiError finds a JSON envelope.
  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }
}
