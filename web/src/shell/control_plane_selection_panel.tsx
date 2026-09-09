import type { ReactNode } from 'react';

import { adminChrome } from '@/lib/admin_chrome';
import { adminSpacing, adminTypography } from '@/lib/admin_spacing';
import { cn } from '@/lib/utils';

export type ControlPlaneSelectionPanelProps = {
  title?: string;
  selectedTitle?: ReactNode;
  emptyHint: string;
  children?: ReactNode;
  meta?: ReactNode;
  className?: string;
};

export function ControlPlaneSelectionPanel({
  title = 'Control panel',
  selectedTitle,
  emptyHint,
  children,
  meta,
  className,
}: ControlPlaneSelectionPanelProps) {
  const hasSelection = selectedTitle != null && selectedTitle !== '';

  return (
    <section
      aria-label={title}
      className={cn(
        adminChrome.panel,
        `flex min-h-[12rem] w-full ${adminSpacing.flex.columnLg} ${adminSpacing.inset.panel} lg:sticky lg:top-4`,
        className
      )}
    >
      <h2 className={adminTypography.panelTitle}>{title}</h2>
      {hasSelection ? (
        <p className={cn('whitespace-normal', adminTypography.body)}>{selectedTitle}</p>
      ) : (
        <p className={cn('whitespace-normal', adminTypography.bodyMuted)}>{emptyHint}</p>
      )}
      {meta ? (
        <div className={cn('whitespace-normal', adminTypography.bodyMuted)}>{meta}</div>
      ) : null}
      <div
        aria-disabled={!hasSelection}
        className={cn(
          adminSpacing.flex.columnMd,
          !hasSelection && 'pointer-events-none opacity-50'
        )}
        role="toolbar"
      >
        {children}
      </div>
    </section>
  );
}
