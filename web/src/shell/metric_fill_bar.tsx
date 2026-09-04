import { cn } from '@/lib/utils';

export type MetricFillBarProps = {
  label: string;
  value: string;
  percent: number;
  color: string;
  className?: string;
};

export function MetricFillBar({ label, value, percent, color, className }: MetricFillBarProps) {
  const width = Math.max(0, Math.min(100, percent));

  return (
    <div className={cn('flex flex-col gap-1 grid gap-1', className)}>
      <div className="flex items-center justify-between gap-3 text-xs">
        <span className="text-muted-foreground">{label}</span>
        <span className="tabular-nums text-foreground">{value}</span>
      </div>
      <div aria-hidden className="h-1.5 overflow-hidden rounded-full bg-muted">
        <div
          className="h-full rounded-full transition-[width] duration-200"
          style={{ width: `${width}%`, backgroundColor: color }}
        />
      </div>
    </div>
  );
}
