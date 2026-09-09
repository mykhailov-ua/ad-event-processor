import { Link, useLocation } from 'react-router-dom';

import { useBreadcrumbSegmentLabels } from '@/shell/breadcrumb_context';
import { buildBreadcrumbs } from '@/lib/breadcrumbs';
import { trackerNavIconForPathname } from '@/lib/tracker_nav';
import { cn } from '@/lib/utils';

export function PageBreadcrumbs({ }: {}) {
  const { pathname } = useLocation();
  const segmentLabels = useBreadcrumbSegmentLabels();
  const crumbs = buildBreadcrumbs(pathname, segmentLabels);
  const PageIcon = trackerNavIconForPathname(pathname);

  if (crumbs.length === 0) {
    return null;
  }

  const lastCrumb = crumbs[crumbs.length - 1];

  return (
    <nav aria-label="Breadcrumb">
      <ol >
        <li >
          <PageIcon aria-hidden  />
          {crumbs.length === 1 && lastCrumb ? (
            <span >
              {lastCrumb.label}
            </span>
          ) : (
            <div >
              {crumbs.map((crumb, index) => {
                const isLast = index === crumbs.length - 1;
                return (
                  <span
                    key={`${crumb.label}-${index}`}
                   
                  >
                    {index > 0 ? (
                      <span aria-hidden >
                        /
                      </span>
                    ) : null}
                    {crumb.href && !isLast ? (
                      <Link
                       
                        to={crumb.href}
                      >
                        {crumb.label}
                      </Link>
                    ) : isLast ? (
                      <h1
                        aria-current="page"
                       
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
