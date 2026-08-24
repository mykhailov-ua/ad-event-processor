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

export function fraudTierFromScore(score: number): FraudTierResult {
  let n = Number(score);
  if (!Number.isFinite(n) || n < 0) n = 0;
  if (n > 100) n = 100;
  if (n <= FRAUD_TIER_PASS_MAX) return { tier: 'pass', score: n };
  if (n <= FRAUD_TIER_SUSPECT_MAX) return { tier: 'suspect', score: n };
  if (n <= FRAUD_TIER_IVT_MAX) return { tier: 'ivt', score: n };
  return { tier: 'block', score: n };
}

export function fraudTierBandRows(): FraudTierBandRow[] {
  return fraudTierBandRowsFromThresholds(
    FRAUD_TIER_PASS_MAX,
    FRAUD_TIER_SUSPECT_MAX,
    FRAUD_TIER_IVT_MAX
  );
}

export function fraudTierBandRowsFromThresholds(
  passMax: number,
  suspectMax: number,
  ivtMax: number
): FraudTierBandRow[] {
  const pass = Math.max(0, Math.min(100, Math.round(passMax)));
  const suspect = Math.max(pass, Math.min(100, Math.round(suspectMax)));
  const ivt = Math.max(suspect, Math.min(100, Math.round(ivtMax)));
  return [
    { tier: 'pass', range: `0-${pass}`, action: 'Allow' },
    {
      tier: 'suspect',
      range: `${pass + 1}-${suspect}`,
      action: 'Monitor / boost',
    },
    {
      tier: 'ivt',
      range: `${suspect + 1}-${ivt}`,
      action: 'Ghost IVT (if enabled)',
    },
    { tier: 'block', range: `${ivt + 1}-100`, action: 'Block at edge' },
  ];
}
