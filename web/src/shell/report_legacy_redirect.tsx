import { Navigate, useLocation, useParams } from 'react-router-dom';

import { normalizeReportRouteSplat, resolveReportCatalogKey } from '@/lib/report_paths';

function buildExportsHref(locationSearch: string, reportKey?: string): string {
  const params = new URLSearchParams(locationSearch);
  if (reportKey) {
    params.set('kind', 'report');
    params.set('report_key', resolveReportCatalogKey(reportKey));
  }
  const query = params.toString();
  return query ? `/exports?${query}` : '/exports';
}

export function ReportLegacyRedirect() {
  const location = useLocation();
  const params = useParams();
  const reportKey = normalizeReportRouteSplat(params['*']);

  return <Navigate replace to={buildExportsHref(location.search, reportKey)} />;
}

export function ReportIndexLegacyRedirect() {
  const location = useLocation();

  return <Navigate replace to={buildExportsHref(location.search)} />;
}
