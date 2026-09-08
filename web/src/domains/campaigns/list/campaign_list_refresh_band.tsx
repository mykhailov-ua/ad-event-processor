import { RefreshCw } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { displayRelativeTimestamp, displayTimestamp } from '@/lib/display';
import { cn } from '@/lib/utils';

export type CampaignListRefreshBandProps = {
  className?: string;
  disabled?: boolean;
  lastUpdatedAt?: string | null;
  loading?: boolean;
  onRefresh: () => void;
};

export function CampaignListRefreshBand({
  className,
  disabled = false,
  lastUpdatedAt,
  loading = false,
  onRefresh,
}: CampaignListRefreshBandProps) {
  const updatedLabel = lastUpdatedAt
    ? displayRelativeTimestamp(lastUpdatedAt)
    : 'Not loaded yet';
  const statusText =
    updatedLabel === 'Not loaded yet' ? updatedLabel : `Updated ${updatedLabel}`;
  const busy = loading || disabled;

  return (
    <div
      aria-label="List refresh"
      className={cn('flex shrink-0 flex-nowrap items-center gap-2', className)}
      role="group"
    >
      <span
        className="shrink-0 whitespace-nowrap text-right text-[11px] leading-none text-muted-foreground"
        title={lastUpdatedAt ? `Last updated ${displayTimestamp(lastUpdatedAt)}` : undefined}
      >
        {statusText}
      </span>
      <Button
        aria-busy={loading || undefined}
        aria-label="Refresh campaign list"
        className={cn('size-7 shrink-0 p-0', busy && 'pointer-events-none')}
        title="Refresh campaign list"
        type="button"
        variant="outline"
        onClick={() => {
          if (busy) {
            return;
          }
          onRefresh();
        }}
      >
        <RefreshCw
          aria-hidden
          className={cn('h-4 w-4 shrink-0', loading && 'animate-spin')}
        />
      </Button>
    </div>
  );
}
