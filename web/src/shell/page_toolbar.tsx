import type { HTMLAttributes, ReactNode } from 'react';

import { cn } from '@/lib/utils';

export type PageToolbarProps = HTMLAttributes<HTMLDivElement> & {
  children: ReactNode;
};

export function PageToolbar({ children, ...props }: PageToolbarProps) {
  return <div {...props}>{children}</div>;
}
