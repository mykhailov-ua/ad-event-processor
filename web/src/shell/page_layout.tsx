import type { ReactNode } from 'react';

import { cn } from '@/lib/utils';

/** Canonical inset for route outlet and full-page fallbacks (skeleton, blocking error). */
export const pageCanvasInsetClass = 'flex w-full min-w-0 flex-col p-4';

/** Min height for directory/dashboard pages inside the scroll canvas (see app.css vars). */
export const pageFillViewportMinHeightClass = 'min-h-[var(--page-fill-min-height)]';

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

export function PageCanvasInset({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return <div className={cn(pageCanvasInsetClass, className)}>{children}</div>;
}

/** Flat L0 workspace; hugs content (sparse pages keep canvas bottom padding). */
export const pageWorkspaceFlatClass = 'flex min-w-0 flex-col gap-3';

/** Fill viewport below page header for directory tables and dashboards. */
export const pageWorkspaceFillClass = 'flex min-h-0 min-w-0 flex-1 flex-col gap-3';

/** Semantic page bands: thin dividers with compact vertical padding between direct children. */
export const pageSectionStackClass =
  'flex min-w-0 flex-col [&>*]:pb-3 [&>*:last-child]:pb-0 [&>*+*]:border-t [&>*+*]:border-border [&>*+*]:pt-3';

export function PageSectionStack({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return <div className={cn(pageSectionStackClass, className)}>{children}</div>;
}

const pageFooterFlatClass =
  'flex shrink-0 flex-wrap items-center gap-3 border-0 border-t border-border bg-transparent pt-3';

export type PageLayoutProps = {
  title?: ReactNode;
  description?: ReactNode;
  badge?: ReactNode;
  headerActions?: ReactNode;
  controlPanel?: ReactNode;
  aside?: ReactNode;
  footer?: ReactNode;
  workspaceClassName?: string;
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
  mainClassName,
  asideClassName,
  footerClassName,
  fillViewport,
  children,
}: PageLayoutProps) {
  const stretchWorkspace = resolvePageFillViewport(fillViewport, footer);

  return (
    <div
      className={cn(
        'flex w-full min-w-0 flex-col gap-3',
        stretchWorkspace && pageFillViewportMinHeightClass,
        stretchWorkspace && 'min-h-0 flex-1'
      )}
    >
      {title != null && title !== '' ? (
        <header className="grid grid-cols-[1fr_auto] items-start gap-2">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <h1>{title}</h1>
            {badge}
            {description ? <span className="text-muted-foreground">{description}</span> : null}
          </div>
          {headerActions ? (
            <div className="flex flex-wrap items-center gap-2">{headerActions}</div>
          ) : null}
        </header>
      ) : null}

      <div
        className={cn(
          stretchWorkspace ? pageWorkspaceFillClass : pageWorkspaceFlatClass,
          workspaceClassName
        )}
      >
        {controlPanel ? (
          <div className="relative z-[5] flex shrink-0 flex-col gap-2">{controlPanel}</div>
        ) : null}

        <div
          className={cn(
            aside
              ? 'grid min-w-0 grid-cols-1 items-start gap-2 lg:grid-cols-[minmax(0,1fr)_minmax(16rem,22rem)]'
              : 'grid min-w-0 grid-cols-1 items-start gap-2',
            stretchWorkspace && 'min-h-0 flex-1'
          )}
        >
          <main
            className={cn(
              pageSectionStackClass,
              'w-full min-w-0',
              stretchWorkspace && 'min-h-0 flex-1',
              mainClassName
            )}
          >
            {children}
          </main>
          {aside ? (
            <aside className={cn('flex min-w-0 flex-col gap-2 self-start', asideClassName)}>
              {aside}
            </aside>
          ) : null}
        </div>

        {footer ? (
          <footer className={cn('relative z-[5]', pageFooterFlatClass, footerClassName)}>
            {footer}
          </footer>
        ) : null}
      </div>
    </div>
  );
}
