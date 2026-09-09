import { cn } from '@/lib/utils';

export type MetricFillBarProps = {
  label: string;
  value: string;
  percent: number;
  /** Tailwind fill class (e.g. CHART_SWATCH_CLASS token or bg-primary). CSP-safe: no inline backgroundColor. */
  fillClassName: string;
};

export function MetricFillBar({
  label,
  value,
  percent,
  fillClassName,
}: MetricFillBarProps) {
  const width = Math.max(0, Math.min(100, percent));

  return (
    <div >
      <div >
        <span >{label}</span>
        <span >{value}</span>
      </div>
      <div aria-hidden >
        <div
         
         
        />
      </div>
    </div>
  );
}
