import { Navigate, useParams, useSearchParams } from 'react-router-dom';

import {
  isTypedCatalogReportKey,
  normalizeReportRouteSplat,
  reportTitleFromKey,
  resolveReportStubRequiresCustomer,
  resolveReportCatalogKey,
} from '@/lib/report_paths';
import { ExportOnlyReportStub } from '@/shell/export_only_report_stub';

export function ReportExportStubPage() {
  const params = useParams();
  const [searchParams] = useSearchParams();
  const splatKey = normalizeReportRouteSplat(params['*']);

  if (!splatKey) {
    return <Navigate replace to="/exports" />;
  }

  const catalogKey = resolveReportCatalogKey(splatKey);
  const catalogUnknown = !isTypedCatalogReportKey(catalogKey);
  const customerId = searchParams.get('customer_id') ?? undefined;

  return (
    <ExportOnlyReportStub
      catalogUnknown={catalogUnknown}
      customerId={customerId}
      format={searchParams.get('format') ?? undefined}
      from={searchParams.get('from') ?? undefined}
      reportKey={catalogKey}
      requiresCustomer={resolveReportStubRequiresCustomer(catalogKey)}
      title={catalogUnknown ? reportTitleFromKey(catalogKey) : undefined}
      to={searchParams.get('to') ?? undefined}
    />
  );
}
