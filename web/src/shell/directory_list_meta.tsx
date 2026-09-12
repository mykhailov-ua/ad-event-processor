import type { ReactNode } from 'react';

import { adminTypography } from '@/lib/admin_spacing';
import { cn } from '@/lib/utils';

export function DirectoryListMeta({ children }: { children: ReactNode }) {
  return <p className={cn('whitespace-nowrap', adminTypography.bodyMuted)}>{children}</p>;
}
