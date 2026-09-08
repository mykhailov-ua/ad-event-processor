import { cn } from '@/lib/utils';

export type MetricFillBarProps = {
  label: string;
  value: string;
  percent: number;
  /** Tailwind fill class (e.g. CHART_SWATCH_CLASS token or bg-primary). CSP-safe: no inline backgroundColor. */
  fillClassName: string;
  className?: string;
};

export function MetricFillBar({
  label,
  value,
  percent,
  fillClassName,
  className,
}: MetricFillBarProps) {
  const width = Math.max(0, Math.min(100, percent));

  return (
    <div className={cn('flex flex-col gap-1', className)}>
      <div className="grid grid-cols-[1fr_auto] items-center gap-3 text-xs">
        <span className="text-muted-foreground">{label}</span>
        <span className="text-foreground">{value}</span>
      </div>
      <div aria-hidden className="h-1.5 overflow-hidden rounded-full bg-muted">
        <div
          className={cn('h-full rounded-full transition-[width] duration-200', fillClassName)}
          style={{ width: `${width}%` }}
        />
      </div>
    </div>
  );
}
