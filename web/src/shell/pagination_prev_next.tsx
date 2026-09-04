import { toast } from 'sonner';

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
        className={cn(
          '!h-auto min-h-7 rounded-[5px] px-2.5 py-1 text-[13px] leading-[18px]',
          inline ? undefined : 'flex-1',
          !canGoPrev && 'opacity-50',
        )}
        shape={inline ? undefined : 'pill'}
        type="button"
        variant={variant}
        onClick={() => {
          if (disabled) {
            return;
          }
          if (!canGoPrev) {
            toast.message('Already on first page');
            return;
          }
          onPrev();
        }}
      >
        {prevLabel}
      </Button>
      <Button
        aria-label={nextLabel}
        className={cn(
          '!h-auto min-h-7 rounded-[5px] px-2.5 py-1 text-[13px] leading-[18px]',
          inline ? undefined : 'flex-1',
          !canGoNext && 'opacity-50',
        )}
        shape={inline ? undefined : 'pill'}
        type="button"
        variant={variant}
        onClick={() => {
          if (disabled) {
            return;
          }
          if (!canGoNext) {
            toast.message('Already on last page');
            return;
          }
          onNext();
        }}
      >
        {nextLabel}
      </Button>
    </div>
  );
}
