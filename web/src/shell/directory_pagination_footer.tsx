import type { ButtonVariant } from '@/lib/admin_chrome';
import { cn } from '@/lib/utils';
import { DirectoryListMeta } from '@/shell/directory_list_meta';
import { PaginationPageSize } from '@/shell/pagination_page_size';
import { PaginationPages } from '@/shell/pagination_pages';
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
  page?: number;
  pageCount?: number;
  onPageChange?: (page: number) => void;
  showPrevNext?: boolean;
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
  page,
  pageCount,
  onPageChange,
  showPrevNext = true,
  className,
}: DirectoryPaginationFooterProps) {
  const showPageSize = onLimitChange != null && pageSizeId != null && limit != null;
  const showPageNumbers =
    page != null && pageCount != null && pageCount > 1 && onPageChange != null;

  return (
    <div className={cn('flex flex-wrap items-center gap-3', className)}>
      {rangeLabel ? (
        <DirectoryListMeta className="shrink-0 text-[13px] tabular-nums text-muted-foreground">
          {rangeLabel}
        </DirectoryListMeta>
      ) : null}
      {showPrevNext ? (
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
      ) : null}
      {showPageNumbers ? (
        <PaginationPages
          disabled={disabled}
          page={page}
          pageCount={pageCount}
          onPageChange={onPageChange}
        />
      ) : null}
      {showPageSize ? (
        <PaginationPageSize
          disabled={disabled}
          id={pageSizeId}
          value={limit}
          onChange={onLimitChange}
        />
      ) : null}
    </div>
  );
}
