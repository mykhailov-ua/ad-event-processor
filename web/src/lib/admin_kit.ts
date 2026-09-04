/**
 * Admin Control Panel UI kit tokens. Primitives import from here or admin_chrome;
 * domains use components/ui and shell, not these strings directly.
 */
export const adminKit = {
  controlRadius: 'rounded-[5px]',
  panelRadius: 'rounded-[10px]',
  pillRadius: 'rounded-full',
  focusRing:
    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
  controlHeight: 'min-h-7',
  controlText: 'text-[13px] leading-[18px]',
  labelCaps: 'text-[11px] font-semibold uppercase leading-[14px] tracking-normal text-muted-foreground',
  tableHeader: 'text-[11px] font-bold uppercase leading-[14px] text-muted-foreground',
  tableRowHeight: 'h-[34px]',
} as const;

export type AdminStatusTone =
  | 'active'
  | 'paused'
  | 'archived'
  | 'error'
  | 'draft'
  | 'scheduled'
  | 'muted';

export const adminStatusBadgeClass: Record<AdminStatusTone, string> = {
  active: 'border-transparent bg-emerald-500/10 text-emerald-500',
  paused: 'border-transparent bg-amber-500/10 text-amber-500',
  archived: 'border-border bg-muted text-muted-foreground',
  error: 'border-transparent bg-destructive/10 text-destructive',
  draft: 'border-transparent bg-sky-500/10 text-sky-500',
  scheduled: 'border-transparent bg-orange-500/10 text-orange-500',
  muted: 'border-border bg-muted/50 text-muted-foreground',
};

export const adminStatusBadgeBase =
  'inline-flex max-w-full shrink-0 items-center whitespace-nowrap rounded-full border px-2 py-0.5 text-xs font-medium leading-4';

export type AdminAlertTone = 'success' | 'error' | 'warning';

export const adminAlertClass: Record<AdminAlertTone, string> = {
  success: 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900/50 dark:bg-emerald-950/20 dark:text-emerald-400',
  error: 'border-destructive/20 bg-destructive/10 text-destructive',
  warning: 'border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/20 dark:text-amber-400',
};

export function campaignStatusToAdminTone(
  status: string,
  statusTone?: 'success' | 'warning' | 'muted' | string,
): AdminStatusTone {
  if (statusTone === 'success') {
    return 'active';
  }
  if (statusTone === 'warning') {
    return 'paused';
  }
  if (statusTone === 'muted') {
    return 'archived';
  }
  switch (status.trim().toUpperCase()) {
    case 'ACTIVE':
      return 'active';
    case 'PAUSED':
      return 'paused';
    case 'ARCHIVED':
      return 'archived';
    case 'DRAFT':
      return 'draft';
    case 'SCHEDULED':
      return 'scheduled';
    case 'ERROR':
    case 'FAILED':
      return 'error';
    default:
      return 'muted';
  }
}
