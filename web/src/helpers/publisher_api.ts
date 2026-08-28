import { api } from './api_client.js';
import { to } from '../lib/to.js';

export type PublisherKpis = {
  impressions?: number;
  fill_rate?: number;
  ecpm_micro?: number;
  ivt_rate?: number;
};

export type PublisherPlacementRow = {
  placement_id: string;
  impressions?: number;
  clicks?: number;
  fill_rate?: number;
  revenue_micro?: number;
  ecpm_micro?: number;
};

export type PublisherDashboard = {
  seller_id?: string;
  publisher_account_id?: string;
  from?: string;
  to?: string;
  kpis?: PublisherKpis;
  placements?: PublisherPlacementRow[];
};

export type PublisherStatementRow = {
  id?: number;
  amount_micro?: number;
  created_at?: string;
  campaign_id?: string;
  idempotency_hash?: string;
};

export type PublisherStatementListResponse = {
  items?: PublisherStatementRow[];
  total?: number;
};

export type PublisherRangeParams = {
  from: string;
  to: string;
};

export function buildPublisherDashboardUrl(params: PublisherRangeParams): string {
  const qs = new URLSearchParams({
    from: params.from,
    to: params.to,
  });
  return `/api/v1/publisher/dashboard?${qs.toString()}`;
}

export function buildPublisherStatementsUrl(
  params: PublisherRangeParams & { limit: number; offset: number }
): string {
  const qs = new URLSearchParams({
    from: params.from,
    to: params.to,
    limit: String(params.limit),
    offset: String(params.offset),
  });
  return `/api/v1/publisher/statements?${qs.toString()}`;
}

export async function fetchPublisherDashboard(
  params: PublisherRangeParams,
  signal?: AbortSignal
): Promise<PublisherDashboard> {
  const [result, err] = await to(api<PublisherDashboard>(buildPublisherDashboardUrl(params), { signal }));
  if (err) throw err;
  return result?.data ?? {};
}

export async function fetchPublisherStatements(
  params: PublisherRangeParams & { limit: number; offset: number },
  signal?: AbortSignal
): Promise<PublisherStatementListResponse> {
  const [result, err] = await to(
    api<PublisherStatementListResponse>(buildPublisherStatementsUrl(params), { signal })
  );
  if (err) throw err;
  return result?.data ?? {};
}
