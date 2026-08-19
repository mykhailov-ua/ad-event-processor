import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import { isValidFraudIPHash } from './fraud_api.js';

export type FraudDecisionThresholds = {
  scope?: string;
  pass_max: number;
  suspect_max: number;
  ivt_max: number;
  block_above: number;
};

export type FraudDecision = {
  ip_hash: string;
  campaign_id: string;
  window_start: string;
  evaluated_at: string;
  disclaimer: string;
  tier: string;
  score: number;
  ml_probability: number;
  adjusted_probability: number;
  residential_proxy: boolean;
  structural_fraud: boolean;
  fp_guard_applied: boolean;
  model_score?: number;
  model_name?: string;
  score_missing?: boolean;
  features: Record<string, number>;
  campaign_thresholds: FraudDecisionThresholds;
};


export async function fetchFraudDecision(
  customerId: string,
  params: { ip_hash: string; campaign_id?: string; hours?: number }
): Promise<FraudDecision | null> {
  const hash = params.ip_hash.trim().toLowerCase();
  if (!isValidFraudIPHash(hash)) {
    throw new Error('ip_hash must be 32 hex characters');
  }
  const qs = new URLSearchParams({
    customer_id: customerId,
    ip_hash: hash,
  });
  if (params.campaign_id?.trim()) {
    qs.set('campaign_id', params.campaign_id.trim());
  }
  if (params.hours != null && params.hours > 0) {
    qs.set('hours', String(params.hours));
  }
  const res = await api<FraudDecision>(`/api/v1/fraud/decisions?${qs.toString()}`);
  return res.data ?? null;
}


export function fraudDecisionTierLabel(tier: string): string {
  switch (tier) {
    case 'pass':
      return 'Pass';
    case 'suspect':
      return 'Suspect';
    case 'ivt':
      return 'IVT';
    case 'block':
      return 'Block';
    default:
      return tier || 'Unknown';
  }
}

export type FraudOverrideRequest = {
  campaign_id?: string;
  ip?: string;
  ip_hash?: string;
};


export async function postFraudOverride(
  customerId: string,
  body: FraudOverrideRequest
): Promise<void> {
  const qs = new URLSearchParams({ customer_id: customerId });
  await apiConfirmed(`/api/v1/fraud/overrides?${qs.toString()}`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}
