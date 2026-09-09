import type { CampaignListMetricsTotalsResponse } from '@/api/campaigns_types';
import { displayCount } from '@/lib/display';
import type { Campaign } from '@/api/types';
import type { CampaignStatusTone } from '@/domains/campaigns/list/campaign_list_row_tone';

export type CampaignWithMoneyDisplay = Campaign & {
  budget_limit_display?: string;
  current_spend_display?: string;
  status_label?: string;
  status_tone?: CampaignStatusTone;
};

const MICRO_PER_USD = 1_000_000;

export function formatTableMoneyNumber(amount?: number | null): { text: string; isZero: boolean } {
  if (amount == null || !Number.isFinite(amount) || amount === 0) {
    return { text: '0.00', isZero: true };
  }
  const formatted = new Intl.NumberFormat('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(Math.abs(amount));
  const text = amount < 0 ? `-${formatted}` : formatted;
  return { text, isZero: false };
}

export function formatTableMoneyFromMicro(micro?: number | null): {
  text: string;
  valUsd: number;
  isZero: boolean;
} {
  if (micro == null || !Number.isFinite(micro) || micro === 0) {
    return { text: '0.00', valUsd: 0, isZero: true };
  }
  const valUsd = micro / MICRO_PER_USD;
  const res = formatTableMoneyNumber(valUsd);
  return { text: res.text, valUsd, isZero: res.isZero };
}

export function parseAndFormatTableMoneyStr(raw?: string | null): {
  text: string;
  valUsd: number;
  isZero: boolean;
} {
  if (!raw?.trim()) {
    return { text: '0.00', valUsd: 0, isZero: true };
  }
  const cleaned = raw.replace(/USD\s*|\$|,/gi, '').trim();
  const val = Number.parseFloat(cleaned);
  if (!Number.isFinite(val) || val === 0) {
    return { text: '0.00', valUsd: 0, isZero: true };
  }
  const res = formatTableMoneyNumber(val);
  return { text: res.text, valUsd: val, isZero: res.isZero };
}

export function formatTableCount(val?: number | null): {
  text: string;
  val: number;
  isZero: boolean;
} {
  if (val == null || val === 0) {
    return { text: '0', val: 0, isZero: true };
  }
  return { text: displayCount(val), val, isZero: false };
}

export function microQueryParamToUsdInput(microRaw: string): string {
  if (!microRaw.trim()) {
    return '';
  }
  const micro = Number.parseInt(microRaw, 10);
  if (!Number.isFinite(micro) || micro < 0) {
    return '';
  }
  return (micro / MICRO_PER_USD).toFixed(2);
}

export function usdInputToMicroQueryParam(usdRaw: string): number | undefined {
  if (!usdRaw.trim()) {
    return undefined;
  }
  const usd = Number.parseFloat(usdRaw.replace(/[$,]/g, ''));
  if (!Number.isFinite(usd) || usd < 0) {
    return undefined;
  }
  return Math.round(usd * MICRO_PER_USD);
}

export type CampaignListTotals = {
  flows: number;
  clicks: number;
  impressions: number;
  blocks: number;
  conversions: number;
  revenueMicro: number;
  costMicro: number;
  profitMicro: number;
};

export function emptyCampaignListTotals(): CampaignListTotals {
  return {
    flows: 0,
    clicks: 0,
    impressions: 0,
    blocks: 0,
    conversions: 0,
    revenueMicro: 0,
    costMicro: 0,
    profitMicro: 0,
  };
}

/** Maps GET /api/v1/campaigns/metrics-totals; economics derived on server via enrichCampaignListMetricsRowDerived. */
export function campaignListTotalsFromMetricsTotals(
  response: CampaignListMetricsTotalsResponse
): CampaignListTotals {
  const row = response.totals;
  return {
    flows: response.flow_count,
    clicks: row.clicks ?? 0,
    impressions: row.impressions ?? 0,
    blocks: row.blocks ?? 0,
    conversions: row.conversions ?? 0,
    revenueMicro: row.revenue_micro ?? 0,
    costMicro: row.cost_micro ?? 0,
    profitMicro: row.profit_micro ?? 0,
  };
}
