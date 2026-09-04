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
    <div className={cn('admin-progress', className)}>
      <div className="admin-progress__header">
        <span className="admin-progress__label">{label}</span>
        {showValue ? <span className="admin-progress__value tabular-nums">{clamped.toFixed(0)}%</span> : null}
      </div>
      <div aria-hidden className="admin-progress__track">
        <div className="admin-progress__fill" style={{ width: `${clamped}%` }} />
      </div>
    </div>
  );
}
