import { Button } from '@/components/ui/button';

export function OpsListFooter({
  summary,
  canGoPrev,
  canGoNext,
  disabled = false,
  onPrev,
  onNext,
}: {
  summary?: string;
  canGoPrev: boolean;
  canGoNext: boolean;
  disabled?: boolean;
  onPrev: () => void;
  onNext: () => void;
}) {
  return (
    <div className="admin-footer-bar">
      <div aria-label="Pagination" className="admin-footer-pagination">
        <Button
          aria-label="Previous page"
          disabled={disabled || !canGoPrev}
          type="button"
          variant="secondary"
          onClick={onPrev}
        >
          Prev
        </Button>
        <Button
          aria-label="Next page"
          disabled={disabled || !canGoNext}
          type="button"
          variant="secondary"
          onClick={onNext}
        >
          Next
        </Button>
        {summary ? <span className="admin-muted tabular-nums">{summary}</span> : null}
      </div>
    </div>
  );
}
