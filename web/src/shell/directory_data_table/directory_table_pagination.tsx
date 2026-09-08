import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

export type DirectoryTablePaginationProps = {
  page: number;
  pageCount: number;
  pageSize: number;
  totalRows: number;
  start: number;
  end: number;
  onPageChange: (page: number) => void;
  className?: string;
  truncatedTotal?: number;
};

export function DirectoryTablePagination({
  page,
  pageCount,
  pageSize,
  totalRows,
  start,
  end,
  onPageChange,
  className,
  truncatedTotal,
}: DirectoryTablePaginationProps) {
  if (totalRows <= pageSize && truncatedTotal == null) {
    return null;
  }

  const visibleStart = totalRows === 0 ? 0 : start + 1;
  const visibleEnd = end;
  const totalLabel = truncatedTotal ?? totalRows;

  return (
    <div
      className={cn(
        'grid grid-cols-[1fr_auto] items-center gap-2 border-t border-border px-3 py-2 text-xs text-muted-foreground',
        className
      )}
    >
      <p>
        {truncatedTotal != null
          ? `Showing ${visibleStart}-${visibleEnd} of top ${totalRows} (${totalLabel} total).`
          : `Showing ${visibleStart}-${visibleEnd} of ${totalRows}.`}
      </p>
      <div className="flex items-center gap-1">
        <Button
          disabled={page <= 0}
          type="button"
          variant="ghost"
          onClick={() => onPageChange(page - 1)}
        >
          Previous
        </Button>
        <span className="px-1">
          {page + 1} / {pageCount}
        </span>
        <Button
          disabled={page + 1 >= pageCount}
          type="button"
          variant="ghost"
          onClick={() => onPageChange(page + 1)}
        >
          Next
        </Button>
      </div>
    </div>
  );
}
