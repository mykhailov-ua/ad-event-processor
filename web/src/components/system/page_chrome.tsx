import type { ReactNode } from 'react';

export type PageChromeProps = {
  title: ReactNode;
  description?: ReactNode;
  badge?: ReactNode;
  actions?: ReactNode;
  children?: ReactNode;
};

export function PageChrome({ title, description, badge, actions, children }: PageChromeProps) {
  return (
    <section className="grid min-w-0 max-w-full gap-6">
      <header className="flex flex-wrap items-start gap-4">
        <div className="min-w-0 flex-1 grid gap-1.5">
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="text-xl font-semibold tracking-tight">{title}</h1>
            {badge}
          </div>
          {description ? (
            <p className="text-sm text-muted-foreground">{description}</p>
          ) : null}
        </div>
        {actions ? (
          <div className="ml-auto flex flex-wrap items-center gap-2 [&_button]:rounded-full">
            {actions}
          </div>
        ) : null}
      </header>
      {children}
    </section>
  );
}
