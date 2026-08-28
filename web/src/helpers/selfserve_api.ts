import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import { to } from '../lib/to.js';
import type { BillingStatement } from './customers_api.js';

function newIdempotencyKey(): string {
  return crypto.randomUUID();
}

export type BuyerPortfolioCampaignRow = {
  id: string;
  name: string;
  status: string;
  pacing_mode?: string;
  impressions_7d?: number;
  clicks_7d?: number;
  spend_micro?: number;
  budget_micro?: number;
  utilization_pct?: number;
  pacing_drift_pct?: number;
  estimated_pacing_drift_pct?: number;
  overspend_risk?: boolean;
  margin_breach?: boolean;
};

export type BuyerPortfolioKpis = {
  spend_micro?: number;
  revenue_micro?: number;
  profit_micro?: number;
  conversions?: number;
  cpa_micro?: number;
  roi_pct?: number;
  freshness?: {
    as_of?: string;
    stale?: boolean;
    ch_lag_seconds?: number;
  };
};

export type BuyerPortfolioResponse = {
  customer_id?: string;
  period?: { from?: string; to?: string; timezone?: string };
  active?: number;
  paused?: number;
  archived?: number;
  impressions_7d?: number;
  clicks_7d?: number;
  overspend_count?: number;
  kpis?: BuyerPortfolioKpis;
  campaigns?: BuyerPortfolioCampaignRow[];
};

export type SelfServeTemplate = {
  id: string;
  customer_id?: string;
  name: string;
  budget_limit?: string;
  pacing_mode?: string;
  daily_budget?: string;
  timezone?: string;
  freq_limit?: number;
  freq_window?: number;
  target_countries?: string[];
  brand_id?: string;
};

export type SelfServeTemplateListResponse = {
  items?: SelfServeTemplate[];
  total?: number;
};

export type SelfServeCreateCampaignRequest = {
  template_id: string;
  name: string;
  budget_limit_micro?: number;
  customer_id?: string;
};

export type SelfServeCreateCampaignResponse = {
  id: string;
};

export type SelfServePaymentIntentRequest = {
  amount_micro: number;
  currency?: string;
  customer_id?: string;
};

export type SelfServePaymentIntentResponse = {
  intent_id?: string;
  status?: string;
  checkout_url?: string;
  provider_ref?: string;
  deposit_address?: string;
  deposit_network?: string;
  deposit_qr_svg?: string;
};

export type SelfServeInvoice = {
  id?: string;
  customer_id?: string;
  billing_month?: string;
  total_micro?: number;
  status?: string;
  currency?: string;
};

export type SelfServeInvoiceListResponse = {
  invoices?: SelfServeInvoice[];
  total?: number;
};

export type SelfServeApiKeyCreateResponse = {
  id?: string;
  name?: string;
  raw_key?: string;
  scopes?: string[];
  expires_at?: string;
};

export type BuyerDashboardParams = {
  customer_id?: string;
  from?: string;
  to?: string;
};

export function buildBuyerDashboardUrl(params: BuyerDashboardParams): string {
  const qs = new URLSearchParams();
  if (params.customer_id) qs.set('customer_id', params.customer_id);
  if (params.from) qs.set('from', params.from);
  if (params.to) qs.set('to', params.to);
  const query = qs.toString();
  return query ? `/api/v1/dashboards/buyer?${query}` : '/api/v1/dashboards/buyer';
}

export function buildSelfServeTemplatesUrl(customerId?: string): string {
  const qs = new URLSearchParams({ limit: '50', offset: '0' });
  if (customerId) qs.set('customer_id', customerId);
  return `/api/v1/selfserve/templates?${qs.toString()}`;
}

export function buildSelfServeInvoicesUrl(limit: number, offset: number, customerId?: string): string {
  const qs = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  });
  if (customerId) qs.set('customer_id', customerId);
  return `/api/v1/selfserve/invoices?${qs.toString()}`;
}

export function buildSelfServeStatementUrl(month?: string, from?: string, to?: string): string {
  const qs = new URLSearchParams();
  if (month) qs.set('month', month);
  if (from) qs.set('from', from);
  if (to) qs.set('to', to);
  const query = qs.toString();
  return query
    ? `/api/v1/selfserve/billing/statement?${query}`
    : '/api/v1/selfserve/billing/statement';
}

export async function fetchBuyerPortfolio(
  params: BuyerDashboardParams,
  signal?: AbortSignal
): Promise<BuyerPortfolioResponse> {
  const [result, err] = await to(api<BuyerPortfolioResponse>(buildBuyerDashboardUrl(params), { signal }));
  if (err) throw err;
  return result?.data ?? {};
}

export async function pauseSelfServeCampaign(campaignId: string, reason = ''): Promise<void> {
  const result = await apiConfirmed(`/api/v1/selfserve/campaigns/${encodeURIComponent(campaignId)}/pause`, {
    method: 'POST',
    body: JSON.stringify({ reason }),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('pause failed');
  }
}

export async function resumeSelfServeCampaign(campaignId: string, reason = ''): Promise<void> {
  const result = await apiConfirmed(`/api/v1/selfserve/campaigns/${encodeURIComponent(campaignId)}/resume`, {
    method: 'POST',
    body: JSON.stringify({ reason }),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('resume failed');
  }
}

export async function fetchSelfServeTemplates(
  customerId?: string,
  signal?: AbortSignal
): Promise<SelfServeTemplateListResponse> {
  const [result, err] = await to(
    api<SelfServeTemplateListResponse>(buildSelfServeTemplatesUrl(customerId), { signal })
  );
  if (err) throw err;
  return result?.data ?? {};
}

export async function createSelfServeCampaign(
  body: SelfServeCreateCampaignRequest
): Promise<SelfServeCreateCampaignResponse> {
  const result = await apiConfirmed('/api/v1/selfserve/campaigns', {
    method: 'POST',
    headers: { 'Idempotency-Key': newIdempotencyKey() },
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('create campaign failed');
  }
  return (result.data ?? {}) as SelfServeCreateCampaignResponse;
}

export async function createSelfServePaymentIntent(
  body: SelfServePaymentIntentRequest
): Promise<SelfServePaymentIntentResponse> {
  const result = await apiConfirmed('/api/v1/selfserve/payment-intents', {
    method: 'POST',
    headers: { 'Idempotency-Key': newIdempotencyKey() },
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('payment intent failed');
  }
  return (result.data ?? {}) as SelfServePaymentIntentResponse;
}

export async function fetchSelfServeStatement(
  params: { month?: string; from?: string; to?: string },
  signal?: AbortSignal
): Promise<BillingStatement> {
  const [result, err] = await to(
    api<BillingStatement>(buildSelfServeStatementUrl(params.month, params.from, params.to), { signal })
  );
  if (err) throw err;
  return result?.data ?? {};
}

export async function fetchSelfServeInvoices(
  limit: number,
  offset: number,
  customerId?: string,
  signal?: AbortSignal
): Promise<SelfServeInvoiceListResponse> {
  const [result, err] = await to(
    api<SelfServeInvoiceListResponse>(buildSelfServeInvoicesUrl(limit, offset, customerId), { signal })
  );
  if (err) throw err;
  return result?.data ?? {};
}

export async function createSelfServeApiKey(
  name: string,
  scopes?: string[]
): Promise<SelfServeApiKeyCreateResponse> {
  const result = await apiConfirmed('/api/v1/selfserve/api-keys', {
    method: 'POST',
    body: JSON.stringify({ name, scopes: scopes ?? [] }),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('api key create failed');
  }
  return (result.data ?? {}) as SelfServeApiKeyCreateResponse;
}
