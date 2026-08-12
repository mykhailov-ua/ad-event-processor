import type { RouteContext } from '../lib/router_types.js';
import { mountSimpleReport, type ReportColumn } from './report_simple.js';

const SPEND_VELOCITY_COLS: ReportColumn[] = [
  { key: 'bucket', label: 'Hour' },
  { key: 'spend_micro', label: 'Spend', format: 'money' },
  { key: 'clicks', label: 'Clicks', format: 'number' },
];

const DAYPART_COLS: ReportColumn[] = [
  { key: 'hour', label: 'Hour (UTC)' },
  { key: 'clicks', label: 'Clicks', format: 'number' },
];

const GEO_DEVICE_COLS: ReportColumn[] = [
  { key: 'country', label: 'Country' },
  { key: 'device', label: 'Device' },
  { key: 'clicks', label: 'Clicks', format: 'number' },
];

const SOURCE_QUALITY_COLS: ReportColumn[] = [
  { key: 'placement_id', label: 'Placement' },
  { key: 'campaign_id', label: 'Campaign' },
  { key: 'clicks', label: 'Clicks', format: 'number' },
  { key: 'conversions', label: 'Conv.', format: 'number' },
  { key: 'ivt_rate', label: 'IVT %', format: 'rate' },
  { key: 'roi_pct', label: 'ROI %', format: 'pct' },
];

const DISCREPANCY_COLS: ReportColumn[] = [
  { key: 'campaign_id', label: 'Campaign' },
  { key: 'buy_spend_micro', label: 'Buy spend', format: 'money' },
  { key: 'sell_rev_micro', label: 'Sell revenue', format: 'money' },
  { key: 'delta_micro', label: 'Delta', format: 'money' },
  { key: 'delta_pct', label: 'Delta %', format: 'pct' },
];

const TRUE_ROI_COLS: ReportColumn[] = [
  { key: 'campaign_id', label: 'Campaign' },
  { key: 'ad_spend_micro', label: 'Ad Spend', format: 'money' },
  { key: 'revenue_micro', label: 'Revenue', format: 'money' },
  { key: 'true_profit_micro', label: 'True Profit', format: 'money' },
  { key: 'true_roi_pct', label: 'True ROI %', format: 'pct' },
  { key: 'true_cpa_micro', label: 'True CPA', format: 'money' },
  { key: 'conversions', label: 'Conv.', format: 'number' },
];

const CAMPAIGN_OVERVIEW_COLS: ReportColumn[] = [
  { key: 'name', label: 'Campaign' },
  { key: 'status', label: 'Status' },
  { key: 'impressions_7d', label: 'Impr. 7d', format: 'number' },
  { key: 'clicks_7d', label: 'Clicks 7d', format: 'number' },
  { key: 'utilization_pct', label: 'Budget %', format: 'pct' },
  { key: 'pacing_drift_pct', label: 'Drift %', format: 'pct' },
  { key: 'overspend_risk', label: 'Risk' },
];

const CUSTOMER_PORTFOLIO_COLS: ReportColumn[] = [
  { key: 'row_type', label: 'Type' },
  { key: 'campaign_id', label: 'Campaign ID' },
  { key: 'name', label: 'Name' },
  { key: 'status', label: 'Status' },
  { key: 'active', label: 'Active', format: 'number' },
  { key: 'paused', label: 'Paused', format: 'number' },
  { key: 'impressions_7d', label: 'Impr. 7d', format: 'number' },
  { key: 'clicks_7d', label: 'Clicks 7d', format: 'number' },
  { key: 'utilization_pct', label: 'Budget %', format: 'pct' },
  { key: 'pacing_drift_pct', label: 'Drift %', format: 'pct' },
  { key: 'overspend_risk', label: 'Risk' },
];

/**
 * Mount spend velocity report view.
 *
 * @param {HTMLElement} container
 * @param {import('../lib/router.js').RouteContext} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mountSpendVelocity(container: HTMLElement, ctx: RouteContext) {
  return mountSimpleReport(container, ctx, {
    title: 'Spend velocity',
    endpoint: 'spend-velocity',
    columns: SPEND_VELOCITY_COLS,
  });
}

/**
 * Mount daypart heatmap report view.
 *
 * @param {HTMLElement} container
 * @param {import('../lib/router.js').RouteContext} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mountDaypartHeatmap(container: HTMLElement, ctx: RouteContext) {
  return mountSimpleReport(container, ctx, {
    title: 'Daypart heatmap',
    endpoint: 'daypart-heatmap',
    columns: DAYPART_COLS,
  });
}

/**
 * Mount campaign geo and device report view.
 *
 * @param {HTMLElement} container
 * @param {import('../lib/router.js').RouteContext} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mountCampaignGeoDevice(container: HTMLElement, ctx: RouteContext) {
  return mountSimpleReport(container, ctx, {
    title: 'Geo & device',
    endpoint: 'campaign-geo-device',
    columns: GEO_DEVICE_COLS,
  });
}

/**
 * Mount source quality report view.
 *
 * @param {HTMLElement} container
 * @param {import('../lib/router.js').RouteContext} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mountSourceQuality(container: HTMLElement, ctx: RouteContext) {
  return mountSimpleReport(container, ctx, {
    title: 'Source quality',
    endpoint: 'source-quality',
    columns: SOURCE_QUALITY_COLS,
  });
}

/**
 * Mount buy/sell discrepancy report view.
 *
 * @param {HTMLElement} container
 * @param {import('../lib/router.js').RouteContext} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mountDiscrepancyBuySell(container: HTMLElement, ctx: RouteContext) {
  return mountSimpleReport(container, ctx, {
    title: 'Buy/sell discrepancy',
    endpoint: 'discrepancy-buy-sell',
    columns: DISCREPANCY_COLS,
  });
}

/**
 * Mount True ROI report (Cost Sync Ad Spend vs revenue).
 *
 * @param {HTMLElement} container
 * @param {import('../lib/router.js').RouteContext} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mountTrueROI(container: HTMLElement, ctx: RouteContext) {
  return mountSimpleReport(container, ctx, {
    title: 'True ROI',
    endpoint: 'true-roi',
    columns: TRUE_ROI_COLS,
  });
}

/**
 * Mount campaign overview report view.
 *
 * @param {HTMLElement} container
 * @param {import('../lib/router.js').RouteContext} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mountCampaignOverview(container: HTMLElement, ctx: RouteContext) {
  return mountSimpleReport(container, ctx, {
    title: 'Campaign overview',
    endpoint: 'campaign-overview',
    columns: CAMPAIGN_OVERVIEW_COLS,
  });
}

/**
 * Mount customer portfolio report view.
 *
 * @param {HTMLElement} container
 * @param {import('../lib/router.js').RouteContext} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mountCustomerPortfolio(container: HTMLElement, ctx: RouteContext) {
  return mountSimpleReport(container, ctx, {
    title: 'Customer portfolio',
    endpoint: 'customer-portfolio',
    columns: CUSTOMER_PORTFOLIO_COLS,
  });
}
