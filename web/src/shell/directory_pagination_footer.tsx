import type { ButtonVariant } from '@/lib/admin_chrome';
import { cn } from '@/lib/utils';
import { DirectoryListMeta } from '@/shell/directory_list_meta';
import { PaginationPageSize } from '@/shell/pagination_page_size';
import { PaginationPrevNext } from '@/shell/pagination_prev_next';

export type DirectoryPaginationFooterProps = {
  canGoPrev: boolean;
  canGoNext: boolean;
  disabled?: boolean;
  onPrev: () => void;
  onNext: () => void;
  rangeLabel?: string;
  pageSizeId?: string;
  limit?: number;
  onLimitChange?: (limit: number) => void;
  prevLabel?: string;
  nextLabel?: string;
  variant?: ButtonVariant;
  layout?: 'inline' | 'split';
  className?: string;
};

export function DirectoryPaginationFooter({
  canGoPrev,
  canGoNext,
  disabled = false,
  onPrev,
  onNext,
  rangeLabel,
  pageSizeId,
  limit,
  onLimitChange,
  prevLabel = 'Previous',
  nextLabel = 'Next',
  variant = 'secondary',
  layout = 'inline',
  className,
}: DirectoryPaginationFooterProps) {
  const showPageSize =
    onLimitChange != null && pageSizeId != null && limit != null;

  return (
    <div className={cn('flex flex-wrap items-center gap-2', className)}>
      <PaginationPrevNext
        canGoNext={canGoNext}
        canGoPrev={canGoPrev}
        disabled={disabled}
        layout={layout}
        nextLabel={nextLabel}
        prevLabel={prevLabel}
        variant={variant}
        onNext={onNext}
        onPrev={onPrev}
      />
      {showPageSize ? (
        <PaginationPageSize
          disabled={disabled}
          id={pageSizeId}
          value={limit}
          onChange={onLimitChange}
        />
      ) : null}
      {rangeLabel ? (
        <DirectoryListMeta className="tabular-nums">{rangeLabel}</DirectoryListMeta>
      ) : null}
    </div>
  );
}
