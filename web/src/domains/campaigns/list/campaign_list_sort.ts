import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';

// Client sort applies only to the current server page (limit/offset snapshot).
// Metric columns are not in CampaignListQuery sort; regime S per frontend-hot-path.mdc.

import type { CampaignListMiddleColumnId } from '@/domains/campaigns/list/campaign_list_columns';
import {
  formatApproveRate,
  formatCpmUsd,
  formatLpCtr,
  formatRelativeRate,
  formatSourceCtr,
  resolveCampaignFunnelCounts,
} from '@/domains/campaigns/list/campaign_list_funnel';
import type { CampaignSortField, SortOrder } from '@/domains/campaigns/list/campaigns_list_types';
import { campaignListRevenueMicro } from '@/domains/campaigns/list/campaign_list_format';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';

export const CAMPAIGN_SERVER_SORT_FIELDS = new Set<CampaignSortField>([
  'name',
  'updated_at',
  'spend',
  'budget_limit',
]);

export function isCampaignServerSortField(field: CampaignSortField): boolean {
  return CAMPAIGN_SERVER_SORT_FIELDS.has(field);
}

export function isCampaignClientSortField(
  field: CampaignSortField,
): field is CampaignListMiddleColumnId {
  return !isCampaignServerSortField(field);
}

function parsePercentSortValue(raw: string): number {
  const cleaned = raw.replace('%', '').trim();
  const parsed = Number.parseFloat(cleaned);
  return Number.isFinite(parsed) ? parsed : 0;
}

export function campaignListClientSortValue(
  campaign: Campaign,
  field: CampaignListMiddleColumnId,
  metrics: CampaignListMetrics | undefined,
  margin: CampaignMargin | undefined,
): number {
  const clicks = metrics?.clicks ?? 0;
  const impressions = metrics?.impressions ?? 0;
  const conversions = metrics?.conversions ?? 0;
  const costMicro = margin?.rtb_cost_micro ?? 0;
  const revenueMicro = campaignListRevenueMicro(margin);
  const profitMicro = margin?.operator_margin_micro ?? 0;
  const funnel = resolveCampaignFunnelCounts(metrics);
  const blocks = metrics?.blocks ?? 0;

  switch (field) {
    case 'tags':
    case 'status':
    case 'flow':
    case 'owner':
    case 'countries':
    case 'group':
      return 0;
    case 'clicks':
      return clicks;
    case 'impressions':
      return impressions;
    case 'unique_clicks':
      return metrics?.unique_clicks ?? 0;
    case 'ctr':
      return parsePercentSortValue(formatSourceCtr(clicks, impressions));
    case 'lp_clicks':
      return funnel.lpClicks;
    case 'lp_views':
      return funnel.lpViews;
    case 'lp_ctr':
      return parsePercentSortValue(formatLpCtr(funnel.lpClicks, clicks));
    case 'blocks':
      return blocks;
    case 'block_pct':
      return parsePercentSortValue(formatRelativeRate(blocks, clicks));
    case 'bots':
      return funnel.bots;
    case 'bot_pct':
      return parsePercentSortValue(formatRelativeRate(funnel.bots, clicks));
    case 'leads':
      return funnel.rawLeads;
    case 'approved':
      return funnel.approved;
    case 'hold_leads':
      return funnel.hold;
    case 'rejected_leads':
      return funnel.rejected;
    case 'approve_rate':
      return parsePercentSortValue(formatApproveRate(funnel.approved, funnel.rawLeads));
    case 'revenue':
      return revenueMicro;
    case 'cost':
      return costMicro;
    case 'profit':
      return profitMicro;
    case 'cr':
      return clicks > 0 ? conversions / clicks : 0;
    case 'epc':
      return clicks > 0 ? revenueMicro / clicks : 0;
    case 'cpc':
      return clicks > 0 ? costMicro / clicks : 0;
    case 'cpa':
      return conversions > 0 ? costMicro / conversions : 0;
    case 'ecpa':
      return funnel.approved > 0 ? costMicro / funnel.approved : 0;
    case 'cpm':
      return Number.parseFloat(formatCpmUsd(costMicro, impressions)) || 0;
    case 'roi':
      return costMicro > 0 ? profitMicro / costMicro : 0;
    case 'budget_pct': {
      const spend = Number.parseFloat((campaign.current_spend ?? '0').replace(/[^\d.-]/g, ''));
      const limit = Number.parseFloat((campaign.budget_limit ?? '0').replace(/[^\d.-]/g, ''));
      return limit > 0 ? spend / limit : 0;
    }
    default:
      return 0;
  }
}

export function sortCampaignListItemsClient(
  items: readonly Campaign[],
  field: CampaignListMiddleColumnId,
  order: SortOrder,
  metricsById: Readonly<Record<string, CampaignListMetrics>>,
  marginsById: Readonly<Record<string, CampaignMargin>>,
): Campaign[] {
  const direction = order === 'asc' ? 1 : -1;
  return [...items].sort((left, right) => {
    const leftValue = campaignListClientSortValue(
      left,
      field,
      metricsById[left.id],
      marginsById[left.id],
    );
    const rightValue = campaignListClientSortValue(
      right,
      field,
      metricsById[right.id],
      marginsById[right.id],
    );
    if (leftValue === rightValue) {
      return left.name.localeCompare(right.name) * direction;
    }
    return (leftValue - rightValue) * direction;
  });
}

export function sortFieldForCampaignColumn(
  columnId: string,
): CampaignSortField | undefined {
  switch (columnId) {
    case 'name':
      return 'name';
    case 'id':
      return 'updated_at';
    case 'cost':
      return 'spend';
    case 'tags':
    case 'status':
    case 'clicks':
    case 'impressions':
    case 'ctr':
    case 'unique_clicks':
    case 'lp_clicks':
    case 'lp_views':
    case 'group':
    case 'lp_ctr':
    case 'cr':
    case 'leads':
    case 'approved':
    case 'hold_leads':
    case 'rejected_leads':
    case 'approve_rate':
    case 'epc':
    case 'cpc':
    case 'cpa':
    case 'ecpa':
    case 'cpm':
    case 'blocks':
    case 'block_pct':
    case 'bots':
    case 'bot_pct':
    case 'revenue':
    case 'profit':
    case 'roi':
      return columnId;
    default:
      return undefined;
  }
}
