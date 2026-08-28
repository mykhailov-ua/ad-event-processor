import { to } from '../lib/to.js';
import { api, ApiError } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import { getOrCreate } from './idempotency.js';
import type { DLQEntryDTO, DLQListResponse, FanOutSourceError } from '../types/ops_extra.js';

export type DlqFetchResult = {
  items: DLQEntryDTO[];
  nextCursor: string;
  partialErrors: FanOutSourceError[];
  error?: unknown;
};

export function buildDlqListUrl(cursor = '', limit = 50): string {
  const params = new URLSearchParams({ limit: String(limit) });
  if (cursor) params.set('cursor', cursor);
  return `/api/v1/ops/dlq?${params.toString()}`;
}

export function isOpsDlqEntryRetryable(entry: DLQEntryDTO): boolean {
  if (!entry.id) return false;
  const status = typeof entry.status === 'string' ? entry.status.trim().toUpperCase() : '';
  return status !== 'RETRIED' && status !== 'SUCCESS' && status !== 'SUCCEEDED';
}

export async function fetchOpsDlqPage(cursor = '', limit = 50): Promise<DlqFetchResult> {
  const [res, err] = await to(api<DLQListResponse>(buildDlqListUrl(cursor, limit)));
  if (err) {
    if (err instanceof ApiError && err.status === 503 && err.payload) {
      const payload = err.payload as DLQListResponse;
      return {
        items: payload.items ?? [],
        nextCursor: payload.next_cursor ?? '',
        partialErrors: payload.errors ?? [],
      };
    }
    return { items: [], nextCursor: '', partialErrors: [], error: err };
  }
  const data = res?.data ?? {};
  return {
    items: data.items ?? [],
    nextCursor: data.next_cursor ?? '',
    partialErrors: data.errors ?? [],
  };
}

export async function fetchOpsDlq(cursor = '', limit = 50): Promise<DLQListResponse> {
  const page = await fetchOpsDlqPage(cursor, limit);
  if (page.error) throw page.error;
  return {
    items: page.items,
    next_cursor: page.nextCursor,
    errors: page.partialErrors.length > 0 ? page.partialErrors : undefined,
  };
}

export async function retryOpsDlq(id: string): Promise<void> {
  const scope = `ops-dlq-retry:${id}`;
  await apiConfirmed(`/api/v1/ops/dlq/${encodeURIComponent(id)}/retry`, {
    method: 'POST',
    body: '{}',
    headers: { 'Idempotency-Key': getOrCreate(scope) },
    idempotencyScope: scope,
  });
}
