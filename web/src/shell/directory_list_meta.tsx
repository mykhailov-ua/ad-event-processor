import type { ReactNode } from 'react';

import { cn } from '@/lib/utils';

export function DirectoryListMeta({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return <p className={cn('text-[13px] leading-[18px] text-muted-foreground', className)}>{children}</p>;
}
