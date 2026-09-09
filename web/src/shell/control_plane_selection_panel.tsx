import type { ReactNode } from 'react';

import { adminChrome } from '@/lib/admin_chrome';
import { adminSpacing, adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type ControlPlaneSelectionPanelProps = {
  title?: string;
  selectedTitle?: ReactNode;
  emptyHint: string;
  children?: ReactNode;
  meta?: ReactNode;
  className?: string;
  /** Sidebar: sticky aside. Toolbar: card above table. Inline: row inside filter panel footer. */
  variant?: 'sidebar' | 'toolbar' | 'inline';
};

export function ControlPlaneSelectionPanel({
  title = 'Control panel',
  selectedTitle,
  emptyHint,
  children,
  meta,
  className,
  variant = 'sidebar',
}: ControlPlaneSelectionPanelProps) {
  const hasSelection = selectedTitle != null && selectedTitle !== '';
  const isToolbar = variant === 'toolbar';
  const isInline = variant === 'inline';

  if (isInline) {
    return (
      <div
        aria-label={title}
        className={cn('flex min-w-0 flex-wrap items-center gap-2', className)}
      >
        {children ? (
          <div
            aria-disabled={!hasSelection}
            className={cn(
              'flex min-w-0 flex-wrap items-center gap-2',
              !hasSelection && 'pointer-events-none opacity-50'
            )}
            role="toolbar"
          >
            {hasSelection ? (
              <>
                <span className={cn('max-w-[14rem] truncate', adminTypography.body)}>
                  {selectedTitle}
                </span>
                {meta}
                <span aria-hidden className="hidden h-4 w-px shrink-0 bg-border sm:inline" />
              </>
            ) : (
              <span className={cn('min-w-0', adminTypography.bodyMuted)}>{emptyHint}</span>
            )}
            <div className={cn(adminSpacing.flex.buttonGroup, 'flex-wrap')}>{children}</div>
          </div>
        ) : null}
      </div>
    );
  }

  return (
    <section
      aria-label={title}
      className={cn(
        adminChrome.panel,
        isToolbar
          ? cn('w-full', adminSpacing.flex.columnMd, adminSpacing.inset.panel)
          : cn(
              `flex min-h-[12rem] w-full ${adminSpacing.flex.columnLg} ${adminSpacing.inset.panel} lg:sticky lg:top-4`
            ),
        className
      )}
    >
      <div
        className={cn(
          isToolbar && 'grid grid-cols-[1fr_auto] items-start gap-2',
          !isToolbar && adminSpacing.flex.columnMd
        )}
      >
        <div className={isToolbar ? adminSpacing.stack.titleBlock : adminSpacing.flex.columnMd}>
          <h2 className={adminTypography.panelTitle}>{title}</h2>
          {hasSelection ? (
            <p className={cn('whitespace-normal', adminTypography.body)}>{selectedTitle}</p>
          ) : (
            <p className={cn('whitespace-normal', adminTypography.bodyMuted)}>{emptyHint}</p>
          )}
        </div>
        {meta ? (
          <div className={cn('whitespace-normal shrink-0', adminTypography.bodyMuted)}>{meta}</div>
        ) : null}
      </div>
      <div
        aria-disabled={!hasSelection}
        className={cn(
          isToolbar ? cn(adminSpacing.flex.buttonGroup, 'flex-wrap') : adminSpacing.flex.columnMd,
          !hasSelection && 'pointer-events-none opacity-50'
        )}
        role="toolbar"
      >
        {children}
      </div>
    </section>
  );
}
