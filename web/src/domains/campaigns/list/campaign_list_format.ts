import {
  formatDashboardCrPct,
  formatDashboardRoiPct,
} from '@/domains/dashboards/dashboard_format';
import { displayCount } from '@/lib/display';

import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { CampaignMargin } from '@/api/types';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';
import { resolveCampaignListRowMetrics } from '@/domains/campaigns/list/campaign_list_row_metrics';

const MICRO_PER_USD = 1_000_000;

export function formatTableMoneyNumber(amount?: number | null): { text: string; isZero: boolean } {
  if (amount == null || !Number.isFinite(amount) || amount === 0) {
    return { text: '0.00', isZero: true };
  }
  const formatted = new Intl.NumberFormat('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount);
  return { text: formatted, isZero: false };
}

export function formatTableMoneyFromMicro(micro?: number | null): { text: string; valUsd: number; isZero: boolean } {
  if (micro == null || !Number.isFinite(micro) || micro === 0) {
    return { text: '0.00', valUsd: 0, isZero: true };
  }
  const valUsd = micro / MICRO_PER_USD;
  const res = formatTableMoneyNumber(valUsd);
  return { text: res.text, valUsd, isZero: res.isZero };
}

export function parseAndFormatTableMoneyStr(raw?: string | null): { text: string; valUsd: number; isZero: boolean } {
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

export function formatTableCount(val?: number | null): { text: string; val: number; isZero: boolean } {
  if (val == null || val === 0) {
    return { text: '0', val: 0, isZero: true };
  }
  return { text: displayCount(val), val, isZero: false };
}

export function formatTableCr(clicks?: number, conversions?: number): { text: string; valPct: number; isZero: boolean } {
  if (clicks == null || clicks <= 0 || conversions == null || conversions <= 0) {
    return { text: '0.00%', valPct: 0, isZero: true };
  }
  const pct = (conversions / clicks) * 100;
  return { text: formatDashboardCrPct(pct), valPct: pct, isZero: false };
}

export function formatTableRoi(profitMicro?: number, costMicro?: number): { text: string; valPct: number; isZero: boolean } {
  if (costMicro == null || costMicro <= 0 || profitMicro == null) {
    return { text: '-', valPct: 0, isZero: true };
  }
  const pct = (profitMicro / costMicro) * 100;
  return { text: formatDashboardRoiPct(pct), valPct: pct, isZero: pct === 0 };
}

export function formatCampaignListCr(clicks?: number, conversions?: number): string {
  return formatTableCr(clicks, conversions).text;
}

export function formatCampaignListRoi(profitMicro?: number, costMicro?: number): string {
  return formatTableRoi(profitMicro, costMicro).text;
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

export function sumCampaignListTotals(
  items: CampaignWithMoneyDisplay[],
  metricsById: Record<string, CampaignListMetrics>,
  marginsById: Record<string, CampaignMargin>,
): CampaignListTotals {
  const totals = emptyCampaignListTotals();
  for (const campaign of items) {
    if (campaign.flow_id) {
      totals.flows += 1;
    }
    const metrics = metricsById[campaign.id];
    const margin = marginsById[campaign.id];
    const row = resolveCampaignListRowMetrics(metrics, margin);
    totals.clicks += row.clicks;
    totals.impressions += row.impressions;
    totals.blocks += row.blocks;
    totals.conversions += metrics?.conversions ?? 0;
    totals.revenueMicro += row.revenueMicro;
    totals.costMicro += row.costMicro;
    totals.profitMicro += row.profitMicro;
  }
  return totals;
}
