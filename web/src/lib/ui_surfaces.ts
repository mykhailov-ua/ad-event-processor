/** Thin scrollbars; webkit thumb styling lives in tailwind.css. */
import { adminSpacing, adminTypography } from '@/lib/admin_spacing';

export const uiScrollbarClass = 'scrollbar-admin';

const messageBase =
  'grid gap-2 rounded-[8px] border p-4 text-[13px] leading-[18px] shadow-none backdrop-blur-none';

export const uiSurfaces = {
  message: messageBase,
  messageError: `${messageBase} border-destructive/40 bg-destructive/10 text-destructive`,
  messageMuted: `${messageBase} border-border/40 bg-muted/30 text-muted-foreground`,
  messageSuccess: `${messageBase} border-admin-status-active/25 bg-admin-status-active/10 text-admin-positive`,
  messageWarning: `${messageBase} border-admin-warn-border bg-admin-warn-bg text-admin-warn`,
  panel:
    'grid gap-3 rounded-[8px] border border-border bg-card p-4 text-card-foreground shadow-none',
  control:
    'inline-flex h-7 min-h-7 shrink-0 items-center justify-center gap-2 border border-border/40 px-3 text-[13px] leading-none [&_svg]:size-4 [&_svg]:shrink-0',
  toolbarBand: 'flex min-w-0 flex-wrap items-center gap-2',
  toolbarBandSplit: 'flex w-full flex-wrap items-center justify-between gap-2',
  toolbarBandActions: 'flex min-w-0 flex-1 flex-wrap items-center gap-2',
  statusMetricsBand: 'flex flex-wrap items-center gap-4',
  chipRow: 'flex flex-wrap gap-2',
  chip: 'inline-flex min-h-7 max-w-full shrink-0 items-center justify-center gap-1.5 whitespace-nowrap border border-border px-3 py-1 text-xs font-semibold leading-[18px] shadow-none transition-colors',
  chipCount: 'text-[11px] font-semibold',
  summaryBand:
    'inline-flex h-7 max-w-full flex-nowrap items-center gap-2.5 overflow-x-auto border border-primary/20 bg-primary/5 px-3 text-card-foreground',
  summaryBandDivider: 'mx-1 h-3 w-px shrink-0 bg-border',
  tableHost:
    'w-full min-w-0 overflow-hidden rounded-[8px] border border-border bg-card text-card-foreground',
  tableHostFill: 'flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden',
  tableHostScroll: `${uiScrollbarClass} min-w-0 overflow-x-auto bg-card`,
  directoryStack: 'flex w-full flex-col gap-3',
  metaLinksBand: `flex flex-wrap items-center ${adminSpacing.gap.lg} ${adminTypography.bodyMuted} [&_a]:text-primary [&_a:hover]:underline`,
  actionLinksBand: 'flex flex-wrap gap-2',
  filterPanel: 'grid gap-4 rounded-[8px] border border-border bg-card p-4 text-muted-foreground',
} as const;

export type UiMessageSurfaceTone = 'error' | 'muted' | 'success' | 'warning';

const messageToneClass: Record<UiMessageSurfaceTone, string> = {
  error: uiSurfaces.messageError,
  muted: uiSurfaces.messageMuted,
  success: uiSurfaces.messageSuccess,
  warning: uiSurfaces.messageWarning,
};

export function uiMessageSurfaceClass(tone: UiMessageSurfaceTone): string {
  return messageToneClass[tone];
}
