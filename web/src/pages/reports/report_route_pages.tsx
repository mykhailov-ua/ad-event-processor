import { ReportQueryPage } from './report_query_page.js';
import {
  CustomerRangeReportPage,
  GEO_REPORT_COLUMNS,
  IVT_REPORT_COLUMNS,
  TRAFFIC_REPORT_COLUMNS,
} from './customer_range_report_page.js';
import { SimpleReportPage } from './simple_report_page.js';
import { SIMPLE_REPORT_CONFIGS } from './report_configs.js';

export function PlacementsReportPage() {
  return <ReportQueryPage endpoint="placements" title="Placements" />;
}

export function KeywordsReportPage() {
  return <ReportQueryPage endpoint="keywords" title="Keywords" />;
}

export function IvtBySourceReportPage() {
  return (
    <CustomerRangeReportPage
      title="IVT by source"
      endpoint="ivt-by-source"
      urlPath="/reports/ivt-by-source"
      columns={IVT_REPORT_COLUMNS}
    />
  );
}

export function TrafficSourcesReportPage() {
  return (
    <CustomerRangeReportPage
      title="Traffic sources"
      endpoint="traffic-sources"
      urlPath="/reports/traffic-sources"
      columns={TRAFFIC_REPORT_COLUMNS}
    />
  );
}

export function GeoRoiReportPage() {
  return (
    <CustomerRangeReportPage
      title="Geo ROI"
      endpoint="geo-roi"
      urlPath="/reports/geo-roi"
      columns={GEO_REPORT_COLUMNS}
    />
  );
}

export function SimpleReportRoutePage({ configKey }: { configKey: string }) {
  const config = SIMPLE_REPORT_CONFIGS.find((c) => c.endpoint === configKey || c.path.endsWith(configKey));
  if (!config) return null;
  return (
    <SimpleReportPage
      title={config.title}
      endpoint={config.endpoint}
      columns={config.columns}
    />
  );
}
