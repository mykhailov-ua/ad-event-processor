import { ApiError, AuthError, NetworkError } from './api_client.js';

export type ServiceErrorView = {
  kind: string;
  title: string;
  message: string;
  status?: number;
  code?: string;
};

export function mapServiceError(err: unknown): ServiceErrorView {
  if (err instanceof AuthError) {
    return {
      kind: 'redirect_login',
      title: 'Session expired',
      message: 'Please sign in again.',
      status: 401,
      code: 'UNAUTHORIZED',
    };
  }

  if (err instanceof NetworkError) {
    return {
      kind: 'toast',
      title: 'Network error',
      message: err.message || 'Could not reach the server.',
      code: 'NETWORK_ERROR',
    };
  }

  if (!(err instanceof ApiError)) {
    const message = err instanceof Error ? err.message : 'request failed';
    return {
      kind: 'toast',
      title: 'Error',
      message,
      code: 'UNKNOWN',
    };
  }

  if (err.status === 403) {
    return {
      kind: 'page',
      title: 'Access denied',
      message: 'You do not have permission for this resource.',
      status: err.status,
      code: err.code,
    };
  }

  return {
    kind: 'toast',
    title: 'Error',
    message: err.message || err.code,
    status: err.status,
    code: err.code,
  };
}
