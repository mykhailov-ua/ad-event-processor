import * as React from 'react';

import { badgeVariantClass, type BadgeVariant } from '@/lib/admin_chrome';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type BadgeProps = React.HTMLAttributes<HTMLDivElement> & {
  variant?: BadgeVariant;
};

function Badge({ className, variant = 'default', ...props }: BadgeProps) {
  return (
    <div
      className={cn(
        'inline-flex items-center rounded-md border text-xs font-medium',
        adminKit.chipPaddingX,
        'py-1',
        badgeVariantClass[variant],
        className
      )}
      {...props}
    />
  );
}

export { Badge };
