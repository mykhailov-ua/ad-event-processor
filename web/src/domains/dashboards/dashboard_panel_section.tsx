import type { ReactNode } from 'react';

import { BentoSection } from '@/shell/bento_card';
import { adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';
import { DirectoryTable, SECTION_TABLE_HOST_CLASS } from '@/shell/directory_table';

export type DashboardPanelSectionProps = {
  title?: string;
  children: ReactNode;
  footer?: ReactNode;
  'data-testid'?: string;
  tableAriaLabel?: string;
};

export function DashboardPanelSection({
  title,
  children,
  footer,
  'data-testid': testId,
  tableAriaLabel,
}: DashboardPanelSectionProps) {
  const table = (
    <div className={cn(SECTION_TABLE_HOST_CLASS, title ? 'mt-0' : undefined)}>
      <DirectoryTable nested scrollable={false}>
        {children}
      </DirectoryTable>
    </div>
  );

  if (!title) {
    return (
      <section aria-label={tableAriaLabel} className="min-w-0" data-testid={testId}>
        {table}
        {footer ? <div className={adminTypography.bodyMuted}>{footer}</div> : null}
      </section>
    );
  }

  return (
    <section aria-label={tableAriaLabel} className="min-w-0" data-testid={testId}>
      <BentoSection title={title}>
        {table}
        {footer ? <div className={adminTypography.bodyMuted}>{footer}</div> : null}
      </BentoSection>
    </section>
  );
}
