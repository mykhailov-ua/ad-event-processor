import type { ReportCatalogRow } from '@/api/types';

export type ExportHubReportFormat = 'csv' | 'json' | 'xlsx' | 'zip';

const EXPORT_HUB_REPORT_FORMATS: ExportHubReportFormat[] = ['csv', 'xlsx', 'json'];

export function resolveExportHubReportFormats(
  reportKey: string,
  catalogRow?: ReportCatalogRow
): ExportHubReportFormat[] {
  if (reportKey === 'fraud-evidence-pack-bulk') {
    return ['zip'];
  }
  const catalogFormats = catalogRow?.export_formats
    ?.map((value) => value.trim())
    .filter(
      (value): value is ExportHubReportFormat =>
        value === 'csv' || value === 'json' || value === 'xlsx' || value === 'zip'
    );
  if (catalogFormats && catalogFormats.length > 0) {
    return catalogFormats;
  }
  return EXPORT_HUB_REPORT_FORMATS;
}
