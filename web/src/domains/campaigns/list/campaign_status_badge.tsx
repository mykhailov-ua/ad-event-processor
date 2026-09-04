import type { Campaign } from '@/api/types';
import { campaignStatusToAdminTone } from '@/lib/admin_kit';
import { formatCampaignStatusLabel } from '@/lib/admin_typography';
import { StatusBadge } from '@/shell/status_badge';

type CampaignStatusTone = 'success' | 'warning' | 'muted' | string;

type CampaignWithStatusDisplay = Campaign & {
  status_label?: string;
  status_tone?: CampaignStatusTone;
};

export type CampaignStatusBadgeProps = {
  campaign: CampaignWithStatusDisplay;
  className?: string;
};

export function CampaignStatusBadge({ campaign, className }: CampaignStatusBadgeProps) {
  const label = formatCampaignStatusLabel(campaign.status, campaign.status_label);
  const tone = campaignStatusToAdminTone(campaign.status, campaign.status_tone);

  return <StatusBadge className={className} label={label} tone={tone} />;
}
