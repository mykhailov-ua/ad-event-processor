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
     
    >
      <span >Customer scope</span>
      <span >{customerName}</span>
      <span >{customerId}</span>
      <Button
        aria-label="Clear customer scope"
       
        onClick={onClear}
        type="button"
        variant="ghost"
      >
        <X aria-hidden  />
        Clear
      </Button>
    </div>
  );
}
