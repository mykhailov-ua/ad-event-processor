import type { CampaignMargin } from '@/api/types';

export type CampaignStatusKey = 'ACTIVE' | 'PAUSED' | 'ARCHIVED' | 'UNKNOWN';

export type CampaignStatusTone = 'success' | 'warning' | 'muted' | string;

export type CampaignListPerformanceRowTone = 'negative' | 'warning';

export function normalizeCampaignStatus(status: string): CampaignStatusKey {
  const normalized = status.trim().toUpperCase();
  if (normalized === 'ACTIVE' || normalized === 'PAUSED' || normalized === 'ARCHIVED') {
    return normalized;
  }
  return 'UNKNOWN';
}

export function resolveCampaignStatusKey(
  status: string,
  statusTone?: CampaignStatusTone,
): CampaignStatusKey {
  if (statusTone === 'success') {
    return 'ACTIVE';
  }
  if (statusTone === 'warning') {
    return 'PAUSED';
  }
  if (statusTone === 'muted') {
    return 'ARCHIVED';
  }
  return normalizeCampaignStatus(status);
}

export function isInactiveCampaignStatus(statusKey: CampaignStatusKey): boolean {
  return statusKey !== 'ACTIVE';
}

export function resolvePerformanceRowTone(
  margin?: CampaignMargin,
): CampaignListPerformanceRowTone | null {
  if (!margin) {
    return null;
  }
  const profitMicro = margin.operator_margin_micro;
  if (profitMicro != null && profitMicro < 0) {
    return 'negative';
  }
  if (margin.margin_breach) {
    return 'warning';
  }
  return null;
}

export function campaignListRowClass(args: {
  status: string;
  statusTone?: CampaignStatusTone;
  selected: boolean;
  highlightActiveRows?: boolean;
  margin?: CampaignMargin;
}): string {
  if (args.selected) {
    return 'admin-row-selected';
  }

  const performanceTone = resolvePerformanceRowTone(args.margin);
  if (performanceTone === 'negative') {
    return 'admin-row-negative';
  }
  if (performanceTone === 'warning') {
    return 'admin-row-warning';
  }

  return '';
}

export function campaignListStatusDotClass(
  status: string,
  statusTone?: CampaignStatusTone,
): string {
  const key = resolveCampaignStatusKey(status, statusTone);
  if (key === 'ACTIVE') {
    return 'admin-table-status-dot--active';
  }
  return 'admin-table-status-dot--muted';
}

export function campaignListRowStatusEdgeClass(
  status: string,
  statusTone?: CampaignStatusTone,
): string {
  const key = resolveCampaignStatusKey(status, statusTone);
  if (key === 'ACTIVE') {
    return 'admin-row-status-edge--active';
  }
  if (key === 'PAUSED') {
    return 'admin-row-status-edge--paused';
  }
  if (key === 'ARCHIVED') {
    return 'admin-row-status-edge--archived';
  }
  return 'admin-row-status-edge--unknown';
}

export function campaignStatusBadgeClass(
  status: string,
  statusTone?: CampaignStatusTone,
): string {
  const key = resolveCampaignStatusKey(status, statusTone);
  if (key === 'ACTIVE') {
    return 'admin-campaign-status admin-campaign-status--active';
  }
  if (key === 'PAUSED') {
    return 'admin-campaign-status admin-campaign-status--paused';
  }
  if (key === 'ARCHIVED') {
    return 'admin-campaign-status admin-campaign-status--archived';
  }
  return 'admin-campaign-status';
}
