import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import { getOrCreate } from './idempotency.js';
import type { DLQListResponse } from '../types/api/ops_extra.js';

/**
 * Build the ops DLQ list URL with optional cursor pagination.
 */
export function buildDlqListUrl(cursor = '', limit = 50): string {
  const params = new URLSearchParams({ limit: String(limit) });
  if (cursor) params.set('cursor', cursor);
  return `/api/v1/ops/dlq?${params.toString()}`;
}

/**
 * List stream DLQ entries from all Redis shards.
 */
export async function fetchOpsDlq(cursor = '', limit = 50): Promise<DLQListResponse> {
  const res = await api<DLQListResponse>(buildDlqListUrl(cursor, limit));
  return res.data ?? { items: [] };
}

/**
 * Retry a failed stream DLQ entry (enqueue only).
 */
export async function retryOpsDlq(id: string): Promise<void> {
  const scope = `ops-dlq-retry:${id}`;
  await apiConfirmed(`/api/v1/ops/dlq/${encodeURIComponent(id)}/retry`, {
    method: 'POST',
    body: '{}',
    headers: { 'Idempotency-Key': getOrCreate(scope) },
    idempotencyScope: scope,
  });
}
