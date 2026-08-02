import { ApiError, AuthError, NetworkError } from './api_client.js';

/**
 * @typedef {'inline'|'page'|'toast'|'redirect_login'|'stub'|'retry'|'conflict'|'empty'|'unavailable'} ServiceErrorKind
 */

/**
 * @typedef {Object} ServiceErrorView
 * @property {ServiceErrorKind} kind
 * @property {string} title
 * @property {string} message
 * @property {number} [status]
 * @property {string} [code]
 * @property {boolean} [stub]
 * @property {number} [retryAfterSec]
 */

/**
 * @param {unknown} err
 * @returns {ServiceErrorView}
 */
export function mapServiceError(err) {
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

  const retryAfterSec = parseRetryAfter(err.responseHeaders?.get('Retry-After'));

  const base = {
    status: err.status,
    code: err.code,
    message: err.message,
    stub: err.stub,
    retryAfterSec,
  };

  switch (err.code) {
    case 'BAD_REQUEST':
      return {
        ...base,
        kind: 'inline',
        title: 'Invalid request',
        message: err.message,
      };
    case 'UNAUTHORIZED':
      return {
        ...base,
        kind: 'redirect_login',
        title: 'Session expired',
        message: err.message,
      };
    case 'FORBIDDEN':
      return {
        ...base,
        kind: 'page',
        title: 'Access denied',
        message: 'You do not have permission for this resource.',
      };
    case 'NOT_FOUND':
      return {
        ...base,
        kind: 'empty',
        title: 'Not found',
        message: 'The resource does not exist or was removed.',
      };
    case 'CONFLICT':
      return {
        ...base,
        kind: 'conflict',
        title: 'Conflict',
        message: err.message || 'conflict',
      };
    case 'LEDGER_DRIFT':
      return {
        ...base,
        kind: 'conflict',
        title: 'Ledger drift',
        message: err.message,
      };
    case 'LIMIT_EXCEEDED':
      return {
        ...base,
        kind: 'inline',
        title: 'Limit exceeded',
        message: err.message,
      };
    case 'RATE_LIMITED':
    case 'TOO_MANY_REQUESTS':
      return {
        ...base,
        kind: 'retry',
        title: 'Rate limited',
        message: retryAfterSec
          ? `Retry after ${retryAfterSec}s`
          : (err.message || 'too many requests'),
        retryAfterSec,
      };
    case 'NOT_IMPLEMENTED':
      return {
        ...base,
        kind: 'stub',
        title: 'Not implemented',
        message: err.message,
        stub: true,
      };
    case 'BILLING_UNAVAILABLE':
    case 'FORECAST_UNAVAILABLE':
    case 'CLICKHOUSE_UNAVAILABLE':
    case 'UNAVAILABLE':
    case 'SERVICE_UNAVAILABLE':
      return {
        ...base,
        kind: 'unavailable',
        title: 'Service unavailable',
        message: partial503Message(err) || err.message,
      };
    case 'INTERNAL':
    case 'INTERNAL_ERROR':
      return {
        ...base,
        kind: 'toast',
        title: 'Internal error',
        message: 'internal error',
      };
    default:
      if (err.status === 503) {
        return {
          ...base,
          kind: 'unavailable',
          title: 'Service unavailable',
          message: partial503Message(err) || err.message,
        };
      }
      return {
        ...base,
        kind: 'toast',
        title: 'Error',
        message: err.message || err.code,
      };
  }
}

/**
 * @param {ApiError} err
 * @returns {string|null}
 */
function partial503Message(err) {
  const errors = err.payload?.errors;
  if (Array.isArray(errors) && errors.length > 0) {
    return errors.join('; ');
  }
  return null;
}

/**
 * @param {string|null|undefined} header
 * @returns {number|undefined}
 */
function parseRetryAfter(header) {
  if (!header) return undefined;
  const n = Number.parseInt(header, 10);
  return Number.isFinite(n) && n > 0 ? n : undefined;
}

/**
 * @param {ServiceErrorView} view
 * @returns {boolean}
 */
export function isPageBlockingError(view) {
  return view.kind === 'page' || view.kind === 'unavailable';
}
