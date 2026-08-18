import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';

export type BreadcrumbItem = {
  label: string;

  to?: string;
};

export type PageHeaderProps = {
  title: string;

  desc?: string;

  breadcrumbs?: BreadcrumbItem[];

  actions?: ReactNode;

  badge?: ReactNode;

  compact?: boolean;
};

export function PageHeader({ title, desc, breadcrumbs, actions, badge, compact }: PageHeaderProps) {
  return (
    <div className={`page-header${compact ? ' page-header--compact' : ''}`}>
      {breadcrumbs && breadcrumbs.length > 0 ? (
        <nav className="page-header__breadcrumbs" aria-label="Breadcrumbs">
          {breadcrumbs.map((crumb, i) => {
            const isLast = i === breadcrumbs.length - 1;
            return (
              <span key={crumb.label} className="page-header__crumb">
                {i > 0 ? (
                  <span className="page-header__crumb-sep" aria-hidden="true">
                    /
                  </span>
                ) : null}
                {crumb.to && !isLast ? (
                  <Link to={crumb.to} className="page-header__crumb-link">
                    {crumb.label}
                  </Link>
                ) : (
                  <span
                    className={isLast ? 'page-header__crumb-current' : 'page-header__crumb-link'}
                  >
                    {crumb.label}
                  </span>
                )}
              </span>
            );
          })}
        </nav>
      ) : null}

      <div className="page-header__row">
        <h1 className="page-header__title">
          {title}
          {badge ? <> {badge}</> : null}
        </h1>
        {actions ? <div className="page-header__actions">{actions}</div> : null}
      </div>

      {desc ? <p className="page-header__desc">{desc}</p> : null}
    </div>
  );
}
