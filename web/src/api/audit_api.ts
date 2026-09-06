import { apiFetch, apiJsonArrayWithTotalCountValidated, parseApiError } from './client.js';
import { parseAuditLogRow } from './validate.js';
import type { AuditExportQuery, AuditListQuery, AuditListResult, AuditLog } from './types.js';

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

export async function listAudit(
  params: AuditListQuery = {},
  signal?: AbortSignal
): Promise<AuditListResult> {
  const { items, total } = await apiJsonArrayWithTotalCountValidated(
    buildAuditListPath(params),
    { signal },
    parseAuditLogRow
  );
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
  signal?: AbortSignal
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
