import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';

import type { CampaignListRowAlert } from '@/domains/campaigns/list/campaign_list_row_tone';

export type CampaignListRowAlertBadgeProps = {
  alert: Exclude<CampaignListRowAlert, 'none'>;
  marginBreach?: boolean;
  budgetUsedPct?: number | null;
  status?: string;
};

function alertTooltip({
  alert,
  marginBreach,
  budgetUsedPct,
  status,
}: CampaignListRowAlertBadgeProps): string {
  if (marginBreach) {
    return 'Margin guard breach in the current reporting window';
  }
  const normalized = status?.trim().toUpperCase() ?? '';
  if (normalized === 'EXHAUSTED') {
    return 'Campaign budget is exhausted';
  }
  if (budgetUsedPct != null && budgetUsedPct >= 90) {
    return `Budget ${budgetUsedPct.toFixed(1)}% used`;
  }
  if (alert === 'critical') {
    return 'Campaign needs attention';
  }
  return 'Campaign budget is nearly exhausted';
}

export function CampaignListRowAlertBadge(props: CampaignListRowAlertBadgeProps) {
  const { alert } = props;
  const toneClass =
    alert === 'critical' ? 'bg-destructive text-black' : 'bg-admin-status-paused text-black';

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          aria-label="Campaign alert"
         
          role="img"
        >
          !
        </span>
      </TooltipTrigger>
      <TooltipContent>{alertTooltip(props)}</TooltipContent>
    </Tooltip>
  );
}
