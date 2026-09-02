export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  // code is wire error.code when present; TIMEOUT is used for client-side abort deadline.
  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }
}
