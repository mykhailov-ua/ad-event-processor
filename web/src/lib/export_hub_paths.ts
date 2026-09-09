import { readExportHubReturnPath } from '@/lib/export_hub_return';

export type ExportHubHrefParams = {
  entry?: string;
  reportKey?: string;
  customerId?: string;
  from?: string;
  to?: string;
  format?: string;
  jobId?: string;
  kind?: 'report' | 'billing' | 'audit' | string;
  rowLimit?: number;
  /** SPA path (including query) to restore when leaving Export hub. */
  returnTo?: string;
};

const EXPORT_HUB_RETURN_TO_PARAM = 'return_to';

function normalizeExportHubReturnTo(value: string | undefined): string | undefined {
  const trimmed = value?.trim();
  if (!trimmed) {
    return undefined;
  }
  if (!trimmed.startsWith('/') || trimmed.startsWith('//')) {
    return undefined;
  }
  return trimmed;
}

/** Safe in-app return target from Export hub query (defaults to bare campaigns list). */
export function resolveExportHubReturnHref(
  searchParams: URLSearchParams,
  fallback = '/campaigns'
): string {
  const fromQuery = normalizeExportHubReturnTo(
    searchParams.get(EXPORT_HUB_RETURN_TO_PARAM) ?? undefined
  );
  if (fromQuery) {
    return fromQuery;
  }
  return readExportHubReturnPath(fallback);
}

export function buildExportHubHref(params: ExportHubHrefParams = {}): string {
  const search = new URLSearchParams();
  if (params.entry?.trim()) {
    search.set('entry', params.entry.trim());
  }
  if (params.kind?.trim()) {
    search.set('kind', params.kind.trim());
  }
  if (params.reportKey?.trim()) {
    search.set('report_key', params.reportKey.trim());
  }
  if (params.customerId?.trim()) {
    search.set('customer_id', params.customerId.trim());
  }
  if (params.from?.trim()) {
    search.set('from', params.from.trim());
  }
  if (params.to?.trim()) {
    search.set('to', params.to.trim());
  }
  if (params.format?.trim()) {
    search.set('format', params.format.trim());
  }
  if (params.jobId?.trim()) {
    search.set('job_id', params.jobId.trim());
  }
  if (params.rowLimit != null && Number.isFinite(params.rowLimit)) {
    search.set('row_limit', String(Math.trunc(params.rowLimit)));
  }
  const returnTo = normalizeExportHubReturnTo(params.returnTo);
  if (returnTo) {
    search.set(EXPORT_HUB_RETURN_TO_PARAM, returnTo);
  }
  const query = search.toString();
  return query ? `/exports?${query}` : '/exports';
}
