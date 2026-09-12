import { RefreshCw } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { useLiveRelativeTimestamp } from '@/hooks/use_live_relative_timestamp';
import { displayTimestamp } from '@/lib/display';
import { adminSpacing, adminTypography } from '@/lib/admin_spacing';
import { cn } from '@/lib/utils';

export type ListRefreshBandProps = {
  ariaLabel: string;
  disabled?: boolean;
  lastUpdatedAt?: string | null;
  loading?: boolean;
  onRefresh: () => void;
  title?: string;
};

export function ListRefreshBand({
  ariaLabel,
  disabled = false,
  lastUpdatedAt,
  loading = false,
  onRefresh,
  title,
}: ListRefreshBandProps) {
  const updatedLabel = useLiveRelativeTimestamp(lastUpdatedAt);
  const statusText = updatedLabel && lastUpdatedAt ? `Updated ${updatedLabel}` : 'Not loaded yet';
  const busy = loading || disabled;

  return (
    <div
      aria-label="List refresh"
      className={cn('flex shrink-0 items-center', adminSpacing.gap.md)}
      role="group"
    >
      <span
        className={cn('whitespace-nowrap', adminTypography.bodyMuted)}
        title={lastUpdatedAt ? `Last updated ${displayTimestamp(lastUpdatedAt)}` : undefined}
      >
        {statusText}
      </span>
      <Button
        className={cn('size-7 shrink-0 p-0', busy && 'pointer-events-none')}
        aria-busy={loading || undefined}
        aria-label={ariaLabel}
        disabled={busy}
        title={title ?? ariaLabel}
        type="button"
        variant="outline"
        onClick={() => {
          if (busy) {
            return;
          }
          onRefresh();
        }}
      >
        <RefreshCw className={cn('h-4 w-4 shrink-0', loading && 'animate-spin')} aria-hidden />
      </Button>
    </div>
  );
}
