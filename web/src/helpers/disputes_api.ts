import { api } from './api_client.js';

export type DisputeRow = {
  intent_id?: string;
  customer_id?: string;
  amount_micro?: number;
  currency?: string;
  provider_dispute_id?: string;
  updated_at?: string;
  chargeback_ledger_entry_ids?: number[];
};

export type DisputeListResponse = {
  disputes?: DisputeRow[];
  total?: number;
};

export type DisputeListParams = {
  limit?: number;
  offset?: number;
  customer_id?: string;
};

export function buildDisputesListUrl(params: DisputeListParams): string {
  const qs = new URLSearchParams();
  if (params.limit != null) qs.set('limit', String(params.limit));
  if (params.offset != null) qs.set('offset', String(params.offset));
  if (params.customer_id) qs.set('customer_id', params.customer_id);
  const query = qs.toString();
  return query ? `/api/v1/disputes?${query}` : '/api/v1/disputes';
}

export async function fetchDisputes(
  params: DisputeListParams,
  signal?: AbortSignal
): Promise<DisputeListResponse> {
  const result = await api<DisputeListResponse>(buildDisputesListUrl(params), { signal });
  return result.data ?? { disputes: [], total: 0 };
}
