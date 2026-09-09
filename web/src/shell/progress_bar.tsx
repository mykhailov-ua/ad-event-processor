import { cn } from '@/lib/utils';

export type ProgressBarProps = {
  label: string;
  valuePct: number;
  showValue?: boolean;
};

export function ProgressBar({ label, valuePct, showValue = true }: ProgressBarProps) {
  const clamped = Math.max(0, Math.min(100, valuePct));

  return (
    <div >
      <div >
        <span >{label}</span>
        {showValue ? (
          <span >{clamped.toFixed(0)}%</span>
        ) : null}
      </div>
      <div aria-hidden >
        <div
         
         
        />
      </div>
    </div>
  );
}
