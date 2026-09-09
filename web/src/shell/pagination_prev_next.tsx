import { Button } from '@/components/ui/button';
import type { ButtonVariant } from '@/lib/admin_chrome';
import { cn } from '@/lib/utils';

export type PaginationPrevNextProps = {
  canGoPrev: boolean;
  canGoNext: boolean;
  disabled?: boolean;
  onPrev: () => void;
  onNext: () => void;
  className?: string;
  variant?: ButtonVariant;
  prevLabel?: string;
  nextLabel?: string;
  layout?: 'split' | 'inline';
};

export function PaginationPrevNext({
  canGoPrev,
  canGoNext,
  disabled = false,
  onPrev,
  onNext,
  className,
  variant = 'outline',
  prevLabel = 'Previous',
  nextLabel = 'Next',
  layout = 'inline',
}: PaginationPrevNextProps) {
  const split = layout === 'split';
  const prevDisabled = disabled || !canGoPrev;
  const nextDisabled = disabled || !canGoNext;

  return (
    <div className={cn(split ? 'flex w-full gap-2' : 'flex items-center gap-2', className)}>
      <Button
        aria-label={prevLabel}
        className={split ? 'flex-1' : undefined}
        disabled={prevDisabled}
        shape={split ? 'pill' : undefined}
        type="button"
        variant={variant}
        onClick={onPrev}
      >
        {prevLabel}
      </Button>
      <Button
        aria-label={nextLabel}
        className={split ? 'flex-1' : undefined}
        disabled={nextDisabled}
        shape={split ? 'pill' : undefined}
        type="button"
        variant={variant}
        onClick={onNext}
      >
        {nextLabel}
      </Button>
    </div>
  );
}
