import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import { getOrCreate } from './idempotency.js';
import type { components } from '../types/generated/openapi.js';
import type { BillingStatementDTO } from '../types/billing.js';

export type PaymentIntentResult = components['schemas']['PaymentIntentCreatedResponse'];

export type { BillingStatementDTO };

/**
 * Create a customer top-up payment intent.
 */
export async function createPaymentIntent(
  amountMicro: number,
  customerId?: string,
  currency = 'USD'
): Promise<PaymentIntentResult> {
  const scope = `payment-intent:${customerId ?? 'session'}:${amountMicro}`;
  const body: components['schemas']['CreatePaymentIntentRequest'] = {
    amount_micro: amountMicro,
    currency,
  };
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
 * Fetch self-serve billing statement for the authenticated customer.
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

/**
 * Map payment intent status wire value to operator label.
 */
export function formatPaymentStatus(status: string): string {
  switch (status) {
    case 'PAYMENT_INTENT_STATUS_PENDING_PROVIDER':
      return 'Awaiting payment';
    case 'PAYMENT_INTENT_STATUS_PROCESSING':
      return 'Confirming on-chain';
    case 'PAYMENT_INTENT_STATUS_SUCCEEDED':
      return 'Paid';
    case 'PAYMENT_INTENT_STATUS_FAILED':
      return 'Failed';
    default:
      return status
        .replace(/^PAYMENT_INTENT_STATUS_/, '')
        .replaceAll('_', ' ')
        .toLowerCase();
  }
}
