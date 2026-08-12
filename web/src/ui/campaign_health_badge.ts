import { el } from '../lib/dom.js';
import { deriveCampaignHealth } from '../models/campaign_health.js';

/**
 * Map health level to status-badge modifier.
 */
function healthMod(level: string): string {
  if (level === 'risk') return 'danger';
  if (level === 'warn') return 'warning';
  if (level === 'paused') return 'muted';
  return 'success';
}

export type CampaignHealthBadgeCtx = {
  portfolioRow?: Record<string, unknown>;
  attentionReason?: string;
  ivtElevated?: boolean;
  marginBreach?: boolean;
  licenseGrace?: boolean;
};

/**
 * Render a compact campaign health badge with tooltip title.
 */
export function renderCampaignHealthBadge(
  campaign: Record<string, unknown>,
  ctx: CampaignHealthBadgeCtx = {},
): HTMLElement {
  const health = deriveCampaignHealth(campaign, ctx);
  return el('span', {
    className: `status-badge status-badge--${healthMod(health.level)}`,
    title: health.title,
  }, health.label);
}
