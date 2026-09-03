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
    return 'bg-blue-50 dark:bg-blue-950/30';
  }

  const performanceTone = resolvePerformanceRowTone(args.margin);
  if (performanceTone === 'negative') {
    return '';
  }
  if (performanceTone === 'warning') {
    return '';
  }

  return '';
}

export function campaignListStatusDotClass(
  status: string,
  statusTone?: CampaignStatusTone,
): string {
  const key = resolveCampaignStatusKey(status, statusTone);
  if (key === 'ACTIVE') {
    return 'inline-block h-2 w-2 rounded-full bg-green-500';
  }
  return 'inline-block h-2 w-2 rounded-full bg-zinc-400';
}

export function campaignListRowStatusEdgeClass(
  status: string,
  statusTone?: CampaignStatusTone,
): string {
  const key = resolveCampaignStatusKey(status, statusTone);
  if (key === 'ACTIVE') {
    return '';
  }
  if (key === 'PAUSED') {
    return '';
  }
  if (key === 'ARCHIVED') {
    return '';
  }
  return '';
}

export function campaignStatusBadgeClass(
  status: string,
  statusTone?: CampaignStatusTone,
): string {
  const key = resolveCampaignStatusKey(status, statusTone);
  if (key === 'ACTIVE') {
    return 'inline-flex items-center gap-1.5 text-xs text-green-700 dark:text-green-300';
  }
  if (key === 'PAUSED') {
    return 'inline-flex items-center gap-1.5 text-xs text-amber-700 dark:text-amber-300';
  }
  if (key === 'ARCHIVED') {
    return 'inline-flex items-center gap-1.5 text-xs text-zinc-500';
  }
  return 'inline-flex items-center gap-1.5 text-xs';
}
