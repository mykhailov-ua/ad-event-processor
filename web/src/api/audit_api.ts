import { ApiError, apiFetch } from './client.js';
import type {
  AuditExportQuery,
  AuditListQuery,
  AuditListResult,
  AuditLog,
} from './types.js';

export type AuditExportResult = {
  blob: Blob;
  truncated: boolean;
  nextCursor?: string;
};

export function buildAuditListPath(params: AuditListQuery = {}): string {
  const search = new URLSearchParams();

  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }
  if (params.redact_pii != null) {
    search.set('redact_pii', String(params.redact_pii));
  }

  const query = search.toString();
  return query ? `/api/v1/audit?${query}` : '/api/v1/audit';
}

async function parseApiError(response: Response): Promise<ApiError> {
  let code = 'HTTP_ERROR';
  let message = response.statusText || `HTTP ${response.status}`;

  try {
    const body: unknown = await response.json();
    if (body && typeof body === 'object') {
      const record = body as Record<string, unknown>;
      const errorField = record.error;
      if (errorField && typeof errorField === 'object') {
        const errObj = errorField as Record<string, unknown>;
        if (typeof errObj.code === 'string') {
          code = errObj.code;
        }
        if (typeof errObj.message === 'string') {
          message = errObj.message;
        }
      } else if (typeof errorField === 'string') {
        message = errorField;
      }
    }
  } catch {
    // Non-JSON error body; keep status text.
  }

  return new ApiError(response.status, code, message);
}

export async function listAudit(
  params: AuditListQuery = {},
  signal?: AbortSignal,
): Promise<AuditListResult> {
  const response = await apiFetch(buildAuditListPath(params), { signal });

  if (!response.ok) {
    throw await parseApiError(response);
  }

  const items = (await response.json()) as AuditLog[];
  const totalHeader = response.headers.get('X-Total-Count');
  const parsedTotal = totalHeader != null ? Number.parseInt(totalHeader, 10) : items.length;
  const total = Number.isFinite(parsedTotal) ? parsedTotal : items.length;

  return { items, total };
}

export function buildAuditExportPath(params: AuditExportQuery = { format: 'csv' }): string {
  const search = new URLSearchParams({ format: params.format ?? 'csv' });

  if (params.customer_id) {
    search.set('customer_id', params.customer_id);
  }
  if (params.cursor) {
    search.set('cursor', params.cursor);
  }
  if (params.redact_pii != null) {
    search.set('redact_pii', String(params.redact_pii));
  }

  return `/api/v1/audit/export?${search.toString()}`;
}

export async function exportAuditCsv(
  params: AuditExportQuery = { format: 'csv' },
  signal?: AbortSignal,
): Promise<AuditExportResult> {
  const response = await apiFetch(buildAuditExportPath(params), { signal });

  if (!response.ok) {
    throw await parseApiError(response);
  }

  const truncated = response.headers.get('X-Export-Truncated') === 'true';
  const nextCursor = response.headers.get('X-Next-Cursor') ?? undefined;

  return {
    blob: await response.blob(),
    truncated,
    nextCursor: nextCursor || undefined,
  };
}
