import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';
import { formatDashboardCrPct, formatDashboardRoiPct } from '@/domains/dashboards/dashboard_format';
import {
  formatTableCount,
  formatTableMoneyFromMicro,
  parseAndFormatTableMoneyStr,
} from '@/domains/campaigns/list/campaign_list_format';
import {
  type CampaignFunnelCounts,
} from '@/domains/campaigns/list/campaign_list_funnel';
import {
  campaignListRowWithoutTraffic,
  resolveCampaignListRowMetrics,
} from '@/domains/campaigns/list/campaign_list_row_metrics';
import {
  campaignListRowClass,
  campaignListRowStatusEdgeClass,
  campaignStatusBadgeClass,
} from '@/domains/campaigns/list/campaign_list_row_tone';
import { profitToneClass, roiToneClass, resolveIndicatorTone } from '@/domains/campaigns/list/campaign_list_tone';
import { campaignDisplayId } from '@/domains/campaigns/list/campaign_display_id';
import { parseCampaignListName } from '@/domains/campaigns/list/campaign_list_name';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';
import { formatCampaignStatusLabel } from '@/lib/admin_typography';
import { resolveCustomerLabel } from '@/lib/customer_label';

export type VmCell = { text: string; isZero: boolean };

export type VmRateCell = { text: string; valPct: number; isZero: boolean };

export type CampaignRowVm = {
  id: string;
  displayId: string;
  nameParts: { title: string; meta: string[] };
  rawName: string;

  statusLabel: string;
  statusBadgeClass: string;

  rowClass: string;
  rowStatusEdgeClass: string;
  hasTraffic: boolean;

  indicatorTone: 'positive' | 'negative' | 'neutral';
  indicatorSymbol: '+' | '-' | '0';
  indicatorTitle: string;

  funnel: CampaignFunnelCounts;
  clicks: VmCell;
  impressions: VmCell;
  uniqueClicks: VmCell;
  lpClicks: VmCell;
  lpViews: VmCell;
  leads: VmCell;
  approved: VmCell;
  holdLeads: VmCell;
  rejectedLeads: VmCell;
  blocks: VmCell;
  bots: VmCell;

  ctr: VmRateCell | null;
  lpCtr: VmRateCell | null;
  cr: VmRateCell;
  approveRate: VmRateCell | null;
  blockPct: string | null;
  botPct: string | null;
  cpm: string | null;

  revenue: VmCell;
  cost: VmCell;
  profit: VmCell;
  profitToneClass: string;
  roi: VmRateCell;
  roiToneClass: string;
  epc: VmCell;
  cpc: VmCell;
  cpa: VmCell;
  ecpa: VmCell;
  budgetPct: number | null;

  groupLabel: string | null;
  groupCustomerId: string;
  flowId: string | null;
  ownerLabel: string;
  ownerId: string;
  countries: string[];
};

function vmOptionalRate(pct?: number | null): VmRateCell | null {
  if (pct == null || !Number.isFinite(pct)) {
    return null;
  }
  return { text: formatDashboardCrPct(pct), valPct: pct, isZero: pct === 0 };
}

function vmRateOrZero(pct?: number | null): VmRateCell {
  const val = pct ?? 0;
  return { text: formatDashboardCrPct(val), valPct: val, isZero: val === 0 };
}

function vmRoi(pct?: number | null): VmRateCell {
  if (pct == null || !Number.isFinite(pct)) {
    return { text: '-', valPct: 0, isZero: true };
  }
  return { text: formatDashboardRoiPct(pct), valPct: pct, isZero: pct === 0 };
}

function optionalRateLabel(pct?: number | null): string | null {
  if (pct == null || !Number.isFinite(pct) || pct <= 0) {
    return null;
  }
  return formatDashboardCrPct(pct);
}

export function buildCampaignRowVm(
  campaign: Campaign,
  metrics: CampaignListMetrics | undefined,
  margin: CampaignMargin | undefined,
  customerNameById: Record<string, string>,
  ownerEmailById: Record<string, string>,
  selected: boolean,
  highlightActiveRows: boolean,
): CampaignRowVm {
  const row = campaign as CampaignWithMoneyDisplay;
  const { clicks, impressions, blocks, costMicro, profitMicro, revenueMicro, funnel } =
    resolveCampaignListRowMetrics(metrics, margin);

  const displayId = campaignDisplayId(campaign);
  const nameParts = parseCampaignListName(campaign.name);

  const statusLabel = formatCampaignStatusLabel(campaign.status, row.status_label);
  const statusBadgeClass = campaignStatusBadgeClass(campaign.status, row.status_tone);

  const rowClass = campaignListRowClass({
    status: campaign.status,
    statusTone: row.status_tone,
    selected,
    highlightActiveRows,
    margin,
  });
  const rowStatusEdgeClass = campaignListRowStatusEdgeClass(campaign.status, row.status_tone);
  const hasTraffic = !campaignListRowWithoutTraffic({
    clicks,
    impressions,
    blocks,
    costMicro,
    profitMicro,
    revenueMicro,
    funnel,
  });

  const indicatorTone = resolveIndicatorTone(margin);
  const indicatorSymbol: '+' | '-' | '0' =
    indicatorTone === 'positive' ? '+' : indicatorTone === 'negative' ? '-' : '0';

  let indicatorTitle = 'Profit = 0.00';
  if (indicatorTone === 'positive') {
    indicatorTitle = `Profitable (+${formatTableMoneyFromMicro(profitMicro).text})`;
  } else if (indicatorTone === 'negative') {
    indicatorTitle =
      profitMicro !== 0
        ? `Unprofitable (${formatTableMoneyFromMicro(profitMicro).text})`
        : 'Unprofitable (ROI -100%)';
  }

  // KPI rates and per-click money come from CampaignListMetricsRow only (BatchCampaignListMetrics).
  const ctr = vmOptionalRate(metrics?.ctr_pct);
  const lpCtr = vmOptionalRate(metrics?.lp_ctr_pct);
  const cr = vmRateOrZero(metrics?.cr_pct);
  const approveRate = vmOptionalRate(metrics?.approve_rate_pct);
  // Omit zero block/bot share labels even when the server sends 0 pct.
  const blockPct = optionalRateLabel(metrics?.block_pct);
  const botPct = optionalRateLabel(metrics?.bot_pct);
  // Server omits cpm_usd when cost or impressions are zero; "0.00" is treated as absent.
  const cpm = metrics?.cpm_usd && metrics.cpm_usd !== '0.00' ? metrics.cpm_usd : null;

  // Window margin micros can be zero before metrics load; fall back to campaign lifetime spend labels.
  const revenue =
    revenueMicro > 0
      ? formatTableMoneyFromMicro(revenueMicro)
      : parseAndFormatTableMoneyStr(row.current_spend_display ?? row.current_spend);

  const cost =
    costMicro > 0
      ? formatTableMoneyFromMicro(costMicro)
      : parseAndFormatTableMoneyStr(row.current_spend_display ?? row.current_spend);

  const profitRes = formatTableMoneyFromMicro(profitMicro);
  const roiRes = vmRoi(metrics?.roi_pct);

  const budgetPct = campaign.budget_used_pct ?? null;

  const groupLabel = resolveCustomerLabel(campaign.customer_id, customerNameById) || null;
  const ownerId = campaign.owner_user_id ?? '';
  const ownerLabel = ownerId ? (ownerEmailById[ownerId] ?? ownerId.slice(0, 8)) : '-';

  return {
    id: campaign.id,
    displayId,
    nameParts,
    rawName: campaign.name,

    statusLabel,
    statusBadgeClass,

    rowClass,
    rowStatusEdgeClass,
    hasTraffic,

    indicatorTone,
    indicatorSymbol,
    indicatorTitle,

    funnel,
    clicks: formatTableCount(clicks),
    impressions: formatTableCount(impressions),
    uniqueClicks: formatTableCount(metrics?.unique_clicks),
    lpClicks: formatTableCount(funnel.lpClicks),
    lpViews: formatTableCount(funnel.lpViews),
    leads: formatTableCount(funnel.rawLeads),
    approved: formatTableCount(funnel.approved),
    holdLeads: formatTableCount(funnel.hold),
    rejectedLeads: formatTableCount(funnel.rejected),
    blocks: formatTableCount(blocks),
    bots: formatTableCount(funnel.bots),

    ctr,
    lpCtr,
    cr,
    approveRate,
    blockPct,
    botPct,
    cpm,

    revenue,
    cost,
    profit: profitRes,
    profitToneClass: profitToneClass(margin),
    roi: roiRes,
    roiToneClass: roiToneClass(margin),
    epc: formatTableMoneyFromMicro(metrics?.epc_micro),
    cpc: formatTableMoneyFromMicro(metrics?.cpc_micro),
    cpa: formatTableMoneyFromMicro(metrics?.cpa_micro),
    ecpa: formatTableMoneyFromMicro(metrics?.ecpa_micro),
    budgetPct,

    groupLabel,
    groupCustomerId: campaign.customer_id,
    flowId: campaign.flow_id ?? null,
    ownerLabel,
    ownerId,
    countries: campaign.target_countries ?? [],
  };
}
