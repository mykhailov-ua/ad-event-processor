import type { ReactNode } from 'react';

import { cn } from '@/lib/utils';

export type PageLayoutProps = {
  title?: ReactNode;
  description?: ReactNode;
  badge?: ReactNode;
  headerActions?: ReactNode;
  controlPanel?: ReactNode;
  aside?: ReactNode;
  footer?: ReactNode;
  workspaceClassName?: string;
  footerClassName?: string;
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
  footerClassName,
  children,
}: PageLayoutProps) {
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-2">
      {title != null && title !== '' ? (
        <header className="flex flex-wrap items-start justify-between gap-2">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <h1>{title}</h1>
            {badge}
            {description ? <span className="text-muted-foreground">{description}</span> : null}
          </div>
          {headerActions ? <div className="flex flex-wrap items-center gap-2">{headerActions}</div> : null}
        </header>
      ) : null}

      <div
        className={cn(
          'flex min-h-0 flex-1 flex-col gap-2 rounded-md border border-border bg-card p-2 text-card-foreground',
          workspaceClassName,
        )}
      >
        {controlPanel ? (
          <div className="relative z-[5] flex shrink-0 flex-col gap-2">{controlPanel}</div>
        ) : null}

        <div
          className={
            aside
              ? 'grid min-h-0 flex-1 grid-cols-1 gap-2 lg:grid-cols-[minmax(0,1fr)_minmax(16rem,22rem)]'
              : 'grid min-h-0 flex-1 grid-cols-1 gap-2'
          }
        >
          <main className="ui-scrollbar flex min-h-0 min-w-0 flex-1 flex-col gap-2 overflow-x-hidden overflow-y-auto">
            {children}
          </main>
          {aside ? <aside className="flex min-h-0 min-w-0 flex-col gap-2 overflow-auto">{aside}</aside> : null}
        </div>

        {footer ? (
          <footer
            className={cn(
              'relative z-[5] flex shrink-0 flex-wrap items-center gap-2 rounded-md border border-border bg-card p-2 text-card-foreground',
              footerClassName,
            )}
          >
            {footer}
          </footer>
        ) : null}
      </div>
    </div>
  );
}
