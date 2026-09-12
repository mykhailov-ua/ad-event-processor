import type { ReactNode } from 'react';

import {
  adminSpacing,
  adminTypography,
  pageCanvasInsetClass,
  pageFooterFlatClass,
  pageSectionStackClass,
  pageWorkspaceFlatClass,
} from '@/lib/admin_spacing';
import { cn } from '@/lib/utils';

export { pageCanvasInsetClass, pageFooterFlatClass, pageSectionStackClass, pageWorkspaceFlatClass };

/** Min height for directory/dashboard pages inside the scroll canvas (see tailwind.css vars). */
export const pageFillViewportMinHeightClass = 'min-h-[var(--page-fill-min-height)]';

/** Fill viewport below page header for directory tables and dashboards. */
export const pageWorkspaceFillClass = cn(pageWorkspaceFlatClass, pageFillViewportMinHeightClass);

/** Footer implies directory fill unless fillViewport is explicitly false. */
export function resolvePageFillViewport(
  fillViewport: boolean | undefined,
  footer: ReactNode | undefined
): boolean {
  if (fillViewport === true) {
    return true;
  }
  if (fillViewport === false) {
    return false;
  }
  return footer != null;
}

export function PageCanvasInset({ children }: { children: ReactNode }) {
  return <div className={pageCanvasInsetClass}>{children}</div>;
}

export function PageSectionStack({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return <div className={cn(pageSectionStackClass, className)}>{children}</div>;
}

export type PageLayoutProps = {
  title?: ReactNode;
  description?: ReactNode;
  badge?: ReactNode;
  headerActions?: ReactNode;
  controlPanel?: ReactNode;
  aside?: ReactNode;
  footer?: ReactNode;
  workspaceClassName?: string;
  headerClassName?: string;
  mainClassName?: string;
  asideClassName?: string;
  footerClassName?: string;
  /**
   * Stretch workspace to viewport (dashboards, tall tables).
   * Default: hugs content. When omitted, any footer band enables fill.
   */
  fillViewport?: boolean;
  children: ReactNode;
};

export function PageLayout({
  title,
  description,
  badge,
  headerActions,
  controlPanel,
  aside,
  footer,
  workspaceClassName,
  headerClassName,
  mainClassName,
  asideClassName,
  footerClassName,
  fillViewport,
  children,
}: PageLayoutProps) {
  const stretchWorkspace = resolvePageFillViewport(fillViewport, footer);

  return (
    <div className="flex min-h-0 min-w-0 w-full flex-1 flex-col">
      {title != null && title !== '' ? (
        <header
          className={cn(adminSpacing.flex.pageHeader, 'border-b border-border', headerClassName)}
        >
          <div className={cn('min-w-0', adminSpacing.stack.titleBlock)}>
            <div className={adminSpacing.flex.buttonGroup}>
              <h1 className={adminTypography.pageTitle}>{title}</h1>
              {badge}
            </div>
            {description ? (
              <div
                className={cn(
                  adminTypography.bodyMuted,
                  '[&_a]:relative [&_a]:z-[1] [&_a]:text-primary [&_a:hover]:underline'
                )}
              >
                {description}
              </div>
            ) : null}
          </div>
          {headerActions ? (
            <div className={adminSpacing.flex.headerActions}>{headerActions}</div>
          ) : null}
        </header>
      ) : null}

      <div
        className={cn(
          stretchWorkspace ? pageWorkspaceFillClass : pageWorkspaceFlatClass,
          'w-full',
          workspaceClassName
        )}
      >
        {controlPanel ? <div className="min-w-0 w-full">{controlPanel}</div> : null}

        <div
          className={cn(
            adminSpacing.grid.mainAside,
            aside
              ? 'grid-cols-1 lg:grid-cols-[minmax(0,1fr)_minmax(16rem,20rem)] grid-rows-[minmax(0,1fr)]'
              : 'grid-rows-[minmax(0,1fr)]'
          )}
        >
          <main
            className={cn(
              'flex min-h-0 min-w-0 flex-1 flex-col',
              adminSpacing.gap.lg,
              mainClassName
            )}
          >
            {children}
          </main>
          {aside ? (
            <aside className={cn('min-w-0 lg:min-h-0', asideClassName)}>{aside}</aside>
          ) : null}
        </div>

        {footer ? (
          <footer className={cn(pageFooterFlatClass, footerClassName)}>{footer}</footer>
        ) : null}
      </div>
    </div>
  );
}
