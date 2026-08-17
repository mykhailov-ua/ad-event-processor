import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import { getOrCreate, clearScope } from './idempotency.js';

export type SelfServeTemplate = {
  id: string;
  name: string;
  budget_limit: string;
  pacing_mode: string;
};

export type SelfServeInvoice = {
  id: string;
  status?: string;
  amount_micro?: number;
  created_at?: string;
};

/**
 * List campaign templates available for self-serve create.
 */
export async function fetchSelfServeTemplates(): Promise<SelfServeTemplate[]> {
  const { data } = await api<{ items?: SelfServeTemplate[] }>('/api/v1/selfserve/templates');
  return data?.items ?? [];
}

/**
 * List self-serve invoices for the session customer.
 */
export async function fetchSelfServeInvoices(): Promise<SelfServeInvoice[]> {
  const { data } = await api<{ invoices?: SelfServeInvoice[] }>('/api/v1/selfserve/invoices');
  return data?.invoices ?? [];
}

/**
 * Create a campaign from a template via self-serve API.
 */
export async function createSelfServeCampaign(body: {
  template_id: string;
  name?: string;
  budget_limit_micro?: number;
}): Promise<{ id: string }> {
  const scope = `selfserve-create:${body.template_id}`;
  const res = await apiConfirmed<{ id: string }>('/api/v1/selfserve/campaigns', {
    method: 'POST',
    headers: { 'Idempotency-Key': getOrCreate(scope) },
    body: JSON.stringify(body),
    idempotencyScope: scope,
  });
  clearScope(scope);
  return res.data as { id: string };
}
