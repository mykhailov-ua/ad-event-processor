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
        adminKit.controlRadius,
        'inline-flex items-center border px-2.5 py-0.5 text-xs font-normal leading-4',
        badgeVariantClass[variant],
        className
      )}
      {...props}
    />
  );
}

export { Badge };
