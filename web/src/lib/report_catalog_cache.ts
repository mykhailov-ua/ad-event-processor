import { getReportCatalog } from '@/api/reports_api';
import type { ReportCatalogResponse } from '@/api/types';

let cachedCatalog: ReportCatalogResponse | undefined;
let inflightCatalog: Promise<ReportCatalogResponse> | undefined;

/**
 * Fetches the report catalog once per browser session; subsequent calls reuse the cached snapshot.
 */
export function fetchReportCatalogCached(signal?: AbortSignal): Promise<ReportCatalogResponse> {
  if (cachedCatalog) {
    return Promise.resolve(cachedCatalog);
  }

  if (inflightCatalog) {
    return inflightCatalog;
  }

  inflightCatalog = getReportCatalog(signal)
    .then((response) => {
      cachedCatalog = response;
      return response;
    })
    .finally(() => {
      inflightCatalog = undefined;
    });

  return inflightCatalog;
}
