import { ApiError } from '@/api/api_error';
import { userErrorMessage } from '@/lib/admin_error';

const EXPORT_TIMEOUT_MESSAGE =
  'The export request timed out. The job may still be running on the server — poll the job ID or narrow the date range and try again.';
const EXPORT_SOURCE_MESSAGE =
  'The report data source failed. Try a smaller date range, lower the row limit, or contact support if this persists.';
const EXPORT_SERVER_MESSAGE =
  'The server could not complete the export. Try again later or contact support.';
const EXPORT_VALIDATION_FALLBACK = 'Check the form fields and try again.';

export function exportHubErrorMessage(error: unknown, fallback = EXPORT_VALIDATION_FALLBACK): string {
  if (error instanceof ApiError) {
    if (error.code === 'TIMEOUT' || (error.status === 0 && error.code === 'TIMEOUT')) {
      return EXPORT_TIMEOUT_MESSAGE;
    }
    if (error.status >= 500) {
      return EXPORT_SERVER_MESSAGE;
    }
    const lower = error.message.toLowerCase();
    if (
      lower.includes('clickhouse') ||
      lower.includes('db::') ||
      lower.includes('code: ') ||
      lower.includes('memory limit')
    ) {
      return EXPORT_SOURCE_MESSAGE;
    }
    if (lower.includes('timed out') || lower.includes('timeout') || lower.includes('deadline')) {
      return EXPORT_TIMEOUT_MESSAGE;
    }
    if (error.message.trim() !== '') {
      return error.message;
    }
    return fallback;
  }

  if (error instanceof Error) {
    const lower = error.message.toLowerCase();
    if (lower.includes('timed out') || lower.includes('timeout') || lower.includes('deadline')) {
      return EXPORT_TIMEOUT_MESSAGE;
    }
    if (
      lower.includes('clickhouse') ||
      lower.includes('db::') ||
      lower.includes('code: ') ||
      lower.includes('memory limit')
    ) {
      return EXPORT_SOURCE_MESSAGE;
    }
    if (error.message.trim() !== '') {
      return error.message;
    }
  }

  return userErrorMessage(error, fallback);
}

export function exportHubJobErrorMessage(stored: string | undefined | null): string | undefined {
  const trimmed = (stored ?? '').trim();
  if (!trimmed) {
    return undefined;
  }
  return exportHubErrorMessage(new Error(trimmed), trimmed);
}
