/** Edge fraud tier score bands (match deploy/nginx/lua/edge-fraud-tier.lua). */
export const FRAUD_TIER_PASS_MAX = 30;
export const FRAUD_TIER_SUSPECT_MAX = 60;
export const FRAUD_TIER_IVT_MAX = 80;

export type FraudTier = 'pass' | 'suspect' | 'ivt' | 'block';

export type FraudTierResult = {
  tier: FraudTier;
  score: number;
};

export type FraudTierBandRow = {
  tier: FraudTier;
  range: string;
  action: string;
};

/**
 * Map a fraud score 0–100 to an edge tier label.
 */
export function fraudTierFromScore(score: number): FraudTierResult {
  let n = Number(score);
  if (!Number.isFinite(n) || n < 0) n = 0;
  if (n > 100) n = 100;
  if (n <= FRAUD_TIER_PASS_MAX) return { tier: 'pass', score: n };
  if (n <= FRAUD_TIER_SUSPECT_MAX) return { tier: 'suspect', score: n };
  if (n <= FRAUD_TIER_IVT_MAX) return { tier: 'ivt', score: n };
  return { tier: 'block', score: n };
}

/**
 * Return human-readable tier band descriptions for the edge panel.
 */
export function fraudTierBandRows(): FraudTierBandRow[] {
  return [
    { tier: 'pass', range: `0–${FRAUD_TIER_PASS_MAX}`, action: 'Allow' },
    { tier: 'suspect', range: `${FRAUD_TIER_PASS_MAX + 1}–${FRAUD_TIER_SUSPECT_MAX}`, action: 'Monitor' },
    { tier: 'ivt', range: `${FRAUD_TIER_SUSPECT_MAX + 1}–${FRAUD_TIER_IVT_MAX}`, action: 'Shadow / throttle' },
    { tier: 'block', range: `${FRAUD_TIER_IVT_MAX + 1}–100`, action: 'Block at edge' },
  ];
}
