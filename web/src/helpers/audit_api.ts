import { api } from './api_client.js';

export type AuditLog = {
  id?: number;
  admin_id?: string;
  action?: string;
  target_type?: string;
  target_id?: string;
  created_at?: string;
};

export type AuditListParams = {
  limit: number;
  offset: number;
  redact_pii: boolean;
};

export type AuditListResult = {
  items: AuditLog[];
  total: number;
};

export function buildAuditListUrl(params: AuditListParams): string {
  const qs = new URLSearchParams({
    limit: String(params.limit),
    offset: String(params.offset),
    redact_pii: params.redact_pii ? 'true' : 'false',
  });
  return `/api/v1/audit?${qs.toString()}`;
}

export async function listAudit(
  params: AuditListParams,
  signal?: AbortSignal
): Promise<AuditListResult> {
  const result = await api<AuditLog[]>(buildAuditListUrl(params), { signal });
  const items = Array.isArray(result.data) ? result.data : [];
  const totalHeader = result.headers.get('X-Total-Count');
  const parsed = totalHeader ? Number.parseInt(totalHeader, 10) : NaN;
  const total = Number.isFinite(parsed) ? parsed : items.length;
  return { items, total };
}

export function buildAuditExportUrl(params: {
  redact_pii: boolean;
  cursor?: string;
}): string {
  const qs = new URLSearchParams({
    format: 'csv',
    redact_pii: params.redact_pii ? 'true' : 'false',
  });
  if (params.cursor) qs.set('cursor', params.cursor);
  return `/api/v1/audit/export?${qs.toString()}`;
}
