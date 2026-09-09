import { adminSpacing, adminTypography } from '@/lib/admin_spacing';
import { uiSurfaces } from '@/lib/ui_surfaces';

/**
 * Admin Control Plane UI kit tokens. Primitives import from here or admin_chrome;
 * domains use components/ui and shell, not these strings directly.
 */
export const adminKit = {
  controlRadius: 'rounded-[8px]',
  panelRadius: 'rounded-[8px]',
  /** Nested chips, menu rows, and insets inside panelRadius / controlRadius shells. */
  nestedRadius: 'rounded-[4px]',
  pillRadius: 'rounded-full',
  /** Horizontal inset for controls; square corners need more than pill-era px-2. */
  controlPaddingX: 'px-3',
  /** Label-to-control gap in filter fields and date pickers. */
  fieldLabelGap: adminSpacing.gap.md,
  /** Toggle/status chip horizontal padding. */
  chipPaddingX: 'px-3',
  chipInnerGap: adminSpacing.gap.sm,
  /** Compact summary / metric band inset. */
  compactInsetX: 'px-3',
  focusRing: 'focus-visible:outline-none focus-visible:ring-0 focus-visible:ring-offset-0',
  controlHeight: 'h-7 min-h-7',
  controlBorder: 'border border-border/40',
  controlText: adminTypography.body,
  /** Form field labels (campaign filter rows, editor fields). */
  fieldLabelClass: adminTypography.label,
  /** Directory table header cell inner shell (campaign list parity). */
  directoryTableHeadInnerClass: `flex w-full items-center ${adminSpacing.gap.sm} ${adminSpacing.inset.tableCellX} ${adminTypography.tableHeader}`,
  buttonShell: uiSurfaces.control,
  labelCaps: adminTypography.caption,
  tableHeader: adminTypography.tableHeader,
  tableRowHeight: 'h-[34px]',
  /** Error surfaces: flat alpha tint; no backdrop-filter (avoids scroll jank). */
  errorSurface: uiSurfaces.messageError,
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

export const adminStatusBadgeBase = `inline-flex max-w-full shrink-0 items-center whitespace-nowrap rounded-md border border-border px-2.5 py-0.5 ${adminTypography.badge} font-normal`;

export type AdminAlertTone = 'success' | 'error' | 'warning';

export const adminAlertClass: Record<AdminAlertTone, string> = {
  success: uiSurfaces.messageSuccess,
  error: uiSurfaces.messageError,
  warning: uiSurfaces.messageWarning,
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

export {
  adminSpacing,
  adminTypography,
  customerDetailSectionClass,
} from '@/lib/admin_spacing';
