import { ApiError } from '../api/api_error.ts';
import { isAdminDevMode } from './admin_dev_mode.ts';

export type AdminErrorKind =
  | 'load'
  | 'render'
  | 'route'
  | 'not-found'
  | 'forbidden';

const KIND_USER_MESSAGE: Record<AdminErrorKind, string> = {
  load: 'This page could not be loaded. Try again or return to the home page.',
  render: 'This page failed to render. Try reloading or go back.',
  route: 'Navigation failed. The link may be invalid or the server returned an error.',
  'not-found': 'The page you requested does not exist in this console.',
  forbidden: 'You do not have permission to view this page.',
};

const KIND_TITLE: Record<AdminErrorKind, string> = {
  load: 'Could not load page',
  render: 'Page error',
  route: 'Navigation error',
  'not-found': 'Page not found',
  forbidden: 'Access denied',
};

export function adminErrorTitle(kind: AdminErrorKind): string {
  return KIND_TITLE[kind];
}

export function adminErrorUserMessage(kind: AdminErrorKind): string {
  return KIND_USER_MESSAGE[kind];
}

export function shouldShowAdminErrorDetails(): boolean {
  return isAdminDevMode();
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object';
}

function routeErrorStatus(error: unknown): number | undefined {
  if (!isRecord(error)) {
    return undefined;
  }
  if (typeof error.status === 'number') {
    return error.status;
  }
  if (typeof error.statusText === 'string' && typeof error.status === 'number') {
    return error.status;
  }
  return undefined;
}

function routeErrorStatusText(error: unknown): string | undefined {
  if (!isRecord(error)) {
    return undefined;
  }
  if (typeof error.statusText === 'string' && error.statusText.trim() !== '') {
    return error.statusText;
  }
  if (typeof error.data === 'string' && error.data.trim() !== '') {
    return error.data;
  }
  return undefined;
}

export function userErrorMessage(
  error: unknown,
  fallback = 'Something went wrong. Try again or return to the home page.',
): string {
  if (error instanceof ApiError) {
    if (error.status === 404) {
      return 'The requested resource was not found.';
    }
    if (error.status === 403) {
      return 'You do not have permission to view this resource.';
    }
    if (error.status === 401) {
      return 'Your session expired. Sign in again.';
    }
    if (error.status === 0 && error.code === 'TIMEOUT') {
      return 'The request timed out. Check your connection and try again.';
    }
    if (error.status >= 500) {
      return 'The server encountered an error. Try again later.';
    }
    if (error.message.trim() !== '') {
      return error.message;
    }
    return fallback;
  }

  const routeStatus = routeErrorStatus(error);
  if (routeStatus === 404) {
    return KIND_USER_MESSAGE['not-found'];
  }
  if (routeStatus === 403) {
    return KIND_USER_MESSAGE.forbidden;
  }
  if (routeStatus != null && routeStatus >= 500) {
    return 'The server encountered an error. Try again later.';
  }

  const routeText = routeErrorStatusText(error);
  if (routeText) {
    return routeText;
  }

  if (error instanceof Error && error.message.trim() !== '') {
    if (shouldShowAdminErrorDetails()) {
      return error.message;
    }
    return fallback;
  }

  if (typeof error === 'string' && error.trim() !== '') {
    if (shouldShowAdminErrorDetails()) {
      return error;
    }
    return fallback;
  }

  return fallback;
}

export function formatAdminErrorDetails(error: unknown, componentStack?: string): string {
  const lines: string[] = [];

  if (error instanceof ApiError) {
    lines.push(`name: ${error.name}`);
    lines.push(`status: ${error.status}`);
    lines.push(`code: ${error.code}`);
    lines.push(`message: ${error.message}`);
    if (error.stack) {
      lines.push('', 'stack:', error.stack);
    }
  } else if (error instanceof Error) {
    lines.push(`name: ${error.name}`);
    lines.push(`message: ${error.message}`);
    if (error.stack) {
      lines.push('', 'stack:', error.stack);
    }
  } else if (isRecord(error)) {
    const status = routeErrorStatus(error);
    if (status != null) {
      lines.push(`status: ${status}`);
    }
    const statusText = routeErrorStatusText(error);
    if (statusText) {
      lines.push(`statusText: ${statusText}`);
    }
    if (typeof error.message === 'string') {
      lines.push(`message: ${error.message}`);
    }
    try {
      lines.push('', 'payload:', JSON.stringify(error, null, 2));
    } catch {
      lines.push('', 'payload: [unserializable]');
    }
  } else if (error != null) {
    lines.push(String(error));
  }

  if (componentStack?.trim()) {
    lines.push('', 'component stack:', componentStack.trim());
  }

  if (typeof window !== 'undefined') {
    lines.push('', `location: ${window.location.href}`);
  }

  return lines.join('\n');
}

export function adminErrorKindFromUnknown(error: unknown): AdminErrorKind {
  if (routeErrorStatus(error) === 404) {
    return 'not-found';
  }
  if (routeErrorStatus(error) === 403) {
    return 'forbidden';
  }
  if (error instanceof ApiError) {
    if (error.status === 404) {
      return 'not-found';
    }
    if (error.status === 403) {
      return 'forbidden';
    }
    return 'load';
  }
  return 'route';
}
