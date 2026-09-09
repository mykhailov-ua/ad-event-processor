import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

export type PaginationPagesProps = {
  page: number;
  pageCount: number;
  disabled?: boolean;
  onPageChange: (page: number) => void;
  maxVisible?: number;
};

function pageRange(page: number, pageCount: number, maxVisible: number): number[] {
  if (pageCount <= maxVisible) {
    return Array.from({ length: pageCount }, (_, index) => index + 1);
  }
  const half = Math.floor(maxVisible / 2);
  let start = Math.max(1, page - half);
  const end = Math.min(pageCount, start + maxVisible - 1);
  start = Math.max(1, end - maxVisible + 1);
  return Array.from({ length: end - start + 1 }, (_, index) => start + index);
}

export function PaginationPages({
  page,
  pageCount,
  disabled = false,
  onPageChange,
  maxVisible = 5,
}: PaginationPagesProps) {
  if (pageCount <= 1) {
    return null;
  }

  const pages = pageRange(page, pageCount, maxVisible);

  return (
    <div aria-label="Pagination" className="flex flex-wrap items-center gap-2">
      {pages.map((pageNumber) => {
        const active = pageNumber === page;
        return (
          <Button
            key={pageNumber}
            aria-current={active ? 'page' : undefined}
            aria-label={`Page ${pageNumber}`}
           
            disabled={disabled}
            type="button"
            variant={active ? 'default' : 'outline'}
            onClick={() => onPageChange(pageNumber)}
          >
            {pageNumber}
          </Button>
        );
      })}
    </div>
  );
}
