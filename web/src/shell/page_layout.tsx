import type { ReactNode } from 'react';

export type PageLayoutProps = {
  title: ReactNode;
  description?: ReactNode;
  badge?: ReactNode;
  headerActions?: ReactNode;
  controlPanel?: ReactNode;
  aside?: ReactNode;
  footer?: ReactNode;
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
  children,
}: PageLayoutProps) {
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-2">
      <header className="flex flex-wrap items-start justify-between gap-2">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <h1>{title}</h1>
          {badge}
          {description ? <span className="text-zinc-500 dark:text-zinc-400">{description}</span> : null}
        </div>
        {headerActions ? <div className="flex flex-wrap items-center gap-2">{headerActions}</div> : null}
      </header>

      <div className="flex min-h-0 flex-1 flex-col gap-2 rounded-md border border-zinc-200 bg-white p-2 dark:border-zinc-800 dark:bg-zinc-950">
        {controlPanel ? <div className="relative z-[5] flex flex-col gap-2">{controlPanel}</div> : null}

        <div
          className={
            aside
              ? 'grid min-h-0 flex-1 grid-cols-1 gap-2 lg:grid-cols-[minmax(0,1fr)_minmax(16rem,22rem)]'
              : 'grid min-h-0 flex-1 grid-cols-1 gap-2'
          }
        >
          <main className="flex min-h-0 min-w-0 flex-1 flex-col gap-2 overflow-hidden">{children}</main>
          {aside ? <aside className="flex min-h-0 min-w-0 flex-col gap-2 overflow-auto">{aside}</aside> : null}
        </div>

        {footer ? <footer className="relative z-[5] flex flex-wrap items-center gap-2 rounded-md border border-zinc-200 bg-white p-2 dark:border-zinc-800 dark:bg-zinc-950">{footer}</footer> : null}
      </div>
    </div>
  );
}
