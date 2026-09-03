import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';
import {
  formatTableCount,
  formatTableCr,
  formatTableMoneyFromMicro,
  formatTableRoi,
  parseAndFormatTableMoneyStr,
} from '@/domains/campaigns/list/campaign_list_format';
import {
  formatApproveRate,
  formatCpmUsd,
  formatLpCtr,
  formatRelativeRate,
  formatSourceCtr,
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
import { rateBenchmarkToneClass, percentRate } from '@/domains/campaigns/list/campaign_list_rate_tone';
import { profitToneClass, roiToneClass, resolveIndicatorTone } from '@/domains/campaigns/list/campaign_list_tone';
import { campaignDisplayId } from '@/domains/campaigns/list/campaign_display_id';
import { parseCampaignListName } from '@/domains/campaigns/list/campaign_list_name';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';
import { formatCampaignStatusLabel } from '@/lib/admin_typography';
import { resolveCustomerLabel } from '@/lib/customer_label';

/** Formatted metric cell value - mirrors the output of formatTableCount / formatTableMoneyFromMicro. */
export type VmCell = { text: string; isZero: boolean };

/** Formatted rate cell - carries the numeric percentage for benchmark tone coloring. */
export type VmRateCell = { text: string; valPct: number; isZero: boolean };

/**
 * Pre-computed, display-ready view model for one campaign table row.
 * Built once before render via buildCampaignRowVm(); all JSX cells read fields directly.
 * No math or formatting happens inside the render pass.
 */
export type CampaignRowVm = {
  // Identity
  id: string;
  displayId: string;
  nameParts: { title: string; meta: string[] };
  rawName: string;

  // Status
  statusLabel: string;
  statusBadgeClass: string;

  // Row decoration
  rowClass: string;
  rowStatusEdgeClass: string;
  hasTraffic: boolean;

  // Profit indicator (left edge column)
  indicatorTone: 'positive' | 'negative' | 'neutral';
  indicatorSymbol: '+' | '-' | '0';
  indicatorTitle: string;

  // Funnel
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

  // Rates
  ctr: VmRateCell | null;       // null = not computable
  lpCtr: VmRateCell | null;
  cr: VmRateCell;
  approveRate: VmRateCell | null;
  blockPct: string | null;
  botPct: string | null;
  cpm: string | null;

  // Money
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
  budgetPct: number | null;    // null = no budget set; value is raw percent

  // Descriptive
  groupLabel: string | null;   // customer name, null = no match
  groupCustomerId: string;
  flowId: string | null;
  ownerLabel: string;
  ownerId: string;
  countries: string[];
};

function isRowWithoutTraffic(row: ReturnType<typeof resolveCampaignListRowMetrics>): boolean {
  return campaignListRowWithoutTraffic(row);
}

/**
 * Builds a fully pre-computed row ViewModel from raw API data.
 * Call once per row (in useMemo or derived array) before the render pass.
 */
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

  // --- Identity ---
  const displayId = campaignDisplayId(campaign);
  const nameParts = parseCampaignListName(campaign.name);

  // --- Status ---
  const statusLabel = formatCampaignStatusLabel(campaign.status, row.status_label);
  const statusBadgeClass = campaignStatusBadgeClass(campaign.status, row.status_tone);

  // --- Row decoration ---
  const rowClass = campaignListRowClass({
    status: campaign.status,
    statusTone: row.status_tone,
    selected,
    highlightActiveRows,
    margin,
  });
  const rowStatusEdgeClass = campaignListRowStatusEdgeClass(campaign.status, row.status_tone);
  const hasTraffic = !isRowWithoutTraffic({
    clicks,
    impressions,
    blocks,
    costMicro,
    profitMicro,
    revenueMicro,
    funnel,
  });

  // --- Profit indicator ---
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

  // --- Funnel cells ---
  const ctr =
    impressions > 0 && clicks > 0
      // percentRate is non-null here: both args are > 0
      ? { text: formatSourceCtr(clicks, impressions), valPct: percentRate(clicks, impressions)!, isZero: false }
      : null;

  const lpCtr =
    clicks > 0 && funnel.lpClicks > 0
      ? { text: formatLpCtr(funnel.lpClicks, clicks), valPct: percentRate(funnel.lpClicks, clicks)!, isZero: false }
      : null;

  const crRes = formatTableCr(clicks, funnel.approved);
  const cr: VmRateCell = crRes;

  const approveRate =
    funnel.rawLeads > 0
      ? {
          text: formatApproveRate(funnel.approved, funnel.rawLeads),
          // percentRate is non-null: rawLeads > 0 and approved >= 0 (clamped below)
          valPct: percentRate(Math.max(0, funnel.approved), funnel.rawLeads) ?? 0,
          isZero: false,
        }
      : null;

  const blockPct = clicks > 0 && blocks > 0 ? formatRelativeRate(blocks, clicks) : null;
  const botPct = clicks > 0 && funnel.bots > 0 ? formatRelativeRate(funnel.bots, clicks) : null;
  const cpm = impressions > 0 && costMicro > 0 ? formatCpmUsd(costMicro, impressions) : null;

  // --- Money ---
  const revenue =
    revenueMicro > 0
      ? formatTableMoneyFromMicro(revenueMicro)
      : parseAndFormatTableMoneyStr(row.current_spend_display ?? row.current_spend);

  const cost =
    costMicro > 0
      ? formatTableMoneyFromMicro(costMicro)
      : parseAndFormatTableMoneyStr(row.current_spend_display ?? row.current_spend);

  const profitRes = formatTableMoneyFromMicro(profitMicro);
  const roiRes = formatTableRoi(profitMicro, costMicro);

  const epcMicro = clicks > 0 ? Math.trunc(revenueMicro / clicks) : 0;
  const cpcMicro = clicks > 0 ? Math.trunc(costMicro / clicks) : 0;
  const cpaMicro = funnel.rawLeads > 0 ? Math.trunc(costMicro / funnel.rawLeads) : 0;
  const ecpaMicro = funnel.approved > 0 ? Math.trunc(costMicro / funnel.approved) : 0;

  // --- Budget pct ---
  const spendUsd = parseAndFormatTableMoneyStr(row.current_spend_display ?? row.current_spend).valUsd;
  const limitUsd = parseAndFormatTableMoneyStr(row.budget_limit_display ?? row.budget_limit).valUsd;
  const budgetPct = limitUsd > 0 ? (spendUsd / limitUsd) * 100 : null;

  // --- Descriptive ---
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
    epc: formatTableMoneyFromMicro(epcMicro),
    cpc: formatTableMoneyFromMicro(cpcMicro),
    cpa: formatTableMoneyFromMicro(cpaMicro),
    ecpa: formatTableMoneyFromMicro(ecpaMicro),
    budgetPct,

    groupLabel,
    groupCustomerId: campaign.customer_id,
    flowId: campaign.flow_id ?? null,
    ownerLabel,
    ownerId,
    countries: campaign.target_countries ?? [],
  };
}
