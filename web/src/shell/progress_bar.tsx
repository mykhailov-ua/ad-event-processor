import { cn } from '@/lib/utils';

export type ProgressBarProps = {
  label: string;
  valuePct: number;
  className?: string;
  showValue?: boolean;
};

export function ProgressBar({ label, valuePct, className, showValue = true }: ProgressBarProps) {
  const clamped = Math.max(0, Math.min(100, valuePct));

  return (
    <div className={cn('flex flex-col gap-1.5', className)}>
      <div className="flex items-center justify-between gap-2 text-[13px] leading-[18px]">
        <span className="font-medium text-foreground">{label}</span>
        {showValue ? (
          <span className="font-semibold text-foreground tabular-nums">{clamped.toFixed(0)}%</span>
        ) : null}
      </div>
      <div aria-hidden className="h-2 overflow-hidden rounded-full bg-muted">
        <div
          className="h-full rounded-full bg-primary transition-[width]"
          style={{ width: `${clamped}%` }}
        />
      </div>
    </div>
  );
}
