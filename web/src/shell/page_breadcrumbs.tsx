import { Link, useLocation } from 'react-router-dom';

import { useBreadcrumbSegmentLabels } from '@/shell/breadcrumb_context';
import { buildBreadcrumbs } from '@/lib/breadcrumbs';
import { trackerNavIconForPathname } from '@/lib/tracker_nav';
import { cn } from '@/lib/utils';

export function PageBreadcrumbs({ className }: { className?: string }) {
  const { pathname } = useLocation();
  const segmentLabels = useBreadcrumbSegmentLabels();
  const crumbs = buildBreadcrumbs(pathname, segmentLabels);
  const PageIcon = trackerNavIconForPathname(pathname);

  if (crumbs.length === 0) {
    return null;
  }

  const lastCrumb = crumbs[crumbs.length - 1];

  return (
    <nav aria-label="Breadcrumb" className={cn('min-w-0', className)}>
      <ol className="m-0 flex list-none items-center gap-2 p-0">
        <li className="flex min-w-0 items-center gap-2">
          <PageIcon aria-hidden className="h-4 w-4 shrink-0 text-muted-foreground" />
          {crumbs.length === 1 && lastCrumb ? (
            <span className="truncate text-[13px] font-medium text-muted-foreground">
              {lastCrumb.label}
            </span>
          ) : (
            <div className="flex min-w-0 flex-wrap items-center gap-1.5 text-sm text-muted-foreground">
              {crumbs.map((crumb, index) => {
                const isLast = index === crumbs.length - 1;
                return (
                  <span key={`${crumb.label}-${index}`} className="inline-flex items-center gap-1.5">
                    {index > 0 ? <span aria-hidden className="text-muted-foreground/60">/</span> : null}
                    {crumb.href && !isLast ? (
                      <Link className="text-muted-foreground hover:text-foreground hover:underline" to={crumb.href}>
                        {crumb.label}
                      </Link>
                    ) : isLast ? (
                      <h1
                        aria-current="page"
                        className="m-0 truncate text-base font-semibold text-foreground"
                      >
                        {crumb.label}
                      </h1>
                    ) : (
                      <span>{crumb.label}</span>
                    )}
                  </span>
                );
              })}
            </div>
          )}
        </li>
      </ol>
    </nav>
  );
}
