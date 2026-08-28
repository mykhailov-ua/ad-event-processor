import { useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import type { SavedView } from '../helpers/report_api.js';
import { useResource } from '../helpers/use_resource.js';
import { ReportsHub } from '../ui/reports/reports_hub.js';

export function ReportHubPage() {
  const user = auth.getUser();
  const customerId = user?.customer_id;
  const [searchParams] = useSearchParams();
  const scopedCustomerId = searchParams.get('customer_id') ?? customerId ?? undefined;

  const {
    data: catalog,
    loading: catalogLoading,
    error: catalogError,
  } = useResource<{ rows: Array<{ key?: string; title?: string; description?: string; category?: string }> }>(
    '/api/v1/reports/catalog'
  );

  const savedViewsUrl = scopedCustomerId
    ? `/api/v1/views?customer_id=${encodeURIComponent(scopedCustomerId)}`
    : null;

  const {
    data: savedViewsPayload,
    loading: savedViewsLoading,
    error: savedViewsError,
  } = useResource<SavedView[]>(savedViewsUrl, {
    skip: !savedViewsUrl,
  });

  const savedViews = Array.isArray(savedViewsPayload) ? savedViewsPayload : [];

  const catalogRows = catalog?.rows ?? [];

  return (
    <ReportsHub
      catalogRows={catalogRows}
      savedViews={savedViews}
      loading={catalogLoading}
      savedViewsLoading={savedViewsLoading}
      error={catalogError}
      savedViewsError={savedViewsError}
    />
  );
}
