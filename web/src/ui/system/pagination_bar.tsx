import { Button } from './button.js';
import styles from './pagination_bar.module.css';

export type PaginationBarProps = {
  limit: number;
  offset: number;
  total: number;
  onOffsetChange: (offset: number) => void;
};

function formatRange(offset: number, limit: number, total: number): string {
  if (total <= 0) return '0 of 0';
  const start = offset + 1;
  const end = Math.min(offset + limit, total);
  return `${start}\u2013${end} of ${total}`;
}

export function PaginationBar({ limit, offset, total, onOffsetChange }: PaginationBarProps) {
  const prevDisabled = offset <= 0;
  const nextDisabled = offset + limit >= total;

  return (
    <nav className={styles.root} aria-label="Pagination">
      <Button
        type="button"
        variant="secondary"
        size="sm"
        disabled={prevDisabled}
        onClick={() => onOffsetChange(Math.max(0, offset - limit))}
      >
        Prev
      </Button>
      <p className={styles.range}>{formatRange(offset, limit, total)}</p>
      <Button
        type="button"
        variant="secondary"
        size="sm"
        disabled={nextDisabled}
        onClick={() => onOffsetChange(offset + limit)}
      >
        Next
      </Button>
    </nav>
  );
}
