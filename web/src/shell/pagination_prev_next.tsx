import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

export type PaginationPrevNextProps = {
  canGoPrev: boolean;
  canGoNext: boolean;
  disabled?: boolean;
  onPrev: () => void;
  onNext: () => void;
  className?: string;
};

export function PaginationPrevNext({
  canGoPrev,
  canGoNext,
  disabled = false,
  onPrev,
  onNext,
  className,
}: PaginationPrevNextProps) {
  return (
    <div className={cn('flex w-full gap-2', className)}>
      <Button
        type="button"
        variant="outline"
        shape="pill"
        className="flex-1 text-sm"
        disabled={disabled || !canGoPrev}
        onClick={onPrev}
      >
        Previous
      </Button>
      <Button
        type="button"
        variant="outline"
        shape="pill"
        className="flex-1 text-sm"
        disabled={disabled || !canGoNext}
        onClick={onNext}
      >
        Next
      </Button>
    </div>
  );
}
