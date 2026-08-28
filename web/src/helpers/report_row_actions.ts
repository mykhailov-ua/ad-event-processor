import { apiConfirmed } from './confirmed_api.js';
import { pauseCampaign } from './campaign_actions.js';

export type ReportRowActionContext = {
  customerId?: string;
  campaignId?: string;
  placementId?: string;
  sub1?: string;
  sub2?: string;
  ivtRate?: number;
  spendMicro?: number;
};

export function reportRowCampaignId(row: Record<string, unknown>): string {
  const id = row.campaign_id;
  return typeof id === 'string' ? id : '';
}

export function reportRowPlacementId(row: Record<string, unknown>): string {
  const placement = row.placement_id ?? row.sub1;
  return typeof placement === 'string' ? placement : '';
}

export function smartAlertPrefillHref(ctx: ReportRowActionContext): string {
  const params = new URLSearchParams();
  if (ctx.customerId) params.set('customer_id', ctx.customerId);
  if (ctx.campaignId) params.set('campaign_id', ctx.campaignId);
  params.set('prefill', '1');
  params.set('metric', ctx.ivtRate != null && ctx.ivtRate > 0.1 ? 'bot_clicks' : 'clicks');
  params.set('operator', 'gt');
  const threshold =
    ctx.ivtRate != null && ctx.ivtRate > 0 ? Math.max(1, Math.round(ctx.ivtRate * 100)) : 100;
  params.set('threshold', String(threshold));
  const label = ctx.placementId || ctx.sub1 || ctx.campaignId || 'report';
  params.set('name', `Alert: ${label}`);
  return `/integrations/smart-alerts?${params.toString()}`;
}

export function costSyncDrillHref(campaignId: string, customerId?: string): string {
  const params = new URLSearchParams();
  if (customerId) params.set('customer_id', customerId);
  params.set('campaign_id', campaignId);
  return `/integrations/cost-sync?${params.toString()}`;
}

export async function blockReportSource(ctx: ReportRowActionContext): Promise<void> {
  const placementId = ctx.placementId || ctx.sub1;
  if (!ctx.campaignId || !placementId) {
    throw new Error('campaign_id and placement/sub required to block source');
  }
  await apiConfirmed(`/api/v1/campaigns/${ctx.campaignId}/placement-blocks`, {
    method: 'POST',
    body: JSON.stringify({ placement_id: placementId }),
    idempotencyScope: `placement-block:${ctx.campaignId}:${placementId}`,
  });
}

export async function pauseReportCampaign(campaignId: string): Promise<void> {
  await pauseCampaign(campaignId);
}
