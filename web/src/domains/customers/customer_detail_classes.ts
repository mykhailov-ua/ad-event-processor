import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

// Full-width inset panel for read-only rows and label|field edit rows (Wallet, Tax, Profile).
// Page-level filter/edit width caps: shell/filter_panel.tsx. Metric/status colors:
// lib/admin_metric_tone.ts and --admin-* in styles/app.css. Do not put max-w-* on inner form
// inside a full-width Card; cap the Card or use CustomerDetailPanel rows.
export const customerDetailPanelClass = cn(
  'min-w-0 divide-y divide-border overflow-hidden border border-border bg-muted/10',
  adminKit.panelRadius
);

export const customerDetailRowClass =
  'grid min-h-[34px] gap-1 px-3 py-2 sm:grid-cols-[10rem_minmax(0,1fr)] sm:items-center sm:gap-4';

export const customerDetailRowLabelClass = 'text-[13px] leading-[18px] text-muted-foreground';

export const customerDetailRowValueClass =
  'min-w-0 text-[13px] leading-[18px] text-foreground break-words';
