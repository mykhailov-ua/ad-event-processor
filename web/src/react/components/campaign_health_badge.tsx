import { deriveCampaignHealth } from '../../models/campaign_health.js';
import type { CampaignDTO } from '../../types/api/campaign.js';

export type CampaignHealthBadgeCtx = {
  portfolioRow?: Record<string, unknown>;
  attentionReason?: string;
  ivtElevated?: boolean;
  marginBreach?: boolean;
  licenseGrace?: boolean;
};

function healthMod(level: string): string {
  if (level === 'risk') return 'danger';
  if (level === 'warn') return 'warning';
  if (level === 'paused') return 'muted';
  return 'success';
}

/**
 * Compact campaign health badge with tooltip title.
 */
export function CampaignHealthBadge({
  campaign,
  ctx = {},
}: {
  campaign: CampaignDTO;
  ctx?: CampaignHealthBadgeCtx;
}) {
  const health = deriveCampaignHealth(campaign, ctx);
  return (
    <span
      className={`status-badge status-badge--${healthMod(health.level)}`}
      title={health.title}
    >
      {health.label}
    </span>
  );
}
