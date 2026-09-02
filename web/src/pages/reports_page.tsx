import { ReportsHub } from '@/domains/reports/reports_hub';
import { useResource } from '@/api/use_resource';
import { fetchReportCatalogCached } from '@/lib/report_catalog_cache';

export function ReportsPage() {
  const { data, error, fetching } = useResource(
    (signal) => fetchReportCatalogCached(signal),
    [],
  );

  return (
    <ReportsHub
      rows={data?.rows ?? []}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
    />
  );
}
