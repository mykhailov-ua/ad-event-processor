import type { ReactNode } from 'react';

import { adminSpacing, adminTypography } from '@/lib/admin_spacing';
import { productDisplayName } from '@/lib/product_display_name';
import { ProductAvatar } from '@/shell/product_avatar';
import { cn } from '@/lib/utils';

export function AuthPageLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-background p-4 sm:p-6">
      <div className={cn('mb-6 flex flex-col items-center', adminSpacing.gap.md)}>
        <ProductAvatar framed size="lg" />
        <span className={cn('whitespace-nowrap tracking-tight', adminTypography.sectionTitle)}>
          {productDisplayName}
        </span>
      </div>
      <div className="w-full max-w-sm">{children}</div>
    </div>
  );
}
