import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';
import { formatDashboardCrPct, formatDashboardRoiPct } from '@/domains/dashboards/dashboard_format';
import {
  formatTableCount,
  formatTableMoneyFromMicro,
  parseAndFormatTableMoneyStr,
} from '@/domains/campaigns/list/campaign_list_format';
import type { CampaignFunnelCounts } from '@/domains/campaigns/list/campaign_list_funnel';
import { resolveCampaignListRowMetrics } from '@/domains/campaigns/list/campaign_list_row_metrics';
import {
  campaignListRowClass,
  campaignStatusBadgeClass,
} from '@/domains/campaigns/list/campaign_list_row_tone';
import { profitToneClassFromMicro, roiToneClassFromRate } from '@/domains/campaigns/list/campaign_list_tone';
import type { CampaignListMiddleColumnId } from '@/domains/campaigns/list/campaign_list_columns';
import { campaignDisplayId } from '@/domains/campaigns/list/campaign_display_id';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';
import { formatCampaignStatusLabel } from '@/lib/admin_typography';
import { resolveCustomerLabel } from '@/lib/customer_label';

export type VmCell = { text: string; isZero: boolean };

export type VmRateCell = { text: string; valPct: number; isZero: boolean };

export type CampaignRowVm = {
  id: string;
  displayId: string;
  rawName: string;

  statusLabel: string;
  statusBadgeClass: string;

  rowClass: string;

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
): CampaignRowVm {
  const row = campaign as CampaignWithMoneyDisplay;
  const { clicks, impressions, blocks, costMicro, profitMicro, revenueMicro, funnel } =
    resolveCampaignListRowMetrics(metrics, margin);

  const displayId = campaignDisplayId(campaign);

  const statusLabel = formatCampaignStatusLabel(campaign.status, row.status_label);
  const statusBadgeClass = campaignStatusBadgeClass(campaign.status, row.status_tone);
  const rowClass = campaignListRowClass(selected);

  const ctr = vmOptionalRate(metrics?.ctr_pct);
  const lpCtr = vmOptionalRate(metrics?.lp_ctr_pct);
  const cr = vmRateOrZero(metrics?.cr_pct);
  const approveRate = vmOptionalRate(metrics?.approve_rate_pct);
  const blockPct = optionalRateLabel(metrics?.block_pct);
  const botPct = optionalRateLabel(metrics?.bot_pct);
  const cpm = metrics?.cpm_usd && metrics.cpm_usd !== '0.00' ? metrics.cpm_usd : null;

  const revenue = formatTableMoneyFromMicro(revenueMicro);
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
    rawName: campaign.name,

    statusLabel,
    statusBadgeClass,

    rowClass,

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
    profitToneClass: profitToneClassFromMicro(profitMicro),
    roi: roiRes,
    roiToneClass: roiToneClassFromRate(roiRes),
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

// Width probe strings must match CampaignListTableMiddleCell output.
export function campaignListMiddleCellDisplayText(
  columnId: CampaignListMiddleColumnId,
  vm: CampaignRowVm,
): string {
  switch (columnId) {
    case 'status':
      return vm.statusLabel;
    case 'tags':
      return '-';
    case 'clicks':
      return vm.clicks.text;
    case 'impressions':
      return vm.impressions.text;
    case 'ctr':
      return vm.ctr?.text ?? '-';
    case 'unique_clicks':
      return vm.uniqueClicks.text;
    case 'lp_clicks':
      return vm.lpClicks.text;
    case 'lp_views':
      return vm.lpViews.text;
    case 'group':
      return vm.groupLabel ?? vm.groupCustomerId;
    case 'lp_ctr':
      return vm.lpCtr?.text ?? '-';
    case 'cr':
      return vm.cr.text;
    case 'leads':
      return vm.leads.text;
    case 'approved':
      return vm.approved.text;
    case 'hold_leads':
      return vm.holdLeads.text;
    case 'rejected_leads':
      return vm.rejectedLeads.text;
    case 'approve_rate':
      return vm.approveRate?.text ?? '-';
    case 'epc':
      return vm.epc.text;
    case 'cpc':
      return vm.cpc.text;
    case 'cpa':
      return vm.cpa.text;
    case 'ecpa':
      return vm.ecpa.text;
    case 'cpm':
      return vm.cpm ?? '-';
    case 'blocks':
      return vm.blocks.text;
    case 'block_pct':
      return vm.blockPct ?? '-';
    case 'bots':
      return vm.bots.text;
    case 'bot_pct':
      return vm.botPct ?? '-';
    case 'revenue':
      return vm.revenue.text;
    case 'cost':
      return vm.cost.text;
    case 'profit':
      return vm.profit.isZero ? '0.00' : vm.profit.text;
    case 'roi':
      return vm.roi.isZero ? '0%' : vm.roi.text;
    case 'budget_pct':
      return vm.budgetPct == null ? '-' : `${vm.budgetPct.toFixed(1)}%`;
    case 'flow':
      return vm.flowId ? vm.flowId.slice(0, 8) : '-';
    case 'owner':
      return vm.ownerLabel;
    case 'countries':
      return vm.countries.length > 0 ? vm.countries.join(', ') : '-';
    default:
      return '';
  }
}
