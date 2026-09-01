import { Link } from 'react-router-dom';

import { EmptyState } from '@/components/system/empty_state';
import { ErrorBlock } from '@/components/system/error_block';
import { HubLinkCard, HubLinkGrid } from '@/components/system/hub_link_card';
import { PageChrome } from '@/components/system/page_chrome';
import { PageSkeleton } from '@/components/system/page_skeleton';
import type { ReportCatalogRow } from '@/api/types';

export type ReportsHubProps = {
  rows: ReportCatalogRow[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

export function ReportsHub({ rows, fetching, error, hasSnapshot }: ReportsHubProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load report catalog" message={error.message} />;
  }

  return (
    <PageChrome title="Reports">
      <div className="text-sm text-muted-foreground">
        <Link className="hover:text-foreground hover:underline" to="/reports/jobs">
          Export jobs
        </Link>
      </div>
      {rows.length === 0 ? (
        <EmptyState
          title="No reports available"
          description="Your role may not have access to any report definitions."
        />
      ) : (
        <HubLinkGrid className="sm:grid-cols-[repeat(auto-fit,minmax(300px,1fr))]">
          {rows.map((row) => {
            const key = row.key ?? row.title ?? 'unknown';
            const path = `/reports/${encodeURIComponent(key)}`;
            const meta = [
              row.category ? row.category : null,
              row.license_gated ? 'license' : null,
              row.default_range ? `range: ${row.default_range}` : null,
              row.export_formats?.length ? `export: ${row.export_formats.join(', ')}` : null,
            ]
              .filter(Boolean)
              .join(' · ');

            return (
              <HubLinkCard
                key={key}
                description={row.description ?? (meta || 'Open report runner')}
                path={path}
                title={row.title ?? key}
              />
            );
          })}
        </HubLinkGrid>
      )}

      {error && hasSnapshot && (
        <ErrorBlock title="Refresh failed" message={error.message} />
      )}
    </PageChrome>
  );
}
