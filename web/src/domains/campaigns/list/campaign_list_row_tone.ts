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
  statusTone?: CampaignStatusTone
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

export type CampaignListRowAccent = 'none' | 'muted' | 'warning' | 'critical';

export type CampaignListRowAlert = 'none' | 'warning' | 'critical';

export function resolveCampaignListRowAccent(
  _status: string,
  _statusTone?: CampaignStatusTone
): CampaignListRowAccent {
  return 'none';
}

export function resolveCampaignListRowAlert(
  status: string,
  statusTone?: CampaignStatusTone,
  options?: {
    budgetUsedPct?: number | null;
    marginBreach?: boolean;
  }
): CampaignListRowAlert {
  const statusKey = resolveCampaignStatusKey(status, statusTone);
  if (statusKey === 'PAUSED' || statusKey === 'ARCHIVED') {
    return 'none';
  }

  const normalized = status.trim().toUpperCase();
  if (options?.marginBreach === true || normalized === 'ERROR' || normalized === 'FAILED') {
    return 'critical';
  }

  const budgetUsedPct = options?.budgetUsedPct;
  if (normalized === 'EXHAUSTED' || (budgetUsedPct != null && budgetUsedPct >= 90)) {
    return 'warning';
  }

  return 'none';
}

export function campaignListRowDataAttributes(
  selected: boolean,
  accent: CampaignListRowAccent = 'none'
): {
  'data-row-selected'?: true;
  'data-row-accent'?: Exclude<CampaignListRowAccent, 'none'>;
} {
  if (selected) {
    return { 'data-row-selected': true };
  }
  if (accent !== 'none') {
    return { 'data-row-accent': accent };
  }
  return {};
}

export function campaignStatusCellClass(status: string, statusTone?: CampaignStatusTone): string {
  const tone = campaignStatusToAdminTone(status, statusTone);
  switch (tone) {
    case 'active':
      return 'bg-admin-status-active/20 text-admin-status-active';
    case 'paused':
      return 'bg-admin-status-paused/20 text-admin-status-paused';
    case 'archived':
      return 'bg-muted/50 text-muted-foreground';
    case 'error':
      return 'bg-destructive/15 text-destructive';
    case 'draft':
      return 'bg-admin-status-draft/20 text-admin-status-draft';
    case 'scheduled':
      return 'bg-admin-status-scheduled/20 text-admin-status-scheduled';
    default:
      return 'bg-muted/40 text-muted-foreground';
  }
}

export function campaignStatusBadgeClass(status: string, statusTone?: CampaignStatusTone): string {
  const tone = campaignStatusToAdminTone(status, statusTone);
  return `${adminStatusBadgeBase} ${adminStatusBadgeClass[tone]}`;
}
