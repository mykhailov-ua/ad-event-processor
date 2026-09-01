import { apiFetch, apiJson } from './client.js';
import type {
  APIKeyCreatedResponse,
  BillingStatement,
  CreateAPIKeyRequest,
  CreatePaymentIntentRequest,
  PaymentIntentCreatedResponse,
  SelfServeCreateCampaignRequest,
  SelfServeInvoiceListResponse,
  SelfServeInvoicesQuery,
  SelfServePauseCampaignRequest,
  SelfServeTemplateListResponse,
} from './types.js';
import type { components } from '../types/generated/openapi.js';

type IDCreatedResponse = components['schemas']['IDCreatedResponse'];

function newIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random()}`;
}

export function buildSelfServeTemplatesPath(customerId: string): string {
  const search = new URLSearchParams();
  search.set('customer_id', customerId);
  return `/api/v1/selfserve/templates?${search.toString()}`;
}

export async function listSelfServeTemplates(
  customerId: string,
  signal?: AbortSignal,
): Promise<SelfServeTemplateListResponse> {
  return apiJson<SelfServeTemplateListResponse>(buildSelfServeTemplatesPath(customerId), {
    signal,
  });
}

export async function createSelfServeCampaign(
  body: SelfServeCreateCampaignRequest,
  signal?: AbortSignal,
): Promise<IDCreatedResponse> {
  return apiJson<IDCreatedResponse>('/api/v1/selfserve/campaigns', {
    method: 'POST',
    headers: {
      'Idempotency-Key': newIdempotencyKey(),
    },
    body: JSON.stringify(body),
    signal,
  });
}

export async function listSelfServeInvoices(
  params: SelfServeInvoicesQuery = {},
  signal?: AbortSignal,
): Promise<SelfServeInvoiceListResponse> {
  const search = new URLSearchParams();
  if (params.customer_id) {
    search.set('customer_id', params.customer_id);
  }
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }
  const query = search.toString();
  const path = query ? `/api/v1/selfserve/invoices?${query}` : '/api/v1/selfserve/invoices';
  return apiJson<SelfServeInvoiceListResponse>(path, { signal });
}

export async function getSelfServeBillingStatement(
  params: { customer_id?: string; month?: string } = {},
  signal?: AbortSignal,
): Promise<BillingStatement> {
  const search = new URLSearchParams();
  if (params.customer_id) {
    search.set('customer_id', params.customer_id);
  }
  if (params.month) {
    search.set('month', params.month);
  }
  const query = search.toString();
  const path = query
    ? `/api/v1/selfserve/billing/statement?${query}`
    : '/api/v1/selfserve/billing/statement';
  return apiJson<BillingStatement>(path, { signal });
}

export async function createSelfServePaymentIntent(
  body: CreatePaymentIntentRequest,
  signal?: AbortSignal,
): Promise<PaymentIntentCreatedResponse> {
  return apiJson<PaymentIntentCreatedResponse>('/api/v1/selfserve/payment-intents', {
    method: 'POST',
    headers: {
      'Idempotency-Key': newIdempotencyKey(),
    },
    body: JSON.stringify(body),
    signal,
  });
}

export async function createSelfServeApiKey(
  body: CreateAPIKeyRequest,
  signal?: AbortSignal,
): Promise<APIKeyCreatedResponse> {
  return apiJson<APIKeyCreatedResponse>('/api/v1/selfserve/api-keys', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function pauseSelfServeCampaign(
  campaignId: string,
  body: SelfServePauseCampaignRequest = {},
  signal?: AbortSignal,
): Promise<void> {
  const response = await apiFetch(
    `/api/v1/selfserve/campaigns/${encodeURIComponent(campaignId)}/pause`,
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    },
  );
  if (!response.ok && response.status !== 202) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
}

export async function resumeSelfServeCampaign(
  campaignId: string,
  body: SelfServePauseCampaignRequest = {},
  signal?: AbortSignal,
): Promise<void> {
  const response = await apiFetch(
    `/api/v1/selfserve/campaigns/${encodeURIComponent(campaignId)}/resume`,
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    },
  );
  if (!response.ok && response.status !== 202) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
}
