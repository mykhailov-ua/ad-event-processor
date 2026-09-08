import { useMemo } from 'react';
import { Link } from 'react-router-dom';

import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { HubLinkCard, HubLinkGrid } from '@/shell/hub_link_card';
import { PageChrome } from '@/shell/page_chrome';
import { MetaLinksBand } from '@/shell/ui_bands';
import { PageSkeleton } from '@/shell/page_skeleton';
import type { ReportCatalogRow } from '@/api/types';
import { reportHubPath } from '@/lib/report_paths';

export type ReportsHubProps = {
  rows: ReportCatalogRow[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

const CATEGORY_ORDER = [
  'traffic',
  'fraud',
  'billing',
  'telegram',
  'rtb',
  'ops',
] as const;

function categoryLabel(category: string): string {
  if (!category) {
    return 'Other';
  }
  return category.charAt(0).toUpperCase() + category.slice(1);
}

function groupRowsByCategory(rows: ReportCatalogRow[]): Array<{ category: string; rows: ReportCatalogRow[] }> {
  const buckets = new Map<string, ReportCatalogRow[]>();
  for (const row of rows) {
    const category = row.category?.trim() || 'other';
    const list = buckets.get(category) ?? [];
    list.push(row);
    buckets.set(category, list);
  }

  const ordered: Array<{ category: string; rows: ReportCatalogRow[] }> = [];
  for (const category of CATEGORY_ORDER) {
    const list = buckets.get(category);
    if (list?.length) {
      ordered.push({ category, rows: list });
      buckets.delete(category);
    }
  }
  for (const [category, list] of [...buckets.entries()].sort(([a], [b]) => a.localeCompare(b))) {
    if (list.length > 0) {
      ordered.push({ category, rows: list });
    }
  }
  return ordered;
}

export function ReportsHub({ rows, fetching, error, hasSnapshot }: ReportsHubProps) {
  const sections = useMemo(() => groupRowsByCategory(rows), [rows]);

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load report catalog" message={error.message} />;
  }

  return (
    <PageChrome title="Reports">
      <MetaLinksBand>
        <Link to="/reports/jobs">Export jobs</Link>
        <Link to="/reports/click-log">Click log</Link>
      </MetaLinksBand>
      {rows.length === 0 ? (
        <EmptyState
          title="No reports available"
          description="Your role may not have access to any report definitions."
        />
      ) : (
        <div className="grid gap-6">
          {sections.map((section) => (
            <section key={section.category} className="grid gap-3">
              <h2 className="text-sm font-medium text-foreground">{categoryLabel(section.category)}</h2>
              <HubLinkGrid className="sm:grid-cols-[repeat(auto-fit,minmax(300px,1fr))]">
                {section.rows.map((row) => {
                  const key = row.key ?? row.title ?? 'unknown';
                  const path = reportHubPath(key);
                  const meta = [
                    row.license_gated ? 'license' : null,
                    row.default_range ? `range: ${row.default_range}` : null,
                    row.export_formats?.length ? `export: ${row.export_formats.join(', ')}` : null,
                  ]
                    .filter(Boolean)
                    .join('  /  ');

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
            </section>
          ))}
        </div>
      )}

      {error && hasSnapshot && <ErrorBlock title="Refresh failed" message={error.message} />}
    </PageChrome>
  );
}
