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
    <div className="admin-page-layout">
      <header className="admin-page-header">
        <div className="admin-page-header-main">
          <h1>{title}</h1>
          {badge}
          {description ? <span className="admin-muted">{description}</span> : null}
        </div>
        {headerActions ? <div className="admin-header-actions">{headerActions}</div> : null}
      </header>

      <div className="admin-page-workspace">
        {controlPanel ? <div className="admin-control-panel">{controlPanel}</div> : null}

        <div className={aside ? 'admin-page-body admin-page-body--with-aside' : 'admin-page-body'}>
          <main className="admin-page-main">{children}</main>
          {aside ? <aside className="admin-page-aside">{aside}</aside> : null}
        </div>

        {footer ? <footer className="admin-page-footer">{footer}</footer> : null}
      </div>
    </div>
  );
}
