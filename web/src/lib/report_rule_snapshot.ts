import type { SourceQualityGroupBy, SourceQualityRow } from '@/api/types';

export type ReportRuleAction = 'pause_campaign' | 'blacklist_placement';

export type ReportRuleFilterContext = {
  customerId: string;
  campaignId?: string;
  from?: string;
  to?: string;
  groupBy?: SourceQualityGroupBy[];
  compare?: boolean;
};

export function defaultReportRuleActionForRow(row: SourceQualityRow): ReportRuleAction {
  if (row.placement_id?.trim()) {
    return 'blacklist_placement';
  }
  return 'pause_campaign';
}

export function resolveReportRuleCampaignId(
  row: SourceQualityRow,
  context: ReportRuleFilterContext
): string | undefined {
  const rowCampaign = row.campaign_id?.trim();
  if (rowCampaign) {
    return rowCampaign;
  }
  const filterCampaign = context.campaignId?.trim();
  return filterCampaign || undefined;
}

export function buildSourceQualityFilterSnapshot(
  row: SourceQualityRow,
  context: ReportRuleFilterContext
): Record<string, string> {
  const snapshot: Record<string, string> = {
    report_key: 'source-quality',
  };
  if (context.customerId.trim()) {
    snapshot.customer_id = context.customerId.trim();
  }
  if (context.from?.trim()) {
    snapshot.from = context.from.trim();
  }
  if (context.to?.trim()) {
    snapshot.to = context.to.trim();
  }
  const campaignId = resolveReportRuleCampaignId(row, context);
  if (campaignId) {
    snapshot.campaign_id = campaignId;
  }
  if (row.placement_id?.trim()) {
    snapshot.placement_id = row.placement_id.trim();
  }
  if (row.country?.trim()) {
    snapshot.country = row.country.trim();
  }
  if (row.city?.trim()) {
    snapshot.city = row.city.trim();
  }
  if (row.device?.trim()) {
    snapshot.device = row.device.trim();
  }
  if (row.sub1?.trim()) {
    snapshot.sub1 = row.sub1.trim();
  }
  if (context.groupBy && context.groupBy.length > 0) {
    snapshot.group_by = context.groupBy.join(',');
  }
  if (context.compare) {
    snapshot.compare = '1';
  }
  return snapshot;
}

export function defaultReportRuleName(
  reportKey: string,
  row: SourceQualityRow,
  action: ReportRuleAction
): string {
  const label =
    row.placement_id ??
    row.campaign_id ??
    row.country ??
    row.city ??
    row.device ??
    row.sub1 ??
    'row';
  const verb = action === 'blacklist_placement' ? 'Blacklist' : 'Pause';
  return `${verb} from ${reportKey}: ${label}`;
}
