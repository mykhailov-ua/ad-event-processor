const EXPORT_ONLY_STUB_MESSAGE =
  'This report is async export only. Control Plane does not render an in-browser table. Open Export Hub to configure filters and enqueue a job.';

const EXPORT_ONLY_STUB_UNKNOWN_CATALOG_PREFIX =
  'This report key is not listed in the static catalog. Export Hub still accepts report_key query parameters. ';

export function exportOnlyReportStubBannerMessage(catalogUnknown?: boolean): string {
  if (catalogUnknown) {
    return `${EXPORT_ONLY_STUB_UNKNOWN_CATALOG_PREFIX}${EXPORT_ONLY_STUB_MESSAGE}`;
  }
  return EXPORT_ONLY_STUB_MESSAGE;
}
