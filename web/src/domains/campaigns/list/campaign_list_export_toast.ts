import { CAMPAIGN_LIST_EXPORT_MAX_ROWS } from '@/domains/campaigns/list/campaign_list_limits';

export function formatCampaignListExportToast(
  exportedCount: number,
  matchedTotal: number,
  truncated: boolean,
  format: 'CSV' | 'JSON',
): string {
  if (truncated) {
    return `Exported ${exportedCount.toLocaleString()} of ${matchedTotal.toLocaleString()} campaign(s) as ${format} (max ${CAMPAIGN_LIST_EXPORT_MAX_ROWS.toLocaleString()})`;
  }
  return `Exported ${exportedCount.toLocaleString()} campaign(s) as ${format}`;
}
