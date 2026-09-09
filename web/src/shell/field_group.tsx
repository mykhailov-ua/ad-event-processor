import type { ReactNode } from 'react';

import { adminChrome } from '@/lib/admin_chrome';
import { adminSpacing, adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type FieldGroupProps = {
  legend: ReactNode;
  disabled?: boolean;
  children: ReactNode;
  className?: string;
};

export function FieldGroup({ legend, disabled, children, className }: FieldGroupProps) {
  return (
    <div
      aria-disabled={disabled || undefined}
      className={cn(
        adminChrome.panel,
        adminSpacing.inset.panel,
        'grid',
        adminSpacing.gap.md,
        disabled && 'pointer-events-none opacity-60',
        className
      )}
    >
      <div className={adminTypography.label}>{legend}</div>
      {children}
    </div>
  );
}
