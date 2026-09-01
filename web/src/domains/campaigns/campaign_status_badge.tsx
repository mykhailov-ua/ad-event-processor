import { Badge } from '@/components/ui/badge';
import type { Campaign } from '@/api/types';

type CampaignStatusTone = 'success' | 'warning' | 'muted' | string;

type CampaignWithStatusDisplay = Campaign & {
  status_label?: string;
  status_tone?: CampaignStatusTone;
};

function normalizeStatus(status: string): string {
  return status.trim().toUpperCase();
}

function statusLabel(status: string, statusLabelOverride?: string): string {
  if (statusLabelOverride) {
    return statusLabelOverride;
  }
  switch (normalizeStatus(status)) {
    case 'ACTIVE':
      return 'Active';
    case 'PAUSED':
      return 'Paused';
    case 'ARCHIVED':
      return 'Archived';
    default:
      return status || 'Unknown';
  }
}

function badgeVariant(
  status: string,
  tone?: CampaignStatusTone,
): 'default' | 'secondary' | 'outline' {
  if (tone === 'success') {
    return 'default';
  }
  if (tone === 'warning') {
    return 'secondary';
  }
  if (tone === 'muted') {
    return 'outline';
  }

  switch (normalizeStatus(status)) {
    case 'ACTIVE':
      return 'default';
    case 'PAUSED':
      return 'secondary';
    default:
      return 'outline';
  }
}

export type CampaignStatusBadgeProps = {
  campaign: CampaignWithStatusDisplay;
  className?: string;
};

export function CampaignStatusBadge({ campaign, className }: CampaignStatusBadgeProps) {
  const label = statusLabel(campaign.status, campaign.status_label);
  const variant = badgeVariant(campaign.status, campaign.status_tone);

  return (
    <Badge className={className} variant={variant}>
      {label}
    </Badge>
  );
}
