/**
 * Admin Control Panel UI kit tokens. Primitives import from here or admin_chrome;
 * domains use components/ui and shell, not these strings directly.
 */
/** Radius scale: preview flat corners (--radius 0). */
export const adminKit = {
  controlRadius: 'rounded-none',
  panelRadius: 'rounded-none',
  pillRadius: 'rounded-none',
  /** Horizontal inset for controls; square corners need more than pill-era px-2. */
  controlPaddingX: 'px-3',
  /** Label-to-control gap in filter fields and date pickers. */
  fieldLabelGap: 'gap-2',
  /** Toggle/status chip horizontal padding. */
  chipPaddingX: 'px-3',
  chipInnerGap: 'gap-1.5',
  /** Compact summary / metric band inset. */
  compactInsetX: 'px-3',
  focusRing:
    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
  controlHeight: 'min-h-7',
  controlBorder: 'border border-border',
  controlText: 'text-[13px] leading-[18px]',
  /** Form field labels (campaign filter rows, editor fields). */
  fieldLabelClass: 'text-[13px] font-medium leading-[18px] text-foreground',
  /** Directory table header cell inner shell (campaign list parity). */
  directoryTableHeadInnerClass:
    'flex w-full items-center gap-1.5 px-4 text-[11px] font-semibold uppercase leading-[14px] tracking-normal text-muted-foreground',
  buttonShell: 'ui-control-surface',
  labelCaps:
    'text-[11px] font-semibold uppercase leading-[14px] tracking-normal text-muted-foreground',
  tableHeader: 'text-[11px] font-semibold uppercase leading-[14px] text-muted-foreground',
  tableRowHeight: 'h-[34px]',
  /** Error surfaces: flat alpha tint; no backdrop-filter (avoids scroll jank). */
  errorSurface: 'ui-message-surface ui-message-surface-error',
  toastSurface:
    'border-border/60 bg-card/90 text-card-foreground shadow-md shadow-black/10 backdrop-blur-none',
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
  'inline-flex max-w-full shrink-0 items-center whitespace-nowrap rounded-none border border-border px-2.5 py-0.5 text-xs font-normal leading-4';

export type AdminAlertTone = 'success' | 'error' | 'warning';

export const adminAlertClass: Record<AdminAlertTone, string> = {
  success: 'ui-message-surface-success',
  error: 'ui-message-surface-error',
  warning: 'ui-message-surface-warning',
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
