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
