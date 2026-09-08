import { X } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type AppliedCustomerBannerProps = {
  customerId: string;
  customerName: string;
  onClear: () => void;
};

export function AppliedCustomerBanner({
  customerId,
  customerName,
  onClear,
}: AppliedCustomerBannerProps) {
  return (
    <div
      className={cn(
        'grid grid-cols-[1fr_auto] items-center gap-2 bg-muted/30 p-2.5 text-sm',
        adminKit.panelRadius
      )}
    >
      <span className="text-muted-foreground">Customer scope</span>
      <span className="font-medium text-foreground">{customerName}</span>
      <span className="text-xs text-muted-foreground">{customerId}</span>
      <Button
        aria-label="Clear customer scope"
        className="gap-1 px-2 text-xs"
        onClick={onClear}
        type="button"
        variant="ghost"
      >
        <X aria-hidden className="h-3.5 w-3.5" />
        Clear
      </Button>
    </div>
  );
}
