import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import { getOrCreate } from './idempotency.js';

export type PaymentIntentResult = {
  intent_id: string;
  status: string;
  checkout_url: string;
  provider_ref?: string;
};

export type BillingStatementDTO = {
  customer_id?: string;
  period?: { from?: string; to?: string };
  opening_balance_micro?: number;
  closing_balance_micro?: number;
  currency?: string;
  [key: string]: unknown;
};

/**
 * Create a self-serve wallet top-up payment intent.
 */
export async function createPaymentIntent(
  amountMicro: number,
  customerId?: string,
  currency = 'USD',
): Promise<PaymentIntentResult> {
  const scope = `payment-intent:${customerId ?? 'session'}:${amountMicro}`;
  const body: Record<string, unknown> = { amount_micro: amountMicro, currency };
  if (customerId) body.customer_id = customerId;
  const res = await apiConfirmed<PaymentIntentResult>('/api/v1/selfserve/payment-intents', {
    method: 'POST',
    body: JSON.stringify(body),
    headers: { 'Idempotency-Key': getOrCreate(scope) },
    idempotencyScope: scope,
  });
  return res.data as PaymentIntentResult;
}

/**
 * Fetch self-serve billing statement for the current month window.
 */
export async function fetchSelfServeStatement(month = ''): Promise<BillingStatementDTO> {
  const params = new URLSearchParams();
  if (month) params.set('month', month);
  const qs = params.toString();
  const path = qs
    ? `/api/v1/selfserve/billing/statement?${qs}`
    : '/api/v1/selfserve/billing/statement';
  const res = await api<BillingStatementDTO>(path);
  return res.data ?? {};
}
