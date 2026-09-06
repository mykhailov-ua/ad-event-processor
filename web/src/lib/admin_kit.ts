/**
 * Admin Control Panel UI kit tokens. Primitives import from here or admin_chrome;
 * domains use components/ui and shell, not these strings directly.
 */
/** Radius scale: rounded-sm/md from --radius (4px / 6px). See ui.mdc Corners. */
export const adminKit = {
  controlRadius: 'rounded-sm',
  panelRadius: 'rounded-md',
  pillRadius: 'rounded-full',
  focusRing:
    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
  controlHeight: 'min-h-7',
  controlText: 'text-[13px] leading-[18px]',
  buttonShell:
    'inline-flex h-7 shrink-0 items-center justify-center gap-2 py-0 text-[13px] leading-none [&_svg]:block [&_svg]:shrink-0',
  labelCaps:
    'text-[11px] font-semibold uppercase leading-[14px] tracking-normal text-muted-foreground',
  tableHeader: 'text-[11px] font-semibold uppercase leading-[14px] text-muted-foreground',
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
  active: 'border-transparent bg-admin-status-active/10 text-admin-status-active',
  paused: 'border-transparent bg-admin-status-paused/10 text-admin-status-paused',
  archived: 'border-border bg-muted text-muted-foreground',
  error: 'border-transparent bg-destructive/10 text-destructive',
  draft: 'border-transparent bg-admin-status-draft/10 text-admin-status-draft',
  scheduled: 'border-transparent bg-admin-status-scheduled/10 text-admin-status-scheduled',
  muted: 'border-border bg-muted/50 text-muted-foreground',
};

export const adminStatusBadgeBase =
  'inline-flex max-w-full shrink-0 items-center whitespace-nowrap rounded-full border px-2 py-0.5 text-xs font-normal leading-4';

export type AdminAlertTone = 'success' | 'error' | 'warning';

export const adminAlertClass: Record<AdminAlertTone, string> = {
  success: 'border-admin-status-active/25 bg-admin-status-active/10 text-admin-positive',
  error: 'border-destructive/20 bg-destructive/10 text-destructive',
  warning: 'border-admin-warn-border bg-admin-warn-bg text-admin-warn',
};

export function campaignStatusToAdminTone(
  status: string,
  statusTone?: 'success' | 'warning' | 'muted' | string
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
