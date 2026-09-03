import * as React from 'react';

import { badgeVariantClass, type BadgeVariant } from '@/lib/admin_chrome';
import { cn } from '@/lib/utils';

export type BadgeProps = React.HTMLAttributes<HTMLDivElement> & {
  variant?: BadgeVariant;
};

function Badge({ className, variant = 'default', ...props }: BadgeProps) {
  return (
    <div
      className={cn(
        'inline-flex items-center rounded-sm border px-2.5 py-0.5 text-xs font-medium',
        badgeVariantClass[variant],
        className,
      )}
      {...props}
    />
  );
}

export { Badge };
