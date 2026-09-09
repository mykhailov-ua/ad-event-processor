import { Navigate, useLocation, useParams } from 'react-router-dom';

import { resolveReportCatalogKey } from '@/lib/report_paths';

function buildExportsHref(locationSearch: string, reportKey?: string): string {
  const params = new URLSearchParams(locationSearch);
  if (reportKey) {
    params.set('kind', 'report');
    params.set('report_key', resolveReportCatalogKey(reportKey));
  }
  const query = params.toString();
  return query ? `/exports?${query}` : '/exports';
}

function normalizeReportSplat(splat: string | undefined): string | undefined {
  const trimmed = splat?.trim();
  if (!trimmed) {
    return undefined;
  }
  const decoded = decodeURIComponent(trimmed).replace(/^\/+|\/+$/g, '');
  return decoded || undefined;
}

export function ReportLegacyRedirect() {
  const location = useLocation();
  const params = useParams();
  const reportKey = normalizeReportSplat(params['*']);

  return <Navigate replace to={buildExportsHref(location.search, reportKey)} />;
}

export function ReportIndexLegacyRedirect() {
  const location = useLocation();

  return <Navigate replace to={buildExportsHref(location.search)} />;
}
