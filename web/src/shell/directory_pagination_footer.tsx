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
  pageSizeLayout?: 'stacked' | 'inline';
  page?: number;
  pageCount?: number;
  onPageChange?: (page: number) => void;
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
  pageSizeLayout = 'stacked',
  page,
  pageCount,
  onPageChange,
  className,
}: DirectoryPaginationFooterProps) {
  const showPageSize =
    onLimitChange != null && pageSizeId != null && limit != null;
  const showPageNumbers =
    page != null && pageCount != null && pageCount > 1 && onPageChange != null;

  return (
    <div className={cn('flex flex-wrap items-center gap-2', className)}>
      {rangeLabel ? (
        <DirectoryListMeta className="shrink-0 text-[13px] tabular-nums text-muted-foreground">
          {rangeLabel}
        </DirectoryListMeta>
      ) : null}
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
          layout={pageSizeLayout}
          value={limit}
          onChange={onLimitChange}
        />
      ) : null}
    </div>
  );
}
