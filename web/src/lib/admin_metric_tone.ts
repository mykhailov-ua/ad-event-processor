/**
 * Metric and ops status colors backed by --admin-* tokens in app.css.
 * Domains import these instead of ad-hoc Tailwind hue families.
 */
export const adminMetricMutedZeroClass = 'tabular-nums text-muted-foreground/60';

export const adminMetricPositiveClass = 'font-semibold text-admin-positive';

export const adminMetricNegativeClass = 'font-semibold text-admin-negative';

export const adminMetricDeltaPositiveClass = 'text-admin-positive';

export const adminMetricDeltaNegativeClass = 'text-admin-negative';

export const adminMetricConversionClass =
  'tabular-nums font-bold text-[hsl(var(--admin-metric-conversion-fg))]';

export const adminMetricApprovedClass = 'tabular-nums font-semibold text-admin-positive';

export const adminOpsHealthyClass = 'text-admin-positive';

export const adminOpsWarnClass = 'text-admin-warn';

export const adminOpsCriticalClass = 'text-admin-negative';

export const adminWizardStepDoneClass =
  'border-admin-status-active/40 bg-admin-status-active/10 text-admin-positive';

export const adminStaleHintClass =
  'inline-flex items-center gap-1 border border-admin-warn-border bg-admin-warn-bg px-1.5 py-0.5 text-[11px] font-semibold leading-[14px] text-admin-warn';

/** Chart-aligned KPI accents (1..5 map to --chart-* tokens). */
export type AdminKpiAccent = 1 | 2 | 3 | 4 | 5;

export const adminKpiAccentValueClass: Record<AdminKpiAccent, string> = {
  1: 'text-chart-1',
  2: 'text-chart-2',
  3: 'text-chart-3',
  4: 'text-chart-4',
  5: 'text-chart-5',
};

export const adminKpiAccentSurfaceClass: Record<AdminKpiAccent, string> = {
  1: 'border-chart-1/35 bg-chart-1/10',
  2: 'border-chart-2/35 bg-chart-2/10',
  3: 'border-chart-3/35 bg-chart-3/10',
  4: 'border-chart-4/35 bg-chart-4/10',
  5: 'border-chart-5/35 bg-chart-5/10',
};

export const adminKpiAccentTopBarClass: Record<AdminKpiAccent, string> = {
  1: 'border-t-2 border-t-chart-1',
  2: 'border-t-2 border-t-chart-2',
  3: 'border-t-2 border-t-chart-3',
  4: 'border-t-2 border-t-chart-4',
  5: 'border-t-2 border-t-chart-5',
};
