import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';

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
    <DirectoryPaginationFooter
      canGoNext={canGoNext}
      canGoPrev={canGoPrev}
      disabled={disabled}
      nextLabel="Next"
      prevLabel="Prev"
      rangeLabel={summary}
      onNext={onNext}
      onPrev={onPrev}
    />
  );
}
