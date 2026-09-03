import type { CampaignListMiddleColumnId } from '@/domains/campaigns/list/campaign_list_columns';
import type { CampaignSortField } from '@/domains/campaigns/list/campaigns_list_types';

export const CAMPAIGN_LIST_API_SORT_FIELDS = [
  'name',
  'updated_at',
  'spend',
  'budget_limit',
  'status',
  'group',
  'flow',
  'owner',
  'countries',
  'budget_pct',
  'clicks',
  'impressions',
  'conversions',
  'unique_clicks',
  'blocks',
  'cost',
  'revenue',
  'profit',
  'roi',
  'ctr',
  'cr',
  'cpc',
  'cpa',
  'ecpa',
  'epc',
  'cpm',
  'block_pct',
  'leads',
  'approved',
  'hold_leads',
  'rejected_leads',
  'approve_rate',
  'lp_clicks',
  'lp_views',
  'lp_ctr',
  'bots',
  'bot_pct',
] as const;

export type CampaignListApiSortField = (typeof CAMPAIGN_LIST_API_SORT_FIELDS)[number];

const CAMPAIGN_LIST_STATS_SORT_FIELDS = new Set<CampaignListApiSortField>([
  'clicks',
  'impressions',
  'conversions',
]);

const CAMPAIGN_LIST_EXTENDED_METRIC_SORT_FIELDS = new Set<CampaignListApiSortField>([
  'unique_clicks',
  'blocks',
  'cost',
  'revenue',
  'profit',
  'roi',
  'ctr',
  'cr',
  'cpc',
  'cpa',
  'ecpa',
  'epc',
  'cpm',
  'block_pct',
  'leads',
  'approved',
  'hold_leads',
  'rejected_leads',
  'approve_rate',
  'lp_clicks',
  'lp_views',
  'lp_ctr',
  'bots',
  'bot_pct',
]);

export function campaignListSortNeedsMetricWindow(field: CampaignListApiSortField): boolean {
  return (
    CAMPAIGN_LIST_STATS_SORT_FIELDS.has(field) ||
    CAMPAIGN_LIST_EXTENDED_METRIC_SORT_FIELDS.has(field)
  );
}

export function campaignListSortToApi(field: CampaignSortField): CampaignListApiSortField {
  if (field === 'id' || field === 'updated_at') {
    return 'updated_at';
  }
  return field as CampaignListApiSortField;
}

export function sortFieldForCampaignColumn(
  columnId: string,
): CampaignSortField | undefined {
  switch (columnId) {
    case 'name':
      return 'name';
    case 'id':
      return 'updated_at';
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
    case 'cost':
    case 'profit':
    case 'roi':
    case 'budget_pct':
    case 'flow':
    case 'owner':
    case 'countries':
      return columnId as CampaignListMiddleColumnId;
    default:
      return undefined;
  }
}
