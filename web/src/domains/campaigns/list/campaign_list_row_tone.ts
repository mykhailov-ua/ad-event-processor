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
  if (selected) {
    return 'campaign-row--selected';
  }
  return '';
}

export function campaignStatusBadgeClass(
  status: string,
  statusTone?: CampaignStatusTone,
): string {
  const tone = campaignStatusToAdminTone(status, statusTone);
  return `${adminStatusBadgeBase} ${adminStatusBadgeClass[tone]}`;
}
