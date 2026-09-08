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

export function resolveCampaignListRowAccent(
  status: string,
  statusTone?: CampaignStatusTone,
  options?: {
    budgetUsedPct?: number | null;
    marginBreach?: boolean;
  }
): CampaignListRowAccent {
  const statusKey = resolveCampaignStatusKey(status, statusTone);
  if (statusKey === 'PAUSED' || statusKey === 'ARCHIVED') {
    return 'muted';
  }

  const normalized = status.trim().toUpperCase();
  if (
    options?.marginBreach === true ||
    normalized === 'ERROR' ||
    normalized === 'FAILED'
  ) {
    return 'critical';
  }

  const budgetUsedPct = options?.budgetUsedPct;
  if (normalized === 'EXHAUSTED' || (budgetUsedPct != null && budgetUsedPct >= 90)) {
    return 'warning';
  }

  return 'none';
}

export function campaignListRowClass(
  selected: boolean,
  accent: CampaignListRowAccent = 'none'
): string {
  const tone: string[] = [];
  if (selected) {
    tone.push(
      '[&>td:not([data-col-pin])]:!bg-primary/22',
      'dark:[&>td:not([data-col-pin])]:!bg-primary/30',
      '[&>td:first-child]:shadow-[inset_3px_0_0_0_hsl(var(--primary))]',
      'hover:[&>td:not([data-col-pin])]:!bg-primary/26',
      'dark:hover:[&>td:not([data-col-pin])]:!bg-primary/34'
    );
    return tone.join(' ');
  }

  switch (accent) {
    case 'muted':
      tone.push(
        '[&>td:not([data-col-pin])]:!bg-muted/30',
        'even:[&>td:not([data-col-pin])]:!bg-muted/38',
        'hover:[&>td:not([data-col-pin])]:!bg-muted/42'
      );
      break;
    case 'warning':
      tone.push(
        '[&>td:not([data-col-pin])]:!bg-admin-warn-bg/80',
        'hover:[&>td:not([data-col-pin])]:!bg-admin-warn-bg'
      );
      break;
    case 'critical':
      tone.push(
        '[&>td:not([data-col-pin])]:!bg-destructive/12',
        'dark:[&>td:not([data-col-pin])]:!bg-destructive/18',
        'hover:[&>td:not([data-col-pin])]:!bg-destructive/16',
        'dark:hover:[&>td:not([data-col-pin])]:!bg-destructive/22'
      );
      break;
    default:
      tone.push(
        'odd:[&_td]:bg-card',
        'even:[&_td]:bg-muted/25',
        'odd:[&_td[data-col-pin]]:bg-admin-table-pin',
        'even:[&_td[data-col-pin]]:bg-admin-table-pin',
        'hover:[&_td]:bg-admin-table-hover/70',
        'hover:[&_td[data-col-pin]]:bg-admin-table-hover'
      );
  }
  return tone.join(' ');
}

export function campaignStatusCellClass(
  status: string,
  statusTone?: CampaignStatusTone
): string {
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
