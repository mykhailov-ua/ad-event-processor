import { X } from 'lucide-react';

import { Button } from '@/components/ui/button';

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
    <div className="flex flex-wrap items-center gap-2 rounded-2xl bg-muted/30 px-4 py-2.5 text-sm">
      <span className="text-muted-foreground">Customer scope</span>
      <span className="font-medium text-foreground">{customerName}</span>
      <span className="font-mono text-xs text-muted-foreground">{customerId}</span>
      <Button
        aria-label="Clear customer scope"
        className="ml-auto gap-1 px-2 text-xs"
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
