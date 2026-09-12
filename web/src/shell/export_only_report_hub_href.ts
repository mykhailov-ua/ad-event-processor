import { buildExportHubHref } from '@/lib/export_hub_paths';
import { defaultReportRange } from '@/lib/report_paths';

export type ExportOnlyReportHubHrefParams = {
  reportKey: string;
  customerId?: string;
  from?: string;
  to?: string;
  format?: string;
  defaultRange?: string;
};

export function buildExportOnlyReportHubHref(
  params: ExportOnlyReportHubHrefParams,
  returnPath: string
): string {
  const trimmedFrom = params.from?.trim();
  const trimmedTo = params.to?.trim();
  let from = trimmedFrom || undefined;
  let to = trimmedTo || undefined;

  if (!from || !to) {
    const defaults = defaultReportRange(params.defaultRange);
    from = from ?? defaults.from;
    to = to ?? defaults.to;
  }

  return buildExportHubHref({
    kind: 'report',
    reportKey: params.reportKey,
    customerId: params.customerId?.trim() || undefined,
    from,
    to,
    format: params.format?.trim() || undefined,
    returnTo: returnPath,
  });
}
