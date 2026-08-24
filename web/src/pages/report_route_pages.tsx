import { ReportQueryPage } from './report_query_page.js';
import {
  CustomerRangeReportPage,
  DATA_QUALITY_REPORT_COLUMNS,
  COST_SYNC_COVERAGE_REPORT_COLUMNS,
  FILTER_REJECT_REPORT_COLUMNS,
  FRAUD_BREAKDOWN_REPORT_COLUMNS,
  GHOST_IMPRESSION_FUNNEL_COLUMNS,
  GEO_REPORT_COLUMNS,
  IVT_REPORT_COLUMNS,
  PACING_DRIFT_REPORT_COLUMNS,
  POSTBACK_RECON_REPORT_COLUMNS,
  RTB_NO_BID_REPORT_COLUMNS,
  RTB_OVERVIEW_REPORT_COLUMNS,
  RTB_GEO_DEVICE_REPORT_COLUMNS,
  TRAFFIC_REPORT_COLUMNS,
} from './report_customer_range_page.js';
import { SimpleReportPage } from './report_simple_page.js';
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

export function DataQualityReportPage() {
  return (
    <CustomerRangeReportPage
      title="Data quality"
      endpoint="data-quality"
      urlPath="/reports/data-quality"
      columns={DATA_QUALITY_REPORT_COLUMNS}
      enableCompare={false}
      enableActions={false}
    />
  );
}

export function CostSyncCoverageReportPage() {
  return (
    <CustomerRangeReportPage
      title="Cost sync coverage"
      endpoint="cost-sync-coverage"
      urlPath="/reports/cost-sync-coverage"
      columns={COST_SYNC_COVERAGE_REPORT_COLUMNS}
      enableCompare={false}
      enableActions={false}
    />
  );
}

export function FilterRejectsReportPage() {
  return (
    <CustomerRangeReportPage
      title="Filter rejects"
      endpoint="filter-rejects"
      urlPath="/reports/filter-rejects"
      columns={FILTER_REJECT_REPORT_COLUMNS}
      enableCompare={false}
      enableActions={false}
      requireCustomer={false}
    />
  );
}

export function FraudBreakdownReportPage() {
  return (
    <CustomerRangeReportPage
      title="Fraud breakdown"
      endpoint="fraud-breakdown"
      urlPath="/reports/fraud-breakdown"
      columns={FRAUD_BREAKDOWN_REPORT_COLUMNS}
      enableCompare={false}
      enableActions={false}
    />
  );
}

export function GhostImpressionFunnelReportPage() {
  return (
    <CustomerRangeReportPage
      title="Ghost impression funnel"
      endpoint="ghost-impression-funnel"
      urlPath="/reports/ghost-impression-funnel"
      columns={GHOST_IMPRESSION_FUNNEL_COLUMNS}
      enableCompare={false}
      enableActions={false}
    />
  );
}

export function RtbOverviewReportPage() {
  return (
    <CustomerRangeReportPage
      title="RTB overview"
      endpoint="rtb/overview"
      urlPath="/reports/rtb/overview"
      columns={RTB_OVERVIEW_REPORT_COLUMNS}
      enableCompare={false}
      enableActions={false}
      requireCustomer={false}
    />
  );
}

export function RtbNoBidReasonsReportPage() {
  return (
    <CustomerRangeReportPage
      title="RTB no-bid reasons"
      endpoint="rtb/no-bid-reasons"
      urlPath="/reports/rtb/no-bid-reasons"
      columns={RTB_NO_BID_REPORT_COLUMNS}
      enableCompare={false}
      enableActions={false}
      requireCustomer={false}
    />
  );
}

export function RtbGeoDeviceReportPage() {
  return (
    <CustomerRangeReportPage
      title="RTB geo & device"
      endpoint="rtb/geo-device"
      urlPath="/reports/rtb/geo-device"
      columns={RTB_GEO_DEVICE_REPORT_COLUMNS}
      enableCompare={false}
      enableActions={false}
      requireCustomer={false}
    />
  );
}

export function PostbackReconReportPage() {
  return (
    <CustomerRangeReportPage
      title="Postback reconciliation"
      endpoint="postback-reconciliation"
      urlPath="/reports/postback-reconciliation"
      columns={POSTBACK_RECON_REPORT_COLUMNS}
      enableCompare={false}
      enableActions={false}
    />
  );
}

export function PacingDriftReportPage() {
  return (
    <CustomerRangeReportPage
      title="Pacing drift"
      endpoint="pacing-drift"
      urlPath="/reports/pacing-drift"
      columns={PACING_DRIFT_REPORT_COLUMNS}
      enableCompare={false}
      enableActions={false}
    />
  );
}

export function SimpleReportRoutePage({ configKey }: { configKey: string }) {
  const config = SIMPLE_REPORT_CONFIGS.find(
    (c) => c.endpoint === configKey || c.path.endsWith(configKey)
  );
  if (!config) return null;
  return (
    <SimpleReportPage title={config.title} endpoint={config.endpoint} columns={config.columns} />
  );
}
