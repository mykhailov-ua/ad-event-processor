import {
  adminStatusBadgeBase,
  adminStatusBadgeClass,
  campaignStatusToAdminTone,
} from '@/lib/admin_kit';

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
  const tone = [
    'odd:[&_td]:bg-background',
    'even:[&_td]:bg-muted/30',
    'hover:[&_td]:bg-accent',
  ];
  if (selected) {
    tone.push('[&_td]:bg-accent/80');
  }
  return tone.join(' ');
}

export function campaignStatusBadgeClass(
  status: string,
  statusTone?: CampaignStatusTone,
): string {
  const tone = campaignStatusToAdminTone(status, statusTone);
  return `${adminStatusBadgeBase} ${adminStatusBadgeClass[tone]}`;
}
