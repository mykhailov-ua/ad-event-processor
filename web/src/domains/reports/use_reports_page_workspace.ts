// reports hub: single catalog fetch per session; report runner pages fetch their own data.
import { useResource } from '@/api/use_resource';
import { fetchReportCatalogCached } from '@/lib/report_catalog_cache';

export function useReportsPageWorkspace() {
  const { data, error, fetching } = useResource((signal) => fetchReportCatalogCached(signal), []);

  return {
    rows: data?.rows ?? [],
    fetching,
    error,
    hasSnapshot: data != null,
  };
}
