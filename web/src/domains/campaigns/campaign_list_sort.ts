import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';

import type { CampaignListMiddleColumnId } from '@/domains/campaigns/campaign_list_columns';
import type { CampaignSortField, SortOrder } from '@/domains/campaigns/campaigns_list_types';
import { campaignListRevenueMicro } from '@/domains/campaigns/campaign_list_format';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/campaign_metrics_shared';

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

export function campaignListClientSortValue(
  campaign: Campaign,
  field: CampaignListMiddleColumnId,
  metrics: CampaignListMetrics | undefined,
  margin: CampaignMargin | undefined,
): number {
  const row = campaign as CampaignWithMoneyDisplay;
  const clicks = metrics?.clicks ?? 0;
  const conversions = metrics?.conversions ?? 0;
  const costMicro = margin?.rtb_cost_micro ?? 0;
  const revenueMicro = campaignListRevenueMicro(margin);
  const profitMicro = margin?.operator_margin_micro ?? 0;

  switch (field) {
    case 'source':
      return row.traffic_template_id ? 1 : 0;
    case 'flows':
      return row.flow_id ? 1 : 0;
    case 'clicks':
      return clicks;
    case 'conversions':
      return conversions;
    case 'revenue':
      return revenueMicro;
    case 'cost':
      return costMicro;
    case 'profit':
      return profitMicro;
    case 'cr':
      return clicks > 0 ? conversions / clicks : 0;
    case 'roi':
      return costMicro > 0 ? profitMicro / costMicro : 0;
    case 'group':
      return 0;
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
    case 'source':
    case 'flows':
    case 'clicks':
    case 'conversions':
    case 'revenue':
    case 'profit':
    case 'cr':
    case 'roi':
    case 'group':
      return columnId;
    default:
      return undefined;
  }
}
