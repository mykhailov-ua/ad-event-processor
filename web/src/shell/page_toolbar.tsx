import type { HTMLAttributes, ReactNode } from 'react';

import { cn } from '@/lib/utils';

export type PageToolbarProps = HTMLAttributes<HTMLDivElement> & {
  children: ReactNode;
};

export function PageToolbar({ children, className, ...props }: PageToolbarProps) {
  return (
    <div className={cn('ui-surface flex flex-wrap items-center gap-3 p-4', className)} {...props}>
      {children}
    </div>
  );
}
