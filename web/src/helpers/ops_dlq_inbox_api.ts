import { to } from '../lib/to.js';
import { api, ApiError } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import { getOrCreate } from './idempotency.js';
import type { FanOutSourceError } from '../types/ops_extra.js';

export type DLQInboxSource = 'stream' | 'postback' | 'capi' | '';

export type DLQInboxEntryDTO = {
  id: string;
  source: string;
  campaign_id?: string;
  event_type?: string;
  error?: string;
  failed_at?: string;
  status?: string;
  retry_count?: number;
  shard_id?: number;
  stream_id?: string;
  entry_id?: string;
  click_id?: string;
  provider?: string;
};

export type DLQInboxListResponse = {
  items?: DLQInboxEntryDTO[];
  next_cursor?: string;
  partial?: boolean;
  errors?: FanOutSourceError[];
};

export type DlqInboxFetchResult = {
  items: DLQInboxEntryDTO[];
  nextCursor: string;
  partialErrors: FanOutSourceError[];
  error?: unknown;
};

export function isDlqInboxEntryRetryable(entry: DLQInboxEntryDTO): boolean {
  const status = typeof entry.status === 'string' ? entry.status.trim().toUpperCase() : '';
  return status !== 'RETRIED' && status !== 'SUCCESS' && status !== 'SUCCEEDED';
}

export function buildDlqInboxListUrl(source: DLQInboxSource = '', cursor = '', limit = 50): string {
  const params = new URLSearchParams({ limit: String(limit) });
  if (source) params.set('source', source);
  if (cursor) params.set('cursor', cursor);
  return `/api/v1/ops/dlq/inbox?${params.toString()}`;
}

export async function fetchDlqInboxPage(
  source: DLQInboxSource = '',
  cursor = '',
  limit = 50
): Promise<DlqInboxFetchResult> {
  const [res, err] = await to(
    api<DLQInboxListResponse>(buildDlqInboxListUrl(source, cursor, limit))
  );
  if (err) {
    if (err instanceof ApiError && err.status === 503 && err.payload) {
      const payload = err.payload as DLQInboxListResponse;
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

export async function retryDlqInboxEntry(entry: DLQInboxEntryDTO): Promise<void> {
  const scope = `ops-dlq-inbox-retry:${entry.source}:${entry.id}`;
  await apiConfirmed(`/api/v1/ops/dlq/inbox/${encodeURIComponent(entry.id)}/retry`, {
    method: 'POST',
    body: JSON.stringify({ source: entry.source }),
    headers: { 'Idempotency-Key': getOrCreate(scope) },
    idempotencyScope: scope,
  });
}
