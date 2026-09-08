import { Link } from 'react-router-dom';

import { campaignListEllipsisTextClass, campaignListNameTextClass } from '@/domains/campaigns/list/campaign_list_classes';
import { campaignReportPath } from '@/lib/campaign_nav';
import { cn } from '@/lib/utils';

/** Analytics drill-down: campaign report (read-only), not the editor. */
export function campaignReportBreakdownLink(row: { id?: string; name?: string }) {
  if (!row.id) {
    return (
      <span className={campaignListNameTextClass} title={row.name}>
        {row.name}
      </span>
    );
  }
  return (
    <Link
      className={cn(campaignListEllipsisTextClass, 'text-primary hover:underline')}
      title={`Open report for ${row.name ?? row.id}`}
      to={campaignReportPath(row.id)}
    >
      {row.name}
    </Link>
  );
}

/** @deprecated Use campaignReportBreakdownLink. */
export const campaignBreakdownLink = campaignReportBreakdownLink;
