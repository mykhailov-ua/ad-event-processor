import type { ReactNode } from 'react';

import { cn } from '@/lib/utils';

/** Flat L0 workspace on main canvas; content panels own L1 borders (tables, cards). */
export const pageWorkspaceFlatClass =
  'flex flex-col gap-3 border-0 bg-transparent p-0 dark:bg-transparent';

const pageFooterFlatClass =
  'flex shrink-0 flex-wrap items-center gap-2 border-0 border-t border-border bg-transparent p-0 pt-2 dark:bg-transparent';

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
  children,
}: PageLayoutProps) {
  return (
    <div className="flex flex-col gap-2">
      {title != null && title !== '' ? (
        <header className="flex flex-wrap items-start justify-between gap-2">
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

      <div className={cn(pageWorkspaceFlatClass, workspaceClassName)}>
        {controlPanel ? (
          <div className="relative z-[5] flex shrink-0 flex-col gap-2">{controlPanel}</div>
        ) : null}

        <div
          className={
            aside
              ? 'grid grid-cols-1 gap-2 lg:grid-cols-[minmax(0,1fr)_minmax(16rem,22rem)]'
              : 'grid grid-cols-1 gap-2'
          }
        >
          <main className={cn('flex min-w-0 flex-col gap-2', mainClassName)}>{children}</main>
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
