export type CampaignStatusKey = 'ACTIVE' | 'PAUSED' | 'ARCHIVED' | 'UNKNOWN';

export type CampaignStatusTone = 'success' | 'warning' | 'muted' | string;

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

export function campaignListRowClass(selected: boolean): string {
  if (selected) {
    return 'bg-blue-50 dark:bg-blue-950/30';
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
