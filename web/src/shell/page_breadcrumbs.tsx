import { Link, useLocation } from 'react-router-dom';

import { useBreadcrumbSegmentLabels } from '@/shell/breadcrumb_context';
import { buildBreadcrumbs } from '@/lib/breadcrumbs';
import { cn } from '@/lib/utils';

export function PageBreadcrumbs({ className }: { className?: string }) {
  const { pathname } = useLocation();
  const segmentLabels = useBreadcrumbSegmentLabels();
  const crumbs = buildBreadcrumbs(pathname, segmentLabels);

  if (crumbs.length === 0) {
    return null;
  }

  return (
    <nav aria-label="Breadcrumb" className={cn(className)}>
      <ol className="admin-toolbar admin-muted" style={{ fontSize: '12px' }}>
        {crumbs.map((crumb, index) => {
          const isLast = index === crumbs.length - 1;

          return (
            <li key={`${crumb.label}-${index}`} className="inline-flex items-center gap-1">
              {index > 0 ? <span aria-hidden>/</span> : null}
              {crumb.href && !isLast ? (
                <Link to={crumb.href}>{crumb.label}</Link>
              ) : (
                <span aria-current={isLast ? 'page' : undefined}>{crumb.label}</span>
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}
