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
  layout = 'split',
}: PaginationPrevNextProps) {
  const inline = layout === 'inline';

  return (
    <div className={cn(inline ? 'flex items-center gap-2' : 'flex w-full gap-2', className)}>
      <Button
        aria-label={prevLabel}
        className={cn('text-sm', inline ? undefined : 'flex-1')}
        disabled={disabled || !canGoPrev}
        shape={inline ? undefined : 'pill'}
        type="button"
        variant={variant}
        onClick={onPrev}
      >
        {prevLabel}
      </Button>
      <Button
        aria-label={nextLabel}
        className={cn('text-sm', inline ? undefined : 'flex-1')}
        disabled={disabled || !canGoNext}
        shape={inline ? undefined : 'pill'}
        type="button"
        variant={variant}
        onClick={onNext}
      >
        {nextLabel}
      </Button>
    </div>
  );
}
